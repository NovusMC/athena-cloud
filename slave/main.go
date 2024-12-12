package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"
)

type config struct {
	MasterAddr string `yaml:"master_addr"`
}

func main() {
	cfg, err := readConfig()
	if err != nil {
		log.Fatalf("error loading config: %v", cfg)
	}

	log.Printf("connecting to master at %q", cfg.MasterAddr)

	conn, err := grpc.NewClient(cfg.MasterAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect to master: %v", err)
	}
	defer conn.Close()

	c := proto.NewMasterClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := c.RegisterSlave(ctx, &proto.RegisterSlaveRequest{Name: "moin"})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.GetMessage())
}

func readConfig() (*config, error) {
	configFile := "slave.yaml"
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
