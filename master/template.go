package main

import (
	"context"
	"fmt"
	"github.com/gokrazy/rsync/rsyncd"
	"io"
	"log"
	"net"
	"os"
	"path"
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
	templates := []string{"global_all", "global_proxy", "global_server"}
	for _, g := range tmpl.m.gm.groups {
		templates = append(templates, g.Name)
	}
	for _, name := range templates {
		err = tmpl.createTemplateDir(name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (tmpl *templateManager) createTemplateDir(name string) error {
	err := os.MkdirAll(path.Join(tmpl.templateDir, name), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
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
