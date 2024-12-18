package main

import (
	"errors"
	"fmt"
	"github.com/fatih/color"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"net"
	"os"
	"protocol"
	"runtime/debug"
	"strings"
	"time"

	"common"
)

type slave struct {
	conn          net.Conn
	cfg           *config
	authenticated bool
	tmpl          *templateManager
	svcm          *serviceManager
	ch            chan<- any
}

type config struct {
	Name           string `yaml:"name"`
	BindAddr       string `yaml:"bind_addr"`
	MasterAddr     string `yaml:"master_addr"`
	FileServerHost string `yaml:"file_server_host"`
	FileServerPort string `yaml:"file_server_port"`
	SecretKey      string `yaml:"secret_key"`
	Memory         int32  `yaml:"memory"`
}

func main() {
	err := os.MkdirAll("logs", 0755)
	if err != nil {
		panic(err)
	}

	logFile, err := os.OpenFile("logs/slave.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logFile.Close()
	}()

	defer recoverPanic()

	log.SetOutput(io.MultiWriter(os.Stdout, common.NewStripAnsiWriter(logFile)))
	log.Println(color.YellowString(common.Header))
	log.SetPrefix(color.YellowString("[slave] "))
	log.Printf("starting Athena-Slave %s", common.Version)

	var s slave

	err = checkForRsync()
	if err != nil {
		log.Fatalf("%v", err)
	}

	ch := make(chan any)
	s.ch = ch

	s.cfg, err = common.ReadConfig("slave.yaml", config{
		Name:           "slave-01",
		BindAddr:       ":3000",
		MasterAddr:     "127.0.0.1:5000",
		FileServerHost: "127.0.0.1",
		FileServerPort: "5001",
		SecretKey:      "",
		Memory:         1024,
	})
	if err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	s.tmpl, err = newTemplateManager(&s)
	if err != nil {
		log.Fatalf("error loading templates: %v", err)
	}

	s.svcm, err = newServiceManager(&s)
	if err != nil {
		log.Fatalf("error initializing service manager: %v", err)
	}

	log.Printf("connecting to master at %s", s.cfg.MasterAddr)
	s.conn, err = net.Dial("tcp", s.cfg.MasterAddr)
	if err != nil {
		log.Fatalf("could not connect to master: %v", err)
	}
	log.Println("connected to master")

	err = s.sendPacket(&protocol.PacketAuthenticate{
		SlaveName: s.cfg.Name,
		SecretKey: s.cfg.SecretKey,
		Memory:    s.cfg.Memory,
	})
	if err != nil {
		log.Fatalf("could not authenticate with master: %v", err)
	}

	go func() {
		defer recoverPanic()
		time.Sleep(10 * time.Second)
		if !s.authenticated {
			log.Println("authentication with master timed out")
			_ = s.conn.Close()
		}
	}()

	lis, err := net.Listen("tcp", s.cfg.BindAddr)
	if err != nil {
		log.Fatalf("failed starting server: %v", err)
	}
	defer func() {
		_ = lis.Close()
	}()
	log.Printf("listening on %s", s.cfg.BindAddr)
	go handleMasterConnection(ch, s.conn)
	go handleServiceConnection(ch, lis)

	s.runCommandQueue(ch)

	log.Printf("disconnecting from master at %s", s.conn.RemoteAddr())
	_ = s.conn.Close()
}

func (s *slave) sendPacket(p proto.Message) error {
	return protocol.SendPacket(s.conn, p)
}

func (s *slave) handlePacketPreAuth(p proto.Message) error {
	switch p := p.(type) {
	case *protocol.PacketAuthSuccess:
		s.authenticated = true
		log.Println("authenticated with master")
	default:
		return fmt.Errorf("received packet before authentication: %T", p)
	}
	return nil
}

