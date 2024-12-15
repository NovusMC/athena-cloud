package main

import (
	"context"
	"fmt"
	"github.com/fatih/color"
	"io"
	"log"
	"net"
	"protocol"
	"strings"
	"time"

	"common"

	"github.com/chzyer/readline"
)

type master struct {
	cfg   *config
	gm    *groupManager
	sm    *slaveManager
	sched *scheduler
	tmpl  *templateManager
	term  io.Writer
}

type config struct {
	BindAddr           string `json:"bind_addr"`
	FileServerBindAddr string `json:"file_server_bind_addr"`
	MinecraftBindAddr  string `json:"minecraft_bind_addr"`
	SecretKey          string `json:"secret_key"`
}

func main() {
	fmt.Println(color.RedString(strings.TrimPrefix(common.Header, "\n")))
	log.SetPrefix(color.WhiteString("[master] "))
	log.Printf("starting Athena-Master %s", common.Version)

	var (
		err error
		m   master
	)

	m.cfg, err = common.ReadConfig("master.yaml", config{
		BindAddr:           "0.0.0.0:5000",
		FileServerBindAddr: "0.0.0.0:5001",
		MinecraftBindAddr:  "0.0.0.0:25565",
		SecretKey:          common.GenerateRandomHex(32),
	})
	if err != nil {
		log.Fatalf("error loading config: %v", m.cfg)
	}

	m.gm, err = newGroupManager(&m)
	if err != nil {
		log.Fatalf("%v", err)
	}

	m.tmpl, err = newTemplateManager(&m)
	if err != nil {
		log.Fatalf("%v", err)
	}

	m.sm = newSlaveManager(&m)
	m.sched = newScheduler(&m)

	err = m.tmpl.startFileServer()
	if err != nil {
		log.Fatalf("failed starting file server: %v", err)
	}

	lis, err := net.Listen("tcp", m.cfg.BindAddr)
	if err != nil {
		log.Fatalf("failed starting server: %v", err)
	}
	defer lis.Close()
	log.Printf("listening on %s", m.cfg.BindAddr)

	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				log.Printf("connection error: %v", err)
				continue
			}
			log.Printf("new connection from %s", conn.RemoteAddr())
			slv := m.sm.newSlave(conn)
			go func() {
				for {
					p, err := protocol.ReadPacket(conn)
					if err != nil {
						log.Printf("failed reading packet: %v", err)
						break
					}
					err = slv.handlePacket(p)
					if err != nil {
						log.Printf("failed handling packet: %v", err)
						break
					}
				}
				_ = conn.Close()
				if slv.authenticated {
					log.Printf("slave %q disconnected", slv.name)
				} else {
					log.Printf("authentication with slave %s failed", conn.RemoteAddr())
				}
				m.sm.removeSlave(slv)
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

	l, err := readline.NewEx(&readline.Config{
		Prompt:            "\033[31m»\033[0m ",
		HistoryFile:       ".athena_history",
		HistorySearchFold: true,
	})
	if err != nil {
		log.Fatalf("failed starting readline: %v", err)
	}
	defer func() {
		_ = l.Close()
	}()
	l.CaptureExitSignal()
	log.SetOutput(l.Stderr())
	m.term = l.Stderr()

	for {
		line, _ := l.Readline()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		args := strings.Split(line, " ")
		_ = cmd.Run(context.Background(), append([]string{""}, args...))
	}
}
