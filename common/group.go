package common

import (
	"errors"
	"fmt"
)

type GroupInfo struct {
	Name        string    `yaml:"name"`
	Type        GroupType `yaml:"type"`
	MinServices int       `yaml:"min_services"`
	MaxServices int       `yaml:"max_services"`
	Memory      int       `yaml:"memory"`
}

type GroupType string

const (
	GroupTypeProxy  GroupType = "proxy"
	GroupTypeServer           = "server"
)

func (g *GroupInfo) Validate() error {
	if g.Name == "" {
		return errors.New("group name cannot be empty")
	}
	if g.Type != GroupTypeProxy && g.Type != GroupTypeServer {
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
