package main

import (
	"common"
	"encoding/json"
	"fmt"
	"log"
	"net"
)

type slaveManager struct {
	slaves []*slave
}

type slave struct {
	name          string
	authenticated bool
	memory        int
	freeMemory    int
}

type slaveHandler struct {
	m     *master
	slave slave
	conn  net.Conn
}

func newSlaveManager() *slaveManager {
	return &slaveManager{}
}

func (h *slaveHandler) HandlePacket(packetType common.PacketType, data json.RawMessage) error {

	if !h.slave.authenticated && packetType != common.PacketTypeAuthenticate {
		return fmt.Errorf("slave not authenticated")
	}

	switch packetType {
	case common.PacketTypeAuthenticate:
		var auth common.PacketAuthenticate
		err := json.Unmarshal(data, &auth)
		if err != nil {
			return err
		}
		if auth.SecretKey != h.m.cfg.SecretKey {
			data, _ := json.Marshal(common.PacketAuthFailed{Message: "invalid secret key"})
			_ = common.SendPacket(h.conn, common.PacketTypeAuthFailed, data)
			return fmt.Errorf("invalid secret key")
		}
		for _, slv := range h.m.sm.slaves {
			if slv.name == auth.SlaveName {
				data, _ := json.Marshal(common.PacketAuthFailed{Message: "slave with name already exists"})
				_ = common.SendPacket(h.conn, common.PacketTypeAuthFailed, data)
				return fmt.Errorf("slave with name %s already exists", auth.SlaveName)
			}
		}
		h.slave.name = auth.SlaveName
		h.slave.memory = auth.Memory
		h.slave.freeMemory = auth.Memory
		h.slave.authenticated = true
		h.m.sm.slaves = append(h.m.sm.slaves, &h.slave)
		log.Printf("slave %q authenticated", h.slave.name)
		return common.SendPacket(h.conn, common.PacketTypeAuthenticate, nil)
	}

	return nil
}

var _ common.PacketHandler = &slaveHandler{}

func (s *slave) schedule(svc *service) {

}
