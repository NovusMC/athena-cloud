package main

import (
	"common"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"protocol"
	"strconv"
	"syscall"
	"time"
)

type serviceManager struct {
	s        *slave
	services []*service
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
	*protocol.Service
	g    *protocol.Group
	conn net.Conn
	dir  string
	key  string
	cmd  *exec.Cmd
}

func (svcm *serviceManager) setConnected(svc *service, conn net.Conn) {
	svc.conn = conn
	svc.State = protocol.Service_STATE_ONLINE
	log.Printf("service %q connected", svc.Name)
	_ = protocol.SendPacket(svcm.s.conn, &protocol.PacketServiceOnline{
		ServiceName: svc.Name,
		Port:        svc.Port,
	})
}

func (svcm *serviceManager) createService(protoService *protocol.Service, group *protocol.Group) (*service, error) {
	svc := &service{
		Service: protoService,
		g:       group,
		key:     common.GenerateRandomHex(32),
	}

	svc.dir = path.Join(svcm.tmpDir, fmt.Sprintf("%s-%s", svc.Name, common.GenerateRandomHex(3)))
	err := os.MkdirAll(svc.dir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	_, port := common.SplitBindAddr(svcm.s.cfg.BindAddr)

	athenaConfig := map[string]any{
		"slaveAddr": "127.0.0.1",
		"slavePort": port,
		"key":       svc.key,
	}
	configBytes, err := json.Marshal(athenaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal athena config: %w", err)
	}
	athenaDataDir := path.Join(svc.dir, "plugins", "athena")
	err = os.MkdirAll(athenaDataDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create athena data directory: %w", err)
	}
	err = os.WriteFile(path.Join(athenaDataDir, "config.json"), configBytes, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write athena config: %w", err)
	}

	var typeSpecificTempl string
	if svc.Type == protocol.Service_TYPE_PROXY {
		typeSpecificTempl = "global_proxy"
	} else if svc.Type == protocol.Service_TYPE_SERVER {
		typeSpecificTempl = "global_server"
	} else {
		return nil, fmt.Errorf("unknown service type: %v", svc.Type)
	}

	log.Println("downloading templates")
	err = svcm.s.tmpl.syncTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to download template: %w", err)
	}

	templates := []string{"global_all", typeSpecificTempl, group.Name}
	for _, tmpl := range templates {
		templateDir := path.Join(svcm.s.tmpl.templateDir, tmpl)
		err = os.CopyFS(svc.dir, os.DirFS(templateDir))
		if err != nil {
			return nil, fmt.Errorf("failed to copy template %q: %w", tmpl, err)
		}
	}
	svcm.services = append(svcm.services, svc)
	return svc, err
}

func (svcm *serviceManager) startService(svc *service) error {
	svc.Port = svcm.findNextFreePort(svc.g.StartPort)
	if svc.Port < 0 {
		return fmt.Errorf("failed to find free port")
	}

	var err error
	svc.cmd = exec.Command("java", fmt.Sprintf("-Xmx%dM", svc.g.Memory), "-jar", "server.jar", "--port", strconv.Itoa(int(svc.Port)))
	svc.cmd.Dir, err = filepath.Abs(svc.dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	svc.cmd.Stdout = os.Stdout
	svc.cmd.Stderr = os.Stderr
	err = svc.cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	go func() {
		err := svc.cmd.Wait()
		if err != nil && err.Error() != "exit status 143" { // 143: exited by SIGTERM
			log.Printf("service %s exited with error: %v", svc.Name, err)
		} else {
			log.Printf("service %s exited", svc.Name)
		}
		svcm.s.ch <- serviceStoppedCmd{svc}
	}()
	return nil
}

func (svcm *serviceManager) stopService(svc *service) error {
	if svc.cmd == nil {
		return fmt.Errorf("service is not running")
	}
	err := svc.cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}
	return nil
}

func (svcm *serviceManager) getService(name string) *service {
	for _, svc := range svcm.services {
		if svc.Name == name {
			return svc
		}
	}
	return nil
}

func (svcm *serviceManager) deleteService(svc *service) error {
	if svc.cmd != nil {
		return fmt.Errorf("service %q is still running", svc.Name)
	}
	svcm.services = common.DeleteItem(svcm.services, svc)
	err := os.RemoveAll(svc.dir)
	if err != nil {
		return fmt.Errorf("failed to remove service directory: %w", err)
	}
	return nil
}

func (svcm *serviceManager) findNextFreePort(startPort int32) int32 {
	usedPorts := make(map[int]struct{})
	for _, svc := range svcm.services {
		if svc.Port > 0 {
			usedPorts[int(svc.Port)] = struct{}{}
		}
	}

	for port := startPort; port < 65535; port++ {
		if _, used := usedPorts[int(port)]; used {
			continue
		}
		if svcm.checkPort("", port) {
			return port
		}
	}
	return -1
}

func (svcm *serviceManager) checkPort(host string, port int32) bool {
	bindAddr := net.JoinHostPort(host, strconv.Itoa(int(port)))
	lis, err := net.Listen("tcp", bindAddr)
	if err != nil {
		return false
	}
	_ = lis.Close()
	return true
}

func handleServiceConnection(ch chan<- any, lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				log.Printf("connection error: %v", err)
			}
			break
		}
		log.Printf("new connection from %s", conn.RemoteAddr())
		go func() {
			var svc *service
			closed := false
			defer func() {
				closed = true
			}()
			go func() {
				time.Sleep(10 * time.Second)
				if svc == nil && !closed {
					closed = true
					log.Printf("service at %s timed out", conn.RemoteAddr())
					_ = conn.Close()
				}
			}()
			p, err := protocol.ReadPacket(conn)
			if err != nil {
				log.Printf("failed reading packet: %v", err)
				_ = conn.Close()
				return
			}
			if p, ok := p.(*protocol.PacketServiceConnect); !ok {
				log.Printf("%s sent invalid packet %T", conn.RemoteAddr(), p)
				_ = conn.Close()
				return
			} else {
				svcCh := make(chan *service)
				ch <- serviceConnectCmd{key: p.Key, conn: conn, svcCh: svcCh}
				svc = <-svcCh
			}
			if svc == nil {
				log.Printf("service at %s could not be identified", conn.RemoteAddr())
				_ = conn.Close()
				return
			}
			for {
				p, err := protocol.ReadPacket(conn)
				if err != nil {
					if !errors.Is(err, io.EOF) {
						log.Printf("failed reading packet: %v", err)
					}
					break
				}
				errCh := make(chan error)
				ch <- handleServicePacketCmd{svc: svc, p: p, errCh: errCh}
				if err = <-errCh; err != nil {
					log.Printf("failed handling packet: %v", err)
					break
				}
			}
			_ = conn.Close()
			log.Printf("service %q disconnected", svc.Name)
			ch <- serviceDisconnectCmd{svc: svc}
		}()
	}
}

func (svc *service) handlePacket(p proto.Message) error {
	return nil
}
