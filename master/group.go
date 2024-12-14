package main

import (
	"common"
	"fmt"
	"github.com/goccy/go-yaml"
	"os"
	"path"
	"strings"
	"sync"
)

type group struct {
	*common.GroupInfo
	services []*service
	mu       sync.RWMutex
}

type groupManager struct {
	m        *master
	groupDir string
	groups   []*group
	mu       sync.RWMutex
}

func newGroupManager(m *master) (*groupManager, error) {
	gm := &groupManager{m: m, groupDir: "groups"}
	err := gm.loadGroups()
	if err != nil {
		return nil, fmt.Errorf("cannot load groups: %w", err)
	}
	return gm, nil
}

func (gm *groupManager) loadGroups() error {
	err := os.MkdirAll(gm.groupDir, 0755)
	if err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}
	files, err := os.ReadDir(gm.groupDir)
	if err != nil {
		return fmt.Errorf("cannot read group files: %w", err)
	}

	var groups []*group
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if !strings.HasSuffix(f.Name(), ".yaml") {
			continue
		}
		fileName := path.Join(gm.groupDir, f.Name())
		b, err := os.ReadFile(fileName)
		if err != nil {
			return fmt.Errorf("cannot read group file: %w", err)
		}
		var info common.GroupInfo
		err = yaml.Unmarshal(b, &info)
		if err != nil {
			return fmt.Errorf("cannot parse group file at %q: %w", fileName, err)
		}
		err = info.Validate()
		if err != nil {
			return fmt.Errorf("invalid group file %q: %w", fileName, err)
		}
		if f.Name() != info.Name+".yaml" {
			return fmt.Errorf("invalid group file: %q: file name must be %q", f.Name(), info.Name+".yaml")
		}
		groups = append(groups, &group{GroupInfo: &info})
	}

	gm.mu.Lock()
	gm.groups = groups
	gm.mu.Unlock()
	return nil
}

func (gm *groupManager) saveGroup(g *common.GroupInfo) error {
	err := g.Validate()
	if err != nil {
		return fmt.Errorf("invalid group: %w", err)
	}
	bytes, err := yaml.Marshal(g)
	if err != nil {
		return fmt.Errorf("cannot marshal group: %w", err)
	}
	err = os.WriteFile(path.Join(gm.groupDir, g.Name+".yaml"), bytes, 0644)
	if err != nil {
		return fmt.Errorf("cannot write group file: %w", err)
	}
	return nil
}

func (gm *groupManager) createGroup(info *common.GroupInfo) error {
	err := info.Validate()
	if err != nil {
		return fmt.Errorf("invalid group: %w", err)
	}
	for _, g := range gm.groups {
		if g.Name == info.Name {
			return fmt.Errorf("group %q already exists", info.Name)
		}
	}
	err = gm.saveGroup(info)
	if err != nil {
		return fmt.Errorf("cannot save group: %w", err)
	}
	gm.mu.Lock()
	gm.groups = append(gm.groups, &group{GroupInfo: info})
	gm.mu.Unlock()
	err = gm.m.tmpl.createTemplateDir(info.Name)
	if err != nil {
		return fmt.Errorf("failed to create template directory: %w", err)
	}
	return nil
}
