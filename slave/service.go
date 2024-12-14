package main

import (
	"common"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"sync"
)

type serviceManager struct {
	s        *slave
	services []*service
	mu       sync.RWMutex
	tmpDir   string
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
	err := os.RemoveAll(svcm.tmpDir)
	if err != nil {
		return fmt.Errorf("failed to remove temp directory: %w", err)
	}
	err = os.MkdirAll(svcm.tmpDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	return nil
}

type service struct {
	*common.ServiceInfo
	g   *common.GroupInfo
	dir string
}

func (svcm *serviceManager) createService(svcInfo *common.ServiceInfo, groupInfo *common.GroupInfo) (*service, error) {
	svc := &service{
		ServiceInfo: svcInfo,
		g:           groupInfo,
	}

	svc.dir = path.Join(svcm.tmpDir, fmt.Sprintf("%s-%s", svc.Name, common.GenerateRandomHex(3)))
	err := os.MkdirAll(svc.dir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	templates := []string{"global_all", fmt.Sprintf("global_%s", svc.Type), groupInfo.Name}
	for _, tmpl := range templates {
		err = svcm.s.tmpl.downloadTemplate(tmpl)
		if err != nil {
			return nil, fmt.Errorf("failed to download template: %w", err)
		}
	}

	templateDir := path.Join(svcm.s.tmpl.templateDir, svc.g.Name)
	err = os.CopyFS(svc.dir, os.DirFS(templateDir))
	if err != nil {
		return nil, fmt.Errorf("failed to copy template directory: %w", err)
	}

	svcm.mu.Lock()
	svcm.services = append(svcm.services, svc)
	svcm.mu.Unlock()
	return svc, err
}

func (svcm *serviceManager) startService(svc *service) error {
	svc.Port = svcm.findNextFreePort(svc.g.StartPort)
	if svc.Port < 0 {
		return fmt.Errorf("failed to find free port")
	}

	var err error
	cmd := exec.Command("java", fmt.Sprintf("-Xmx%dM", svc.g.Memory), "-jar", "server.jar", "--port", strconv.Itoa(svc.Port))
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

func (svcm *serviceManager) findNextFreePort(startPort int) int {
	for port := startPort; port < 65535; port++ {
		if svcm.checkPort("", port) {
			return port
		}
	}
	return -1
}

func (svcm *serviceManager) checkPort(host string, port int) bool {
	bindAddr := net.JoinHostPort(host, strconv.Itoa(port))
	lis, err := net.Listen("tcp", bindAddr)
	if err != nil {
		return false
	}
	_ = lis.Close()
	return true
}
