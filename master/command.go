package main

import (
	"common"
	"context"
	"fmt"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
	"log"
	"os"
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
			err := yaml.NewEncoder(os.Stdout).Encode(m.gm.groups)
			if err != nil {
				return fmt.Errorf("cannot marshal groups: %w", err)
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
				Usage:    "Group type (proxy, server)",
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
			g := &common.GroupInfo{
				Name:        cmd.String("name"),
				Type:        common.GroupType(cmd.String("type")),
				MinServices: int(cmd.Int("min-services")),
				MaxServices: int(cmd.Int("max-services")),
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
