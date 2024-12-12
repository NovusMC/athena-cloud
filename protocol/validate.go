package protocol

import (
	"errors"
	"fmt"
)

func (g *Group) Validate() error {
	if g.Name == "" {
		return errors.New("group name cannot be empty")
	}
	if g.Type != Service_TYPE_PROXY && g.Type != Service_TYPE_SERVER {
		return fmt.Errorf("invalid group type %d", g.Type)
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
