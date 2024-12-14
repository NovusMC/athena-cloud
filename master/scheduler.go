package main

import (
	"fmt"
	"log"
	"protocol"
	"slices"
	"sync"
)

type service struct {
	*protocol.Service
	g *group
	s *slave
}

type scheduler struct {
	m        *master
	services []*service
	mu       sync.RWMutex
}

func newScheduler(m *master) *scheduler {
	return &scheduler{m: m}
}

func (s *scheduler) scheduleServices() {
	s.m.gm.mu.RLock()
	for _, g := range s.m.gm.groups {
		if int32(len(g.services)) < g.MinServices {
			for i := int32(0); i < g.MinServices-int32(len(g.services)); i++ {
				s.createService(g)
			}
		}
	}
	s.m.gm.mu.RUnlock()

	s.mu.RLock()
	for _, svc := range s.services {
		s.scheduleService(svc)
	}
	s.mu.RUnlock()
}

func (s *scheduler) createService(g *group) {
	name := s.getNextServiceName(g.Name)
	svc := &service{
		Service: &protocol.Service{
			Name:   name,
			State:  protocol.Service_STATE_PENDING,
			Type:   g.Type,
			Group:  g.Name,
			Slave:  "",
			Port:   0,
			Memory: 0,
		},
		g: g,
	}
	s.mu.Lock()
	s.services = append(s.services, svc)
	s.mu.Unlock()
	g.mu.Lock()
	g.services = append(g.services, svc)
	g.mu.Unlock()
	log.Printf("service %s created", svc.Name)
}

func (s *scheduler) scheduleService(svc *service) {
	if svc.s != nil {
		return
	}

	s.m.sm.mu.RLock()
	for _, slv := range s.m.sm.slaves {
		if slv.authenticated && slv.freeMemory >= svc.g.Memory && (svc.s == nil || slv.freeMemory < svc.s.freeMemory) {
			svc.s = slv
		}
	}
	s.m.sm.mu.RUnlock()

	if svc.s == nil {
		if svc.State == protocol.Service_STATE_PENDING {
			log.Printf("service %s is pending", svc.Name)
			svc.State = protocol.Service_STATE_WAITING
		}
		return
	}

	svc.State = protocol.Service_STATE_SCHEDULED
	svc.Slave = svc.s.name
	log.Printf("scheduling service %s on slave %s", svc.Name, svc.Slave)
	svc.s.schedule(svc)
}

func (s *scheduler) deleteService(svc *service) {
	s.mu.Lock()
	idx := slices.Index(s.services, svc)
	if idx != -1 {
		s.services = slices.Delete(s.services, idx, idx+1)
	}
	s.mu.Unlock()
}

func (s *scheduler) getNextServiceName(prefix string) string {
	for i := 1; ; i++ {
		name := fmt.Sprintf("%s-%02d", prefix, i)
		if s.getService(name) == nil {
			return name
		}
	}
}

func (s *scheduler) getService(name string) *service {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, svc := range s.services {
		if svc.Name == name {
			return svc
		}
	}
	return nil
}
