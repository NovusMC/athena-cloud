package common

import (
	"errors"
	"fmt"
)

type GroupInfo struct {
	Name        string      `json:"name"`
	Type        ServiceType `json:"type"`
	MinServices int         `json:"min_services"`
	MaxServices int         `json:"max_services"`
	Memory      int         `json:"memory"`
	StartPort   int         `json:"start_port"`
}

func (g *GroupInfo) Validate() error {
	if g.Name == "" {
		return errors.New("group name cannot be empty")
	}
	if g.Type != ServiceTypeProxy && g.Type != ServiceTypeServer {
		return fmt.Errorf("unknown group type %q", g.Type)
	}
	if g.MinServices < 0 {
		return errors.New("min_services cannot be smaller than 0")
	}
	if g.MaxServices < g.MinServices {
		return errors.New("min_services cannot be larger than max_services")
	}
	if g.Memory < 1 {
		return errors.New("memory cannot be smaller than 1")
	}
	return nil
}
