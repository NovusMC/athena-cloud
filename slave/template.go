package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func checkForRsync() error {
	_, err := exec.LookPath("rsync")
	if err != nil {
		return fmt.Errorf("rsync is not installed: %v", err)
	}
	return nil
}

type templateManager struct {
	s           *slave
	templateDir string
}

func newTemplateManager(s *slave) (*templateManager, error) {
	tmpl := &templateManager{
		s:           s,
		templateDir: "template_cache",
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

func (tmpl *templateManager) downloadTemplate(name string) error {
	path, err := filepath.Abs(tmpl.templateDir)
	if err != nil {
		return fmt.Errorf("could not get absolute path for template directory: %w", err)
	}
	url := fmt.Sprintf("rsync://%s/templates/%s", tmpl.s.cfg.FileServerHost, name)
	cmd := exec.Command("rsync", "-a", "--delete", "--port", tmpl.s.cfg.FileServerPort, url, path)
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to sync templates: %w", err)
	}
	return nil
}
