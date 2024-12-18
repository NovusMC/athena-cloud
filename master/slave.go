package main

import (
	"common"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"net"
	"protocol"
	"strings"
)

type slaveManager struct {
	m      *master
	slaves []*slave
}

type slave struct {
	conn          net.Conn
	m             *master
	name          string
	host          string
	authenticated bool
	memory        int32
	freeMemory    int32
}

func newSlaveManager(m *master) *slaveManager {
	return &slaveManager{m: m}
}

func (sm *slaveManager) newSlave(conn net.Conn) *slave {
	host, _ := common.SplitBindAddr(conn.RemoteAddr().String())
	slv := &slave{conn: conn, m: sm.m, host: host}
	sm.slaves = append(sm.slaves, slv)
	return slv
}

func (s *slave) handlePacketPreAuth(p proto.Message) error {
	switch p := p.(type) {
	case *protocol.PacketAuthenticate:
		if p.SecretKey != s.m.cfg.SecretKey {
			_ = s.sendPacket(&protocol.PacketAuthFailed{Message: "invalid secret key"})
			return fmt.Errorf("invalid secret key")
		}
		slv := s.m.sm.getSlave(p.SlaveName)
		if slv != nil {
			_ = s.sendPacket(&protocol.PacketAuthFailed{Message: "slave with name already exists"})
			return fmt.Errorf("slave with name %q already exists", p.SlaveName)
		}
		s.name = p.SlaveName
		s.memory = p.Memory
		s.freeMemory = p.Memory
		s.authenticated = true
		log.Printf("slave %q successfully authenticated", s.name)
		err := s.sendPacket(&protocol.PacketAuthSuccess{})
		if err != nil {
			return fmt.Errorf("failed to send auth success packet: %v", err)
		}
	default:
		return fmt.Errorf("slave not authenticated")
	}
	return nil
}

func (s *slave) handlePacket(p proto.Message) error {
	if !s.authenticated {
		return s.handlePacketPreAuth(p)
	}

	switch p := p.(type) {
	case *protocol.PacketServiceStartFailed:
		log.Printf("slave %q failed to start service %q: %s", s.name, p.ServiceName, p.Message)
		svc := s.m.sched.getService(p.ServiceName)
		if svc != nil {
			svc.State = protocol.Service_STATE_OFFLINE
			err := s.m.sched.deleteService(svc)
			if err != nil {
				log.Printf("failed to delete service %q: %v", svc.Service.Name, err)
			}
		}
	case *protocol.PacketServiceStopped:
		log.Printf("service %q on slave %q stopped", p.ServiceName, s.name)
		svc := s.m.sched.getService(p.ServiceName)
		if svc != nil {
			svc.State = protocol.Service_STATE_OFFLINE
			svc.Port = 0
			err := s.m.sched.deleteService(svc)
			if err != nil {
				fmt.Printf("failed to delete service %q: %v", svc.Service.Name, err)
			}
			if svc.Type == protocol.Service_TYPE_SERVER {
				for _, prx := range s.m.sched.services {
					if prx.Type != protocol.Service_TYPE_PROXY || prx.s == nil || prx.State != protocol.Service_STATE_ONLINE {
						continue
					}
					err = prx.sendPacket(&protocol.PacketProxyUnregisterServer{
						ServerName: svc.Name,
					})
					if err != nil {
						log.Printf("failed to send proxy unregister server packet: %v", err)
					}
				}
			}
		}
	case *protocol.PacketServiceOnline:
		svc := s.m.sched.getService(p.ServiceName)
		if svc != nil {
			svc.State = protocol.Service_STATE_ONLINE
			svc.Port = p.Port
			log.Printf("service %q on slave %q is now online", p.ServiceName, s.name)
			if svc.Type == protocol.Service_TYPE_PROXY {
				for _, srv := range s.m.sched.services {
					if srv.Type != protocol.Service_TYPE_SERVER || srv.s == nil || srv.State != protocol.Service_STATE_ONLINE {
						continue
					}
					err := svc.sendPacket(&protocol.PacketProxyRegisterServer{
						ServerName: srv.Name,
						Host:       srv.s.host,
						Port:       srv.Port,
					})
					if err != nil {
						log.Printf("failed to send proxy register server packet: %v", err)
					}
				}
			} else if svc.Type == protocol.Service_TYPE_SERVER {
				for _, prx := range s.m.sched.services {
					if prx.Type != protocol.Service_TYPE_PROXY || prx.s == nil || prx.State != protocol.Service_STATE_ONLINE {
						continue
					}
					err := prx.sendPacket(&protocol.PacketProxyRegisterServer{
						ServerName: svc.Name,
						Host:       svc.s.host,
						Port:       svc.Port,
					})
					if err != nil {
						log.Printf("failed to send proxy register server packet: %v", err)
					}
				}
			}
		}
	case *protocol.PacketScreenLine:
		if s.m.sc.svc == nil {
			return nil
		}
		lines := strings.Split(p.Line, "\n")
		log2 := log.New(log.Writer(), color.BlueString("[%s] ", s.m.sc.svc.Name), log.Flags()&^log.Ltime&^log.Ldate)
		for _, line := range lines {
			log2.Println(line)
		}
	}
	return nil
}

func (s *slave) schedule(svc *service) {
	_ = s.sendPacket(&protocol.PacketScheduleServiceRequest{
		Service: svc.Service,
		Group:   svc.g.Group,
	})
}

func (sm *slaveManager) removeSlave(slv *slave) {
	if slv.authenticated {
		log.Printf("slave %q disconnected", slv.name)
	} else {
		log.Printf("authentication with slave %q failed", slv.conn.RemoteAddr())
	}
	sm.slaves = common.DeleteItem(sm.slaves, slv)
	for _, svc := range slv.services() {
		svc.s = nil
		svc.Port = 0
		svc.State = protocol.Service_STATE_OFFLINE
		err := slv.m.sched.deleteService(svc)
		if err != nil {
			log.Printf("failed to delete service %q: %v", svc.Service.Name, err)
		}
	}
}

func (sm *slaveManager) getSlave(name string) *slave {
	for _, s := range sm.slaves {
		if s.name == name {
			return s
		}
	}
	return nil
}

func handleSlaveConnection(ch chan<- any, lis net.Listener) {
	defer recoverPanic()

	for {
		conn, err := lis.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				log.Printf("connection error: %v", err)
			}
			break
		}
		log.Printf("new connection from %s", conn.RemoteAddr())
		slvCh := make(chan *slave)
		ch <- createSlaveCmd{conn: conn, slvCh: slvCh}
		slv := <-slvCh
		go func() {
			defer recoverPanic()
			for {
				p, err := protocol.ReadPacket(conn)
				if err != nil {
					if !errors.Is(err, io.EOF) {
						log.Printf("failed reading packet: %v", err)
					}
					break
				}
				errCh := make(chan error)
				ch <- handleSlavePacketCmd{slv: slv, p: p, errCh: errCh}
				err = <-errCh
				if err != nil {
					log.Printf("failed handling packet: %v", err)
					break
				}
			}
			_ = conn.Close()
			ch <- removeSlaveCmd{slv}
		}()
	}
}

func (s *slave) services() []*service {
	var services []*service
	for _, svc := range s.m.sched.services {
		if svc.s == s {
			services = append(services, svc)
		}
	}
	return services
}

func (s *slave) sendPacket(p proto.Message) error {
	if s.conn == nil {
		return fmt.Errorf("not connected")
	}
	return protocol.SendPacket(s.conn, p)
}
