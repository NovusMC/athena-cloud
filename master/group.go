package main

import (
	"common"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path"
	"strings"
)

type group struct {
	info     *common.GroupInfo
	services []*service
}

type groupManager struct {
	groupDir string
	groups   []*group
}

func newGroupManager() (*groupManager, error) {
	gm := &groupManager{groupDir: "groups"}
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
		groups = append(groups, &group{info: &info})
	}

	gm.groups = groups
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
		if g.info.Name == info.Name {
			return fmt.Errorf("group %q already exists", info.Name)
		}
	}
	err = gm.saveGroup(info)
	if err != nil {
		return fmt.Errorf("cannot save group: %w", err)
	}
	gm.groups = append(gm.groups, &group{info: info})
	return nil
}
