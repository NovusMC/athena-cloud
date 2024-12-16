package main

import (
	"common"
	"errors"
	"fmt"
	"github.com/goccy/go-yaml"
	"os"
	"path"
	"protocol"
	"strings"
)

type group struct {
	*protocol.Group
}

type groupManager struct {
	m        *master
	groupDir string
	groups   []*group
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
	if gm.groups != nil {
		return fmt.Errorf("groups already loaded")
	}
	groupInfos, err := gm.loadGroupInfos()
	if err != nil {
		return fmt.Errorf("failed to load group files: %w", err)
	}
	var errs []error
	for _, g := range groupInfos {
		gm.groups = append(gm.groups, &group{Group: g})
	}
	return errors.Join(errs...)
}

func (gm *groupManager) reloadGroups() error {
	groupInfos, err := gm.loadGroupInfos()
	if err != nil {
		return fmt.Errorf("failed to load group files: %w", err)
	}
	m := make(map[string]*protocol.Group)
	for _, g := range groupInfos {
		m[g.Name] = g
	}
	var errs []error
	for _, g := range gm.groups {
		info, exists := m[g.Name]
		delete(m, g.Name)
		if !exists {
			err = gm.deleteGroup(g)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to delete group %q: %w", g.Name, err))
			}
		} else {
			g.Group = info
		}
	}
	for _, g := range m {
		gm.groups = append(gm.groups, &group{Group: g})
		err = gm.m.tmpl.createTemplateDir(g.Name)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to create template directory: %w", err))
		}
	}
	return errors.Join(errs...)
}

func (gm *groupManager) loadGroupInfos() ([]*protocol.Group, error) {
	err := os.MkdirAll(gm.groupDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("cannot create directory: %w", err)
	}
	files, err := os.ReadDir(gm.groupDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read group files: %w", err)
	}

	var groups []*protocol.Group
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if !strings.HasSuffix(f.Name(), ".yaml") && !strings.HasSuffix(f.Name(), ".yml") {
			continue
		}
		fileName := path.Join(gm.groupDir, f.Name())
		b, err := os.ReadFile(fileName)
		if err != nil {
			return nil, fmt.Errorf("cannot read group file: %w", err)
		}
		var g protocol.Group
		err = yaml.Unmarshal(b, &g)
		if err != nil {
			return nil, fmt.Errorf("cannot parse group file at %q: %w", fileName, err)
		}
		err = g.Validate()
		if err != nil {
			return nil, fmt.Errorf("invalid group file %q: %w", fileName, err)
		}
		if f.Name() != g.Name+path.Ext(f.Name()) {
			return nil, fmt.Errorf("invalid group file: %q: file name must be %q", f.Name(), g.Name+".yaml")
		}
		groups = append(groups, &g)
	}

	return groups, nil
}

func (gm *groupManager) saveGroup(g *protocol.Group) error {
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

func (gm *groupManager) createGroup(g *protocol.Group) error {
	err := g.Validate()
	if err != nil {
		return fmt.Errorf("invalid group: %w", err)
	}
	for _, g := range gm.groups {
		if g.Name == g.Name {
			return fmt.Errorf("group %q already exists", g.Name)
		}
	}
	err = gm.saveGroup(g)
	if err != nil {
		return fmt.Errorf("cannot save group: %w", err)
	}
	gm.groups = append(gm.groups, &group{Group: g})
	err = gm.m.tmpl.createTemplateDir(g.Name)
	if err != nil {
		return fmt.Errorf("failed to create template directory: %w", err)
	}
	return nil
}

func (gm *groupManager) deleteGroup(g *group) error {
	gm.groups = common.DeleteItem(gm.groups, g)
	var errs []error
	for _, svc := range gm.services(g) {
		err := gm.m.sched.stopService(svc)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (gm *groupManager) getGroup(name string) *group {
	for _, g := range gm.groups {
		if g.Name == name {
			return g
		}
	}
	return nil
}

func (gm *groupManager) services(g *group) []*service {
	var services []*service
	for _, s := range gm.m.sched.services {
		if s.g == g {
			services = append(services, s)
		}
	}
	return services
}
