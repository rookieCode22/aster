// Package vaultrules provides rule loading from encrypted .astm modules.
package vaultrules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aster/internal/module"

	semgrep_rules "aster/semgrep-rules"
)

// LoadRulesFromVault loads semgrep rules from encrypted .astm files.
// Falls back to embedded rules if no .astm files are found.
// Returns the directory path where rules are extracted (for semgrep CLI usage).
func LoadRulesFromVault(modulesDir string, targetDir string) (string, error) {
	entries, err := os.ReadDir(modulesDir)
	if err != nil || len(entries) == 0 {
		return extractEmbedded(targetDir)
	}

	hasModules := false
	extracted := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".astm") {
			continue
		}
		hasModules = true
		modPath := filepath.Join(modulesDir, entry.Name())

		mod, err := module.Load(modPath)
		if err != nil {
			return "", fmt.Errorf("load %s: %w", entry.Name(), err)
		}
		if mod.Manifest.AssetType != "rules" {
			continue
		}

		for _, asset := range mod.Assets {
			outPath := filepath.Join(targetDir, asset.Path)
			if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
				return "", fmt.Errorf("mkdir %s: %w", filepath.Dir(outPath), err)
			}
			if err := os.WriteFile(outPath, asset.Content, 0o644); err != nil {
				return "", fmt.Errorf("write %s: %w", outPath, err)
			}
			extracted++
		}
	}

	if !hasModules {
		return extractEmbedded(targetDir)
	}

	return targetDir, nil
}

func extractEmbedded(targetDir string) (string, error) {
	dir, err := semgrep_rules.ExtractRulesDir()
	if err != nil {
		return "", fmt.Errorf("extract embedded rules: %w", err)
	}
	return dir, nil
}
