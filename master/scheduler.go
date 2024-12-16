package main

import (
	"common"
	"fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"log"
	"protocol"
)

type service struct {
	*protocol.Service
	g *group
	s *slave
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
		nSvcs := int32(len(s.m.gm.services(g)))
		if nSvcs < g.MinServices {
			for i := int32(0); i < g.MinServices-nSvcs; i++ {
				s.createService(g)
			}
		}
		if nSvcs > g.MaxServices {
			var svcs []*service
			for _, svc := range s.m.gm.services(g) {
				if svc.State != protocol.Service_STATE_STOPPING {
					svcs = append(svcs, svc)
				}
			}
			toStop := svcs[g.MaxServices:]
			for _, svc := range toStop {
				if svc.State == protocol.Service_STATE_ONLINE {
					err := s.stopService(svc)
					if err != nil {
						log.Printf("failed to stop service %q: %v", svc.Name, err)
					}
				} else {
					err := s.deleteService(svc)
					if err != nil {
						log.Printf("failed to delete service %q: %v", svc.Name, err)
					}
				}
			}
		}
	}

	for _, svc := range s.services {
		s.scheduleService(svc)
	}
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
	s.services = append(s.services, svc)
	log.Printf("service %q created", svc.Name)
}

func (s *scheduler) scheduleService(svc *service) {
	if svc.s != nil {
		return
	}

	for _, slv := range s.m.sm.slaves {
		if slv.authenticated && slv.freeMemory >= svc.g.Memory && (svc.s == nil || slv.freeMemory < svc.s.freeMemory) {
			svc.s = slv
		}
	}
	if svc.s == nil {
		return
	}

	svc.State = protocol.Service_STATE_SCHEDULED
	svc.Slave = svc.s.name
	log.Printf("scheduling service %q on slave %q", svc.Name, svc.Slave)
	svc.s.schedule(svc)
}

func (s *scheduler) stopService(svc *service) error {
	log.Printf("stopping service %q on slave %q", svc.Name, svc.Slave)
	if svc.s == nil {
		return fmt.Errorf("service %q is not running", svc.Name)
	}
	svc.State = protocol.Service_STATE_STOPPING
	err := svc.s.sendPacket(&protocol.PacketStopService{
		ServiceName: svc.Name,
	})
	if err != nil {
		return fmt.Errorf("failed to stop service %q on slave %q: %v", svc.Name, svc.Slave, err)
	}
	return nil
}

func (s *scheduler) deleteService(svc *service) error {
	if svc.State == protocol.Service_STATE_SCHEDULED ||
		svc.State == protocol.Service_STATE_ONLINE ||
		svc.State == protocol.Service_STATE_STOPPING {
		return fmt.Errorf("service %q is in state %s", svc.Name, svc.State)
	}
	s.services = common.DeleteItem(s.services, svc)
	log.Printf("service %q deleted", svc.Name)
	return nil
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
		if svc.Name == name {
			return svc
		}
	}
	return nil
}

func (svc *service) sendPacket(p proto.Message) error {
	if svc.s == nil {
		return fmt.Errorf("service %q is not running", svc.Name)
	}
	payload, err := anypb.New(p)
	if err != nil {
		return fmt.Errorf("failed to marshal packet: %v", err)
	}
	return svc.s.sendPacket(&protocol.ServiceEnvelope{
		ServiceName: svc.Name,
		Payload:     payload,
	})
}
