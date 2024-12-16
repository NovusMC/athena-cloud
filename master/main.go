package main

import (
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/urfave/cli/v3"
	"io"
	"log"
	"net"
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
	cli   *cli.Command
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

	ch := make(chan any)

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
	m.cli = newCli(ch, &m)

	err = m.tmpl.startFileServer()
	if err != nil {
		log.Fatalf("failed starting file server: %v", err)
	}

	lis, err := net.Listen("tcp", m.cfg.BindAddr)
	if err != nil {
		log.Fatalf("failed starting server: %v", err)
	}
	defer func() {
		_ = lis.Close()
	}()
	log.Printf("listening on %s", m.cfg.BindAddr)
	go handleSlaveConnection(ch, lis)

	go func() {
		t := time.NewTicker(time.Second)
		for range t.C {
			ch <- scheduleServicesCmd{}
		}
	}()

	l, err := readline.NewEx(&readline.Config{
		Prompt:            "\033[31m»\033[0m ",
		HistoryFile:       ".athena_history",
		HistorySearchFold: true,
	})
	if err != nil {
		log.Fatalf("failed starting readline: %v", err)
	}
	l.CaptureExitSignal()
	log.SetOutput(l.Stderr())
	m.term = l.Stderr()

	go func() {
		for {
			line, err := l.Readline()
			if err != nil {
				if errors.Is(err, readline.ErrInterrupt) {
					continue
				}
				if !errors.Is(err, io.EOF) {
					log.Printf("error reading line: %v", err)
				}
				break
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			args := strings.Split(line, " ")
			args = append([]string{""}, args...)
			ch <- runCliCmd{args}
		}
	}()

	m.runCommandQueue(ch)

	log.Printf("shutting down")
}