func (s *slave) handlePacket(p proto.Message) error {
	if !s.authenticated {
		return s.handlePacketPreAuth(p)
	}

	switch p := p.(type) {
	case *protocol.PacketAuthFailed:
		return fmt.Errorf("authentication failed: %s", p.Message)

	case *protocol.PacketScheduleServiceRequest:
		log.Printf("asked to schedule service %s", p.Service.Name)
		svc, err := s.svcm.createService(p.Service, p.Group)
		if err != nil {
			log.Printf("failed to schedule service %s: %v", p.Service.Name, err)
			err = s.sendPacket(&protocol.PacketServiceStartFailed{
				ServiceName: p.Service.Name,
				Message:     fmt.Sprintf("failed to create service: %v", err),
			})
			if err != nil {
				return fmt.Errorf("failed to send packet: %w", err)
			}
			return nil
		}
		log.Printf("starting service %q", p.Service.Name)
		err = s.svcm.startService(svc)
		if err != nil {
			log.Printf("failed to start service %q: %v", p.Service.Name, err)
			err = s.sendPacket(&protocol.PacketServiceStartFailed{
				ServiceName: p.Service.Name,
				Message:     fmt.Sprintf("failed to start service: %v", err),
			})
			if err != nil {
				return fmt.Errorf("failed to send packet: %w", err)
			}
			return nil
		}

	case *protocol.PacketStopService:
		svc := s.svcm.getService(p.ServiceName)
		if svc == nil {
			log.Printf("service %q not found", p.ServiceName)
			return nil
		}
		log.Printf("stopping service %q", p.ServiceName)
		err := s.svcm.stopService(svc)
		if err != nil {
			log.Printf("failed to stop service %q: %v", p.ServiceName, err)
			return nil
		}

	case *protocol.ServiceEnvelope:
		svc := s.svcm.getService(p.ServiceName)
		if svc == nil {
			log.Printf("service %q not found", p.ServiceName)
			return nil
		}
		msg, err := protocol.UnmarshalPayload(p.Payload)
		if err != nil {
			log.Printf("failed to unmarshal payload: %v", err)
			return nil
		}
		err = svc.sendPacket(msg)
		if err != nil {
			return fmt.Errorf("failed to send packet: %w", err)
		}

	case *protocol.PacketAttachScreen:
		svc := s.svcm.getService(p.ServiceName)
		if svc == nil {
			log.Printf("service %q not found", p.ServiceName)
			return nil
		}
		svc.sc.mu.Lock()
		err := s.sendPacket(&protocol.PacketScreenLine{
			Line: strings.Join(svc.sc.lines, "\n"),
		})
		svc.sc.report = true
		svc.sc.mu.Unlock()
		if err != nil {
			return fmt.Errorf("failed to send packet: %w", err)
		}

	case *protocol.PacketDetachScreen:
		svc := s.svcm.getService(p.ServiceName)
		if svc == nil {
			log.Printf("service %q not found", p.ServiceName)
			return nil
		}
		svc.sc.mu.Lock()
		svc.sc.report = false
		svc.sc.mu.Unlock()

	case *protocol.PacketExecuteServiceCommand:
		svc := s.svcm.getService(p.ServiceName)
		if svc == nil {
			log.Printf("service %q not found", p.ServiceName)
			return nil
		}
		if svc.w == nil {
			log.Printf("service %q is not writeable", p.ServiceName)
			return nil
		}
		_, err := svc.w.Write([]byte(p.Command + "\n"))
		if err != nil {
			log.Printf("failed to write to service %q: %v", p.ServiceName, err)
			return nil
		}
	}
	return nil
}

func handleMasterConnection(ch chan<- any, conn net.Conn) {
	defer recoverPanic()
	for {
		p, err := protocol.ReadPacket(conn)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("failed to read packet: %v", err)
			}
			break
		}
		errCh := make(chan error)
		ch <- handleMasterPacketCmd{p: p, errCh: errCh}
		err = <-errCh
		if err != nil {
			log.Printf("failed to handle packet: %v", err)
			break
		}
	}
	ch <- slaveDisconnectCmd{}
}

func recoverPanic() {
	if r := recover(); r != nil {
		stack := string(debug.Stack())
		linesToSkip := 4
		if strings.HasPrefix(stack, "goroutine") {
			linesToSkip++
		}
		stack = strings.Join(strings.Split(stack, "\n")[linesToSkip:], "\n")
		log.Fatalf("panic: %v\n%s", r, stack)
	}
}
