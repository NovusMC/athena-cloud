package common

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
)

type PacketType string

const (
	PacketTypeAuthenticate           PacketType = "authenticate"
	PacketTypeAuthSuccess            PacketType = "auth_success"
	PacketTypeAuthFailed             PacketType = "auth_failed"
	PacketTypeScheduleServiceRequest PacketType = "schedule_service_request"
)

type Packet struct {
	Type PacketType      `json:"type"`
	Data json.RawMessage `json:"data"`
}

type PacketAuthenticate struct {
	SlaveName string `json:"slave_name"`
	SecretKey string `json:"secret_key"`
	Memory    int    `json:"memory"`
}

type PacketAuthFailed struct {
	Message string `json:"message"`
}

type PacketScheduleServiceRequest struct {
	Name  string     `json:"name"`
	Group *GroupInfo `json:"group"`
}

func SendPacket(conn net.Conn, packetType PacketType, data any) error {
	p := struct {
		Type PacketType `json:"type"`
		Data any        `json:"data"`
	}{
		Type: packetType,
		Data: data,
	}
	enc := json.NewEncoder(conn)
	err := enc.Encode(p)
	if err != nil {
		return fmt.Errorf("failed to encode packet: %v", err)
	}
	return nil
}

type PacketHandler interface {
	HandlePacket(packetType PacketType, data json.RawMessage) error
}

func HandleConnection(conn io.Reader, handler PacketHandler) error {
	decoder := json.NewDecoder(conn)
	for {
		var p Packet
		err := decoder.Decode(&p)
		if err != nil {
			return err
		}

		err = handler.HandlePacket(p.Type, p.Data)
		if err != nil {
			return err
		}
	}
}
