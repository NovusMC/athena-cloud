package protocol

import (
	"encoding/binary"
	"fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/anypb"
	"io"
	"strings"
)

var typeRegistry map[string]func() proto.Message

func populateRegistry() map[string]func() proto.Message {
	registry := make(map[string]func() proto.Message)
	protoregistry.GlobalTypes.RangeMessages(func(mt protoreflect.MessageType) bool {
		name := string(mt.Descriptor().FullName())
		registry[name] = func() proto.Message { return mt.New().Interface() }
		return true
	})
	return registry
}

func SendPacket(w io.Writer, packet proto.Message) error {
	payload, err := anypb.New(packet)
	if err != nil {
		return fmt.Errorf("failed to create Any message: %w", err)
	}
	env := &Envelope{Payload: payload}
	buf, err := proto.Marshal(env)
	if err != nil {
		return fmt.Errorf("failed to marshal envelope: %w", err)
	}
	err = binary.Write(w, binary.BigEndian, uint32(len(buf)))
	if err != nil {
		return fmt.Errorf("failed to write envelope size: %w", err)
	}
	_, err = w.Write(buf)
	if err != nil {
		return fmt.Errorf("failed to write envelope: %w", err)
	}
	return nil
}

func ReadPacket(r io.Reader) (proto.Message, error) {
	var bufLen uint32
	err := binary.Read(r, binary.BigEndian, &bufLen)
	if err != nil {
		return nil, fmt.Errorf("failed to read envelope size: %w", err)
	}
	var buf = make([]byte, bufLen)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read envelope: %w", err)
	}
	env := &Envelope{}
	err = proto.Unmarshal(buf, env)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal envelope: %w", err)
	}
	typeName := strings.TrimPrefix(env.Payload.TypeUrl, "type.googleapis.com/")
	if typeRegistry == nil {
		typeRegistry = populateRegistry()
	}
	factory, found := typeRegistry[typeName]
	if !found {
		return nil, fmt.Errorf("unknown message type: %s", typeName)
	}
	message := factory()
	if err := env.Payload.UnmarshalTo(message); err != nil {
		return nil, fmt.Errorf("failed to decode dynamic message: %w", err)
	}
	return message, nil
}
