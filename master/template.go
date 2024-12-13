package main

import (
	"context"
	"fmt"
	"github.com/gokrazy/rsync/rsyncd"
	"io"
	"log"
	"net"
	"os"
)

type templateManager struct {
	m           *master
	templateDir string
}

func newTemplateManager(m *master) (*templateManager, error) {
	tmpl := &templateManager{
		m:           m,
		templateDir: "templates",
	}
	err := tmpl.init()
	if err != nil {
		return nil, fmt.Errorf("failed to init template manager: %w", err)
	}
	return tmpl, nil
}

func (tmpl *templateManager) init() error {
	err := os.MkdirAll(tmpl.templateDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create template directory: %w", err)
	}
	return nil
}

func (tmpl *templateManager) startFileServer() error {
	srv, err := rsyncd.NewServer([]rsyncd.Module{
		{
			Name: "templates",
			Path: tmpl.templateDir,
		},
	}, rsyncd.WithLogger(log.New(io.Discard, "", 0)))
	if err != nil {
		return fmt.Errorf("failed to create rsync server: %w", err)
	}

	lis, err := net.Listen("tcp", tmpl.m.cfg.FileServerBindAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	log.Printf("started file server on %s\n", tmpl.m.cfg.FileServerBindAddr)

	go func() {
		err := srv.Serve(context.Background(), lis)
		if err != nil {
			fmt.Printf("failed to serve file server: %v\n", err)
		}
	}()
	return nil
}
