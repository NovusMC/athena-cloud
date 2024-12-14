package main

import (
	"fmt"
	"github.com/fatih/color"
	"google.golang.org/protobuf/proto"
	"log"
	"net"
	"protocol"
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
}

type config struct {
	Name           string `yaml:"name"`
	MasterAddr     string `yaml:"master_addr"`
	FileServerHost string `yaml:"file_server_host"`
	FileServerPort string `yaml:"file_server_port"`
	SecretKey      string `yaml:"secret_key"`
	Memory         int32  `yaml:"memory"`
}

func main() {
	fmt.Println(color.RedString(strings.TrimPrefix(common.Header, "\n")))
	log.SetPrefix(color.WhiteString("[slave] "))
	log.Printf("starting Athena-Slave %s", common.Version)

	var (
		err error
		s   slave
	)

	err = checkForRsync()
	if err != nil {
		log.Fatalf("%v", err)
	}

	s.cfg, err = common.ReadConfig("slave.yaml", config{
		Name:           "slave-01",
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
	defer func() {
		_ = s.conn.Close()
	}()
	log.Println("connected to master")

	err = protocol.SendPacket(s.conn, &protocol.PacketAuthenticate{
		SlaveName: s.cfg.Name,
		SecretKey: s.cfg.SecretKey,
		Memory:    s.cfg.Memory,
	})
	if err != nil {
		log.Fatalf("could not authenticate with master: %v", err)
	}

	go func() {
		time.Sleep(10 * time.Second)
		if !s.authenticated {
			log.Println("authentication with master timed out")
			_ = s.conn.Close()
		}
	}()

	for {
		p, err := protocol.ReadPacket(s.conn)
		if err != nil {
			log.Printf("failed to read packet: %v", err)
			break
		}
		err = s.handlePacket(p)
		if err != nil {
			log.Printf("failed to handle packet: %v", err)
			break
		}
	}
	log.Printf("disconnecting from master at %s", s.conn.RemoteAddr())
	_ = s.conn.Close()
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
			_ = protocol.SendPacket(s.conn, &protocol.PacketServiceStartFailed{
				ServiceName: p.Service.Name,
				Message:     fmt.Sprintf("failed to create service: %v", err),
			})
			log.Printf("failed to schedule service %s: %v", p.Service.Name, err)
			return nil
		}
		log.Printf("starting service %q", p.Service.Name)
		err = s.svcm.startService(svc)
		if err != nil {
			_ = protocol.SendPacket(s.conn, &protocol.PacketServiceStartFailed{
				ServiceName: p.Service.Name,
				Message:     fmt.Sprintf("failed to start service: %v", err),
			})
			log.Printf("failed to start service %q: %v", p.Service.Name, err)
			return nil
		}
	}
	return nil
}
