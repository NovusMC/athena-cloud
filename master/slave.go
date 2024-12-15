package main

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"log"
	"net"
	"protocol"
	"slices"
	"sync"
)

type slaveManager struct {
	m      *master
	slaves []*slave
	mu     sync.RWMutex
}

type slave struct {
	conn          net.Conn
	m             *master
	name          string
	authenticated bool
	memory        int32
	freeMemory    int32
}

func newSlaveManager(m *master) *slaveManager {
	return &slaveManager{m: m}
}

func (sm *slaveManager) newSlave(conn net.Conn) *slave {
	slv := &slave{conn: conn, m: sm.m}
	sm.mu.Lock()
	sm.slaves = append(sm.slaves, slv)
	sm.mu.Unlock()
	return slv
}

func (s *slave) handlePacketPreAuth(p proto.Message) error {
	switch p := p.(type) {
	case *protocol.PacketAuthenticate:
		if p.SecretKey != s.m.cfg.SecretKey {
			_ = protocol.SendPacket(s.conn, &protocol.PacketAuthFailed{Message: "invalid secret key"})
			return fmt.Errorf("invalid secret key")
		}
		slv := s.m.sm.getSlave(p.SlaveName)
		if slv != nil {
			_ = protocol.SendPacket(s.conn, &protocol.PacketAuthFailed{Message: "slave with name already exists"})
			return fmt.Errorf("slave with name %q already exists", p.SlaveName)
		}
		s.name = p.SlaveName
		s.memory = p.Memory
		s.freeMemory = p.Memory
		s.authenticated = true
		log.Printf("slave %q successfully authenticated", s.name)
		err := protocol.SendPacket(s.conn, &protocol.PacketAuthSuccess{})
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
			s.m.sched.deleteService(svc)
		}
	case *protocol.PacketServiceStopped:
		svc := s.m.sched.getService(p.ServiceName)
		if svc != nil {
			s.m.sched.deleteService(svc)
		}
		log.Printf("service %q on slave %q stopped", p.ServiceName, s.name)
	}
	return nil
}

func (s *slave) schedule(svc *service) {
	_ = protocol.SendPacket(s.conn, &protocol.PacketScheduleServiceRequest{
		Service: svc.Service,
		Group:   svc.g.Group,
	})
}

func (sm *slaveManager) removeSlave(s *slave) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	idx := slices.Index(sm.slaves, s)
	sm.slaves = slices.Delete(sm.slaves, idx, idx+1)
}

func (sm *slaveManager) getSlave(name string) *slave {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	for _, s := range sm.slaves {
		if s.name == name {
			return s
		}
	}
	return nil
}
