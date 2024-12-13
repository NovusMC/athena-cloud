package main

import (
	"common"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

type serviceManager struct {
	s      *slave
	tmpDir string
}

func newServiceManager(s *slave) (*serviceManager, error) {
	svcm := &serviceManager{
		s:      s,
		tmpDir: "tmp",
	}
	err := svcm.init()
	if err != nil {
		return nil, fmt.Errorf("failed to init service manager: %w", err)
	}
	return svcm, nil
}

func (svcm *serviceManager) init() error {
	err := os.MkdirAll(svcm.tmpDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	return nil
}

type service struct {
	name  string
	group *common.GroupInfo
	dir   string
}

func (svcm *serviceManager) createService(name string, group *common.GroupInfo) error {
	svc := &service{
		name:  name,
		group: group,
	}

	svc.dir = path.Join(svcm.tmpDir, fmt.Sprintf("%s-%s", name, common.GenerateRandomHex(3)))
	err := os.MkdirAll(svc.dir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	err = svcm.s.tmpl.syncTemplates()
	if err != nil {
		return fmt.Errorf("failed to sync templates: %w", err)
	}

	templateDir := path.Join(svcm.s.tmpl.templateDir, group.Name)
	err = os.CopyFS(svc.dir, os.DirFS(templateDir))
	if err != nil {
		return fmt.Errorf("failed to copy template directory: %w", err)
	}

	err = svcm.startService(svc)
	if err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

func (svcm *serviceManager) startService(svc *service) error {
	var err error
	cmd := exec.Command("java", fmt.Sprintf("-Xmx%dM", svc.group.Memory), "-jar", "server.jar")
	cmd.Dir, err = filepath.Abs(svc.dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	return nil
}
