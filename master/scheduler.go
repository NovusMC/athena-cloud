package main

import (
	"fmt"
	"log"
)

type serviceState string

const (
	serviceStatePending   serviceState = "pending"
	serviceStateWaiting   serviceState = "waiting"
	serviceStateScheduled serviceState = "scheduled"
)

type service struct {
	name  string
	state serviceState
	group *group
	slave *slave
}

type scheduler struct {
	m        *master
	services []*service
}

func newScheduler(m *master) *scheduler {
	return &scheduler{m: m}
}

func (s *scheduler) scheduleServices() {
	for _, g := range s.m.gm.groups {
		if len(g.services) < g.info.MinServices {
			for i := 0; i < g.info.MinServices-len(g.services); i++ {
				s.createService(g)
			}
		}
	}

	for _, svc := range s.services {
		s.scheduleService(svc)
	}
}

func (s *scheduler) createService(g *group) {
	name := s.getNextServiceName(g.info.Name)
	svc := &service{
		name:  name,
		state: serviceStatePending,
		group: g,
	}
	s.services = append(s.services, svc)
	g.services = append(g.services, svc)
	log.Printf("service %s created", svc.name)
}

func (s *scheduler) scheduleService(svc *service) {
	if svc.slave != nil {
		return
	}

	for _, slv := range s.m.sm.slaves {
		if slv.freeMemory >= svc.group.info.Memory && (svc.slave == nil || slv.freeMemory < svc.slave.freeMemory) {
			svc.slave = slv
		}
	}
	if svc.slave == nil {
		if svc.state == serviceStatePending {
			log.Printf("service %s is pending", svc.name)
			svc.state = serviceStateWaiting
		}
		return
	}

	svc.state = serviceStateScheduled
	log.Printf("scheduling service %s on slave %s", svc.name, svc.slave.name)
	svc.slave.schedule(svc)
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
	for _, svc := range s.services {
		if svc.name == name {
			return svc
		}
	}
	return nil
}
