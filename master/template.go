package main

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/gokrazy/rsync/rsyncd"
	"io"
	"log"
	"net"
	"os"
	"path"
)

var (
	//go:embed assets/athena-kotlin-stdlib.jar
	athenaKotlinStdlibJar []byte
	//go:embed assets/athena-velocity.jar
	athenaVelocityJar []byte
	//go:embed assets/athena-paper.jar
	athenaPaperJar []byte
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
	templates := []string{"global_all/plugins", "global_proxy/plugins", "global_server/plugins"}
	for _, g := range tmpl.m.gm.groups {
		templates = append(templates, g.Name)
	}
	for _, name := range templates {
		err = tmpl.createTemplateDir(name)
		if err != nil {
			return err
		}
	}
	err = os.WriteFile(path.Join(tmpl.templateDir, "global_all/plugins", "athena-kotlin-stdlib.jar"), athenaKotlinStdlibJar, 0644)
	if err != nil {
		return fmt.Errorf("failed to write athena-kotlin-stdlib.jar: %w", err)
	}
	err = os.WriteFile(path.Join(tmpl.templateDir, "global_proxy/plugins", "athena-velocity.jar"), athenaVelocityJar, 0644)
	if err != nil {
		return fmt.Errorf("failed to write athena-velocity.jar: %w", err)
	}
	err = os.WriteFile(path.Join(tmpl.templateDir, "global_server/plugins", "athena-paper.jar"), athenaPaperJar, 0644)
	if err != nil {
		return fmt.Errorf("failed to write athena-paper.jar: %w", err)
	}
	err = os.WriteFile(path.Join(tmpl.templateDir, "global_server", "eula.txt"), []byte("eula=true\n"), 0644)
	if err != nil {
		return fmt.Errorf("failed to write eula.txt: %w", err)
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
		defer recoverPanic()
		err := srv.Serve(context.Background(), lis)
		if err != nil {
			fmt.Printf("failed to serve file server: %v\n", err)
		}
	}()
	return nil
}
