package main

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"log"
	"net"
	"protocol"
)

type interruptCmd struct{}

type slaveDisconnectCmd struct{}

type handleMasterPacketCmd struct {
	p     proto.Message
	errCh chan<- error
}

type handleServicePacketCmd struct {
	svc   *service
	p     proto.Message
	errCh chan<- error
}

type serviceDisconnectCmd struct {
	svc *service
}

type serviceConnectCmd struct {
	key   string
	conn  net.Conn
	svcCh chan<- *service
}

type serviceStoppedCmd struct {
	svc *service
}

func (s *slave) runCommandQueue(ch <-chan any) {
loop:
	for {
		cmd := <-ch
		switch cmd := cmd.(type) {
		case handleMasterPacketCmd:
			cmd.errCh <- s.handlePacket(cmd.p)
			close(cmd.errCh)
		case handleServicePacketCmd:
			cmd.errCh <- cmd.svc.handlePacket(cmd.p)
			close(cmd.errCh)
		case serviceConnectCmd:
			var connectedSvc *service
			for _, svc := range s.svcm.services {
				if svc.conn == nil && cmd.key == svc.key && svc.State == protocol.Service_STATE_SCHEDULED {
					s.svcm.setConnected(svc, cmd.conn)
					connectedSvc = svc
				}
			}
			cmd.svcCh <- connectedSvc
			close(cmd.svcCh)
		case serviceStoppedCmd:
			err := s.sendPacket(&protocol.PacketServiceStopped{
				ServiceName: cmd.svc.Name,
			})
			if err != nil {
				log.Printf("failed to send service stopped packet: %v", err)
			}
			cmd.svc.State = protocol.Service_STATE_OFFLINE
			cmd.svc.Port = 0
			cmd.svc.cmd = nil
			err = s.svcm.deleteService(cmd.svc)
			if err != nil {
				log.Printf("failed to delete service %q: %v", cmd.svc.Name, err)
			}
		case serviceDisconnectCmd:
			if cmd.svc.State == protocol.Service_STATE_PENDING || cmd.svc.State == protocol.Service_STATE_ONLINE {
				err := s.svcm.stopService(cmd.svc)
				if err != nil {
					log.Printf("failed to stop service %q: %v", cmd.svc.Name, err)
				}
			}
		case slaveDisconnectCmd:
			break loop
		case interruptCmd:
			log.Printf("received interrupt, exiting cleanly")
			_ = s.conn.Close()
			break loop
		default:
			panic(fmt.Sprintf("unknown command type %T", cmd))
		}
	}
}
