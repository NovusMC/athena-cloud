package main

import (
	"common"
	"fmt"
	"log"
	"slices"
	"sync"
)

type service struct {
	*common.ServiceInfo
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
		if len(g.services) < g.MinServices {
			for i := 0; i < g.MinServices-len(g.services); i++ {
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
		ServiceInfo: &common.ServiceInfo{
			Name:  name,
			State: common.ServiceStatePending,
			Type:  g.Type,
			Group: g.Name,
			Slave: "",
			Port:  0,
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
		if slv.freeMemory >= svc.g.Memory && (svc.s == nil || slv.freeMemory < svc.s.freeMemory) {
			svc.s = slv
		}
	}
	s.m.sm.mu.RUnlock()

	if svc.s == nil {
		if svc.State == common.ServiceStatePending {
			log.Printf("service %s is pending", svc.Name)
			svc.State = common.ServiceStateWaiting
		}
		return
	}

	svc.State = common.ServiceStateScheduled
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
