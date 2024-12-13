package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"common"

	"github.com/c-bata/go-prompt"
)

type master struct {
	cfg   *config
	gm    *groupManager
	sm    *slaveManager
	sched *scheduler
	tmpl  *templateManager
}

type config struct {
	BindAddr           string `yaml:"bind_addr"`
	FileServerBindAddr string `yaml:"file_server_bind_addr"`
	SecretKey          string `yaml:"secret_key"`
}

func main() {
	fmt.Print(common.Header)
	log.Printf("starting Athena-Master %s", common.Version)

	var (
		err error
		m   master
	)

	m.cfg, err = common.ReadConfig("master.yaml", config{
		BindAddr:           ":5000",
		FileServerBindAddr: ":5001",
		SecretKey:          common.GenerateRandomHex(32),
	})
	if err != nil {
		log.Fatalf("error loading config: %v", m.cfg)
	}

	m.gm, err = newGroupManager()
	if err != nil {
		log.Fatalf("%v", err)
	}

	m.tmpl, err = newTemplateManager(&m)
	if err != nil {
		log.Fatalf("%v", err)
	}

	m.sm = newSlaveManager()
	m.sched = newScheduler(&m)

	lis, err := net.Listen("tcp", m.cfg.BindAddr)
	if err != nil {
		log.Fatalf("failed starting server: %v", err)
	}
	defer lis.Close()
	log.Printf("listening on %q", m.cfg.BindAddr)

	err = m.tmpl.startFileServer()
	if err != nil {
		log.Fatalf("failed starting file server: %v", err)
	}

	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				log.Printf("connection error: %v", err)
				continue
			}
			log.Printf("new connection from %s", conn.RemoteAddr())
			h := &slaveHandler{m: &m, conn: conn}
			go func() {
				err = common.HandleConnection(conn, h)
				if err != nil && !errors.Is(err, io.EOF) {
					log.Printf("connection error: %v", err)
				}
				_ = conn.Close()
				if h.slave.authenticated {
					log.Printf("slave %q disconnected", h.slave.name)
				} else {
					log.Printf("authentication with slave %s failed", conn.RemoteAddr())
				}
			}()
		}
	}()

	go func() {
		t := time.NewTicker(time.Second)
		for range t.C {
			m.sched.scheduleServices()
		}
	}()

	cmd := newCommand(&m)

	var history []string
	for {
		in := prompt.Input("> ", func(document prompt.Document) []prompt.Suggest {
			return nil
		}, prompt.OptionHistory(history))
		history = append(history, in)
		if len(history) > 100 {
			history = history[len(history)-100:]
		}

		if in == "" {
			continue
		}

		args := strings.Split(in, " ")
		_ = cmd.Run(context.Background(), append([]string{""}, args...))
	}
}
