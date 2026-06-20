package filestore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/monoposer/lowcode-bpmn/internal/bpmnxml"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
)

func processesRoot(dir string) string {
	return filepath.Join(dir, "processes")
}

func processVersionPath(dir, tenantID, key string, version int) string {
	return filepath.Join(processesRoot(dir), tenantID, key, "v"+strconv.Itoa(version)+bpmnxml.DefinitionsFileSuffix)
}

func (s *Store) writeProcessXML(p *engine.DeployedProcess) error {
	if p == nil {
		return nil
	}
	path := processVersionPath(s.dir, p.TenantID, p.Key, p.Version)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create process dir: %w", err)
	}
	data, err := bpmnxml.Marshal(p.Definition)
	if err != nil {
		return fmt.Errorf("marshal bpmn xml: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) deleteProcessXML(tenantID, key string) error {
	dir := filepath.Join(processesRoot(s.dir), tenantID, key)
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *Store) loadProcessDefinitionsFromXML() error {
	root := processesRoot(s.dir)
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, bpmnxml.DefinitionsFileSuffix) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 3 {
			return nil
		}
		tenantID, key := parts[0], parts[1]
		base := filepath.Base(path)
		verStr := strings.TrimPrefix(strings.TrimSuffix(base, bpmnxml.DefinitionsFileSuffix), "v")
		version, err := strconv.Atoi(verStr)
		if err != nil {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		def, err := bpmnxml.Parse(raw)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		dp := &engine.DeployedProcess{
			TenantID:   tenantID,
			Key:        key,
			Version:    version,
			Name:       def.Name,
			Definition: def,
		}
		return s.mem.InsertProcessVersion(context.Background(), dp)
	})
}
