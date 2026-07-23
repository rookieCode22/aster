package module

import (
	"strings"
	"testing"
)

func TestLoadSkillsModule(t *testing.T) {
	// This .astm was encrypted with the current vault key
	m, err := Load("../../build/aster-core-skills.astm")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if err := m.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	t.Logf("Module: %s v%s", m.Manifest.ModuleID, m.Manifest.Version)
	t.Logf("Type: %s, Count: %d, Assets: %d",
		m.Manifest.AssetType, m.Manifest.Count, len(m.Assets))

	if m.Manifest.AssetType != "skills" {
		t.Errorf("asset_type = %q, want skills", m.Manifest.AssetType)
	}
	if len(m.Assets) != m.Manifest.Count {
		t.Errorf("asset count mismatch: got %d assets, manifest claims %d", len(m.Assets), m.Manifest.Count)
	}

	// Verify we can find some known skills
	found := make(map[string]bool)
	for _, a := range m.Assets {
		found[a.Path] = true
	}

	expected := []string{
		"code-audit/sast-scan/SKILL.md",
		"pentest/sql-injection-comprehensive/SKILL.md",
		"code-audit/secret-detection/SKILL.md",
	}
	for _, exp := range expected {
		if !found[exp] {
			t.Errorf("expected asset %q not found in module", exp)
		} else {
			t.Logf("  ✓ found %s", exp)
		}
	}
}

func TestLoadRulesModule(t *testing.T) {
	m, err := Load("../../build/aster-core-rules.astm")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if err := m.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	t.Logf("Module: %s v%s", m.Manifest.ModuleID, m.Manifest.Version)
	t.Logf("Type: %s, Count: %d, Assets: %d",
		m.Manifest.AssetType, m.Manifest.Count, len(m.Assets))

	// Check one known rule
	for _, a := range m.Assets {
		if strings.HasSuffix(a.Path, ".yaml") && strings.Contains(string(a.Content), "sql-injection") {
			t.Logf("  ✓ found sql-injection rule: %s (%d bytes)", a.Path, len(a.Content))
			return
		}
	}
	t.Error("no sql-injection rule found in module")
}
