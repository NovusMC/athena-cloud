package main

import (
	"github.com/ergochat/readline"
	"github.com/fatih/color"
	"github.com/urfave/cli/v3"
	"io"
	"log"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"common"
)

type master struct {
	cfg   *config
	gm    *groupManager
	sm    *slaveManager
	sched *scheduler
	tmpl  *templateManager
	term  io.Writer
	cli   *cli.Command
	sc    *screen
}

type config struct {
	BindAddr           string `json:"bind_addr"`
	FileServerBindAddr string `json:"file_server_bind_addr"`
	SecretKey          string `json:"secret_key"`
}

func main() {
	err := os.MkdirAll("logs", 0755)
	if err != nil {
		panic(err)
	}

	logFile, err := os.OpenFile("logs/master.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logFile.Close()
	}()

	defer recoverPanic()

	l, err := readline.NewEx(&readline.Config{
		Prompt:            "\033[31m»\033[0m ",
		HistoryFile:       ".athena_history",
		HistorySearchFold: true,
	})
	if err != nil {
		log.Fatalf("failed starting readline: %v", err)
	}
	l.CaptureExitSignal()
	defer func() {
		_ = l.Close()
	}()

	outWriter := io.MultiWriter(l.Stderr(), common.NewStripAnsiWriter(logFile))
	log.SetOutput(outWriter)

	log.SetPrefix(color.RedString("[master] "))
	log.Println(color.RedString(common.Header))
	log.Printf("starting Athena-Master %s", common.Version)

	m := master{term: outWriter}
	ch := make(chan any)

	m.cfg, err = common.ReadConfig("master.yaml", config{
		BindAddr:           "0.0.0.0:5000",
		FileServerBindAddr: "0.0.0.0:5001",
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

	m.sc = newScreen()
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
		defer recoverPanic()
		t := time.NewTicker(time.Second)
		for range t.C {
			ch <- scheduleServicesCmd{}
		}
	}()

	running := true
	go func() {
		defer recoverPanic()
		for running {
			line, _ := l.Readline()
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
	running = false

	log.Printf("shutting down")
}

func recoverPanic() {
	if r := recover(); r != nil {
		stack := string(debug.Stack())
		linesToSkip := 4
		if strings.HasPrefix(stack, "goroutine") {
			linesToSkip++
		}
		stack = strings.Join(strings.Split(stack, "\n")[linesToSkip:], "\n")
		log.Fatalf("panic: %v\n%s", r, stack)
	}
}
