// Package vaultskills provides skill loading from encrypted .astm modules
// with automatic fallback to embedded filesystem for development mode.
package vaultskills

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"aster/internal/module"
	"aster/internal/service"

	skillspkg "aster/skills"
)

// LoadSkillsFromVault loads skills from encrypted .astm files in the given
// directory. Falls back to the embedded skills if no .astm files are found.
//
// Returns the skill service populated with all loaded skills.
func LoadSkillsFromVault(ctx context.Context, modulesDir string, fallbackFS fs.FS) (*service.SkillService, int, error) {
	svc := service.NewSkillServiceWithMemory()

	entries, err := os.ReadDir(modulesDir)
	if err != nil || len(entries) == 0 {
		// No modules directory or empty — fall back to embedded
		return loadFallback(ctx, svc, fallbackFS)
	}

	total := 0
	hasModules := false
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".astm") {
			continue
		}
		hasModules = true
		modPath := filepath.Join(modulesDir, entry.Name())

		mod, err := module.Load(modPath)
		if err != nil {
			return nil, total, fmt.Errorf("load %s: %w", entry.Name(), err)
		}
		if mod.Manifest.AssetType != "skills" {
			continue
		}

		for _, asset := range mod.Assets {
			sk, err := service.ParseSkillMarkdown(string(asset.Content))
			if err != nil {
				continue // skip malformed skills
			}
			sk.Source = mod.Manifest.ModuleID
			if err := svc.ImportSkill(ctx, sk); err != nil {
				continue
			}
			total++
		}
	}

	if !hasModules {
		return loadFallback(ctx, svc, fallbackFS)
	}

	return svc, total, nil
}

func loadFallback(ctx context.Context, svc *service.SkillService, fallbackFS fs.FS) (*service.SkillService, int, error) {
	if fallbackFS == nil {
		fallbackFS = skillspkg.EmbeddedSkills
	}
	n, err := svc.ImportSkillsFromFSWithSource(ctx, fallbackFS, "embedded-dev", "")
	if err != nil {
		return nil, 0, fmt.Errorf("fallback skills load: %w", err)
	}
	return svc, n, nil
}
