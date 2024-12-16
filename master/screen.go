package main

import (
	"fmt"
	"log"
	"protocol"
)

type screen struct {
	svc *service
}

func newScreen() *screen {
	return &screen{}
}

func (sc *screen) attach(svc *service) error {
	if sc.svc != nil {
		return fmt.Errorf("screen already enabled")
	}
	if svc.s == nil {
		return fmt.Errorf("service not connected")
	}
	err := svc.s.sendPacket(&protocol.PacketAttachScreen{
		ServiceName: svc.Name,
	})
	if err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}
	sc.svc = svc
	log.Printf("attached to service %q", svc.Name)
	return nil
}

func (sc *screen) detach() error {
	if sc.svc == nil {
		return fmt.Errorf("no service in screen")
	}
	if sc.svc.s == nil {
		sc.svc = nil
		return nil
	}
	var err error
	if sc.svc.State == protocol.Service_STATE_SCHEDULED || sc.svc.State == protocol.Service_STATE_ONLINE || sc.svc.State == protocol.Service_STATE_STOPPING {
		err = sc.svc.s.sendPacket(&protocol.PacketDetachScreen{
			ServiceName: sc.svc.Name,
		})
	}
	log.Printf("detached from service %q", sc.svc.Name)
	sc.svc = nil
	if err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}
	return nil
}
