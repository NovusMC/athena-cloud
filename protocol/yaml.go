package protocol

import (
	"fmt"
	"strings"
)

func (s *Service_Type) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return fmt.Errorf("cannot unmarshal service type: %w", err)
	}
	if val, ok := Service_Type_value[strings.ToUpper("TYPE_"+str)]; ok {
		*s = Service_Type(val)
		return nil
	}
	return fmt.Errorf("invalid service type: %s", str)
}

func (s Service_Type) MarshalYAML() (interface{}, error) {
	return strings.ToLower(strings.TrimPrefix(s.String(), "TYPE_")), nil
}

func (s *Service_State) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return fmt.Errorf("cannot unmarshal service state: %w", err)
	}
	if val, ok := Service_State_value[strings.ToUpper("STATE_"+str)]; ok {
		*s = Service_State(val)
		return nil
	}
	return fmt.Errorf("invalid service state: %s", str)
}

func (s Service_State) MarshalYAML() (interface{}, error) {
	return strings.ToLower(strings.TrimPrefix(s.String(), "STATE_")), nil
}
