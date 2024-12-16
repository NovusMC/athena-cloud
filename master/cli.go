package main

import (
	"common"
	"context"
	"fmt"
	"github.com/urfave/cli/v3"
	"log"
	"protocol"
	"strings"
)

func newCli(ch chan<- any, m *master) *cli.Command {
	cmd := &cli.Command{
		ExitErrHandler: func(context.Context, *cli.Command, error) {},

		Name:      "master",
		Writer:    m.term,
		ErrWriter: m.term,

		Commands: []*cli.Command{
			{
				Name:  "shutdown",
				Usage: "Shut down master",
				Action: func(ctx context.Context, command *cli.Command) error {
					go func() {
						ch <- masterShutdownCmd{}
					}()
					return nil
				},
			},
			{
				Name:    "group",
				Aliases: []string{"groups"},
				Usage:   "Manage groups",
				Commands: []*cli.Command{
					newGroupCreateCmd(m),
					newGroupListCmd(m),
					newGroupReloadCmd(m),
				},
			},
			{
				Name:    "service",
				Aliases: []string{"svc"},
				Usage:   "Manage services",
				Commands: []*cli.Command{
					newServiceStopCmd(m),
					newServiceListCmd(m),
				},
			},
		},
	}

	return cmd
}

func newServiceStopCmd(m *master) *cli.Command {
	var svcName string
	cmd := &cli.Command{
		Name:  "stop",
		Usage: "Stops a service",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:        "<name>",
				Destination: &svcName,
				Min:         1,
				Max:         1,
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			svc := m.sched.getService(svcName)
			if svc == nil {
				return fmt.Errorf("unknown service: %s", svcName)
			}
			err := m.sched.stopService(svc)
			if err != nil {
				return fmt.Errorf("failed to stop service: %w", err)
			}
			return nil
		},
	}
	return cmd
}

func newServiceListCmd(m *master) *cli.Command {
	cmd := &cli.Command{
		Name:  "list",
		Usage: "List services",
		Action: func(ctx context.Context, command *cli.Command) error {
			log.Println("List of services:")
			var svcs []*protocol.Service
			for _, svc := range m.sched.services {
				svcs = append(svcs, svc.Service)
			}
			err := common.EncodeYamlColorized(svcs, m.term)
			if err != nil {
				return fmt.Errorf("cannot marshal services: %w", err)
			}
			return nil
		},
	}
	return cmd
}

func newGroupReloadCmd(m *master) *cli.Command {
	cmd := &cli.Command{
		Name:  "reload",
		Usage: "Reload groups",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			err := m.gm.reloadGroups()
			if err != nil {
				return fmt.Errorf("cannot reload groups: %w", err)
			}
			log.Println("groups reloaded")
			return nil
		},
	}
	return cmd
}

func newGroupListCmd(m *master) *cli.Command {
	cmd := &cli.Command{
		Name:  "list",
		Usage: "List all groups",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			log.Println("List of groups:")
			var groups []*protocol.Group
			for _, g := range m.gm.groups {
				groups = append(groups, g.Group)
			}
			err := common.EncodeYamlColorized(groups, m.term)
			if err != nil {
				return fmt.Errorf("cannot marshal group: %w", err)
			}
			return nil
		},
	}
	return cmd
}

func newGroupCreateCmd(m *master) *cli.Command {
	cmd := &cli.Command{
		Name:  "create",
		Usage: "Create a group",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Usage:    "Group name",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "type",
				Usage:    "Group type (Proxy, Server)",
				Required: true,
			},
			&cli.IntFlag{
				Name:     "min-services",
				Usage:    "Minimum amount of online services",
				Required: true,
			},
			&cli.IntFlag{
				Name:     "max-services",
				Usage:    "Maximum amount of online services",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			t, exists := protocol.Service_Type_value[strings.ToUpper("TYPE_"+cmd.String("type"))]
			if !exists {
				return fmt.Errorf("unknown service type: %s", cmd.String("type"))
			}
			g := &protocol.Group{
				Name:        cmd.String("name"),
				Type:        protocol.Service_Type(t),
				MinServices: int32(cmd.Int("min-services")),
				MaxServices: int32(cmd.Int("max-services")),
			}
			err := m.gm.createGroup(g)
			if err != nil {
				return fmt.Errorf("cannot create group: %w", err)
			}
			log.Printf("Successfully created group %q", g.Name)
			return nil
		},
	}
	return cmd
}
