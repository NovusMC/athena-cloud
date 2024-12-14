package main

import (
	"common"
	"context"
	"fmt"
	"github.com/urfave/cli/v3"
	"log"
	"os"
	"protocol"
	"strings"
)

func newCommand(m *master) *cli.Command {
	cmd := &cli.Command{
		ExitErrHandler: func(ctx context.Context, command *cli.Command, err error) {
			log.Printf("error: %v", err)
		},

		Name: "master",

		Commands: []*cli.Command{
			{
				Name:  "exit",
				Usage: "Shut down master",
				Action: func(ctx context.Context, command *cli.Command) error {
					os.Exit(0)
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
					newServiceListCmd(m),
				},
			},
		},
	}

	return cmd
}

func newServiceListCmd(m *master) *cli.Command {
	cmd := &cli.Command{
		Name:  "list",
		Usage: "List services",
		Action: func(ctx context.Context, command *cli.Command) error {
			m.sched.mu.RLock()
			defer m.sched.mu.RUnlock()
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
			err := m.gm.loadGroups()
			if err != nil {
				return fmt.Errorf("cannot reload groups: %w", err)
			}
			log.Println("Groups reloaded")
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
