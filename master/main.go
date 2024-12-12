package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"proto"

	"github.com/c-bata/go-prompt"
	"github.com/urfave/cli/v3"
	"google.golang.org/grpc"

	"gopkg.in/yaml.v3"
)

type config struct {
	BindAddr string `yaml:"bind_addr"`
}

type server struct {
	proto.UnimplementedMasterServer
}

func (server) RegisterSlave(context.Context, *proto.RegisterSlaveRequest) (*proto.RegisterSlaveResponse, error) {
	return &proto.RegisterSlaveResponse{Message: "moin"}, nil
}

func main() {
	cfg, err := readConfig()
	if err != nil {
		log.Fatalf("error loading config: %v", cfg)
	}

	lis, err := net.Listen("tcp", cfg.BindAddr)
	if err != nil {
		log.Fatalf("failed starting server: %v", err)
	}
	log.Printf("listening on %q", cfg.BindAddr)

	s := grpc.NewServer()
	proto.RegisterMasterServer(s, server{})
	go func() {
		if err = s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	var history []string
	for {
		in := prompt.Input("> ", func(document prompt.Document) []prompt.Suggest {
			return nil
		}, prompt.OptionHistory(history))
		history = append(history, in)
		if len(history) > 100 {
			history = history[len(history)-100:]
		}

		cmd := &cli.Command{
			Name:  "boom",
			Usage: "make an explosive entrance",
			Action: func(context.Context, *cli.Command) error {
				fmt.Println("boom! I say!")
				return nil
			},
		}

		args := strings.Split(in, " ")
		cmd.Run(context.Background(), args)
	}
}

func readConfig() (*config, error) {
	configFile := "master.yaml"
	var cfg config

	bytes, err := os.ReadFile(configFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {

			bytes, err = yaml.Marshal(cfg)
			if err != nil {
				return nil, fmt.Errorf("error marshaling empty config: %w", err)
			}

			err = os.WriteFile(configFile, bytes, 0644)
			if err != nil {
				return nil, fmt.Errorf("error writing config file: %w", err)
			}

			os.Exit(1)
			return nil, nil
		}
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	err = yaml.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &cfg, nil
}
