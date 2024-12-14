package main

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"common"
)

type slave struct {
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
	Memory         int    `yaml:"memory"`
}

type handler struct {
	conn net.Conn
	s    *slave
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
	conn, err := net.Dial("tcp", s.cfg.MasterAddr)
	if err != nil {
		log.Fatalf("could not connect to master: %v", err)
	}
	defer func() {
		_ = conn.Close()
	}()
	log.Println("connected to master")

	err = common.SendPacket(conn, common.PacketTypeAuthenticate, common.PacketAuthenticate{
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
			_ = conn.Close()
		}
	}()

	h := &handler{conn: conn, s: &s}
	err = common.HandleConnection(io.TeeReader(conn, os.Stdout), h)
	//err = common.HandleConnection(conn, h)
	if err != nil {
		log.Fatalf("connection error: %v", err)
	}
	log.Printf("disconnecting from master at %s", conn.RemoteAddr())
	_ = conn.Close()
}

func (h *handler) HandlePacket(packetType common.PacketType, data json.RawMessage) error {
	switch packetType {
	case common.PacketTypeAuthenticate:
		h.s.authenticated = true
		log.Println("authenticated with master")
	case common.PacketTypeAuthFailed:
		var p common.PacketAuthFailed
		err := json.Unmarshal(data, &p)
		if err != nil {
			return fmt.Errorf("error unmarshalling packet: %w", err)
		}
		return fmt.Errorf("authentication failed: %s", p.Message)
	case common.PacketTypeScheduleServiceRequest:
		var p common.PacketScheduleServiceRequest
		err := json.Unmarshal(data, &p)
		if err != nil {
			return fmt.Errorf("error unmarshalling packet: %w", err)
		}
		log.Printf("asked to schedule service %s", p.Service.Name)
		svc, err := h.s.svcm.createService(p.Service, p.Group)
		if err != nil {
			_ = common.SendPacket(h.conn, common.PacketTypeServiceStartFailed, common.PacketServiceStartFailed{
				ServiceName: p.Service.Name,
				Message:     fmt.Sprintf("failed to create service: %v", err),
			})
			log.Printf("failed to schedule service %s: %v", p.Service.Name, err)
			return nil
		}
		log.Printf("starting service %q", p.Service.Name)
		err = h.s.svcm.startService(svc)
		if err != nil {
			_ = common.SendPacket(h.conn, common.PacketTypeServiceStartFailed, common.PacketServiceStartFailed{
				ServiceName: p.Service.Name,
				Message:     fmt.Sprintf("failed to start service: %v", err),
			})
			log.Printf("failed to start service %q: %v", p.Service.Name, err)
			return nil
		}
	}
	return nil
}
