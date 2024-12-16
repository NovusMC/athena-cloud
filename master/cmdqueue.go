package main

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/proto"
	"log"
	"net"
)

type masterShutdownCmd struct{}

type scheduleServicesCmd struct{}

type runCliCmd struct {
	args []string
}

type createSlaveCmd struct {
	conn  net.Conn
	slvCh chan<- *slave
}

type removeSlaveCmd struct {
	slv *slave
}

type handleSlavePacketCmd struct {
	slv   *slave
	p     proto.Message
	errCh chan<- error
}

func (m *master) runCommandQueue(ch <-chan any) {
loop:
	for {
		cmd := <-ch
		switch cmd := cmd.(type) {
		case masterShutdownCmd:
			break loop
		case scheduleServicesCmd:
			m.sched.scheduleServices()
		case runCliCmd:
			err := m.cli.Run(context.Background(), cmd.args)
			if err != nil {
				log.Printf("%v", err)
			}
		case createSlaveCmd:
			slv := m.sm.newSlave(cmd.conn)
			cmd.slvCh <- slv
			close(cmd.slvCh)
		case removeSlaveCmd:
			m.sm.removeSlave(cmd.slv)
		case handleSlavePacketCmd:
			cmd.errCh <- cmd.slv.handlePacket(cmd.p)
			close(cmd.errCh)
		default:
			panic(fmt.Sprintf("unknown command type %T", cmd))
		}
	}
}
