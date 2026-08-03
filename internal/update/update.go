// Package update provides a content-update mechanism for Aster's encrypted
// .astm content packages (skills/rules). It discovers staged updates, compares
// them against installed modules, and applies newer packages into the active
// modules directory.
//
// Design notes:
//   - This is an offline appliance. Updates are staged on disk (or fetched by a
//     sidecar) under a configurable directory, then validated and applied here.
//   - "License binding" is inherent: an .astm only decrypts with the server's
//     vault key, so only genuine packages can ever be applied.
package update

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"aster/internal/module"
)

// VersionCompare orders two dot-separated version strings (e.g. "1.2.0").
// Returns -1 if a < b, 0 if equal, +1 if a > b. It compares numerically per
// segment so "1.10.0" > "1.9.0".
func VersionCompare(a, b string) int {
	as := strings.Split(strings.TrimPrefix(a, "v"), ".")
	bs := strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		var av, bv int
		if i < len(as) {
			fmt.Sscanf(as[i], "%d", &av)
		}
		if i < len(bs) {
			fmt.Sscanf(bs[i], "%d", &bv)
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

// UpdateInfo describes a staged content package that could be installed.
type UpdateInfo struct {
	ModuleID    string `json:"module_id"`
	Version     string `json:"version"`
	AssetType   string `json:"asset_type"`
	Description string `json:"description,omitempty"`
	Count       int    `json:"count"`
	Filename    string `json:"filename"`
}

// InstalledInfo describes a module already installed in the active directory.
type InstalledInfo struct {
	ModuleID  string `json:"module_id"`
	Version   string `json:"version"`
	AssetType string `json:"asset_type"`
	Count     int    `json:"count"`
}

// installedByID is inlined into installedVersion for readability.

// availableStaged scans the staging directory for .astm files and parses each.
func availableStaged(stageDir string) ([]UpdateInfo, error) {
	out := []UpdateInfo{}
	entries, err := os.ReadDir(stageDir)
	if err != nil {
		return nil, err // caller decides how to handle missing dir
	}
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".astm") {
			continue
		}
		p := filepath.Join(stageDir, e.Name())
		mod, err := module.Load(p)
		if err != nil {
			// Skip unreadable staged files; surface in debug only.
			continue
		}
		out = append(out, UpdateInfo{
			ModuleID:    mod.Manifest.ModuleID,
			Version:     mod.Manifest.Version,
			AssetType:   mod.Manifest.AssetType,
			Description: mod.Manifest.Description,
			Count:       mod.Manifest.Count,
			Filename:    e.Name(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ModuleID < out[j].ModuleID })
	return out, nil
}

// installedInspects the active modules directory for currently installed modules.
func installedInspect(modulesDir string) ([]InstalledInfo, error) {
	out := []InstalledInfo{}
	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".astm") {
			continue
		}
		mod, err := module.Load(filepath.Join(modulesDir, e.Name()))
		if err != nil {
			continue
		}
		out = append(out, InstalledInfo{
			ModuleID:  mod.Manifest.ModuleID,
			Version:   mod.Manifest.Version,
			AssetType: mod.Manifest.AssetType,
			Count:     mod.Manifest.Count,
		})
	}
	return out, nil
}

// installedByID returns the installed version of a module id, or "" if absent.
func installedVersion(modulesDir, moduleID string) (string, error) {
	inst, err := installedInspect(modulesDir)
	if err != nil {
		return "", err
	}
	for _, i := range inst {
		if i.ModuleID == moduleID {
			return i.Version, nil
		}
	}
	return "", nil
}

// CheckResult is the response of a check: installed modules plus available
// staged updates that are newer than what's installed.
type CheckResult struct {
	Installed []InstalledInfo `json:"installed"`
	Updates   []UpdateInfo    `json:"updates"`
}

// Check compares staged updates against installed modules.
func Check(stageDir, modulesDir string) (*CheckResult, error) {
	res := &CheckResult{Installed: []InstalledInfo{}, Updates: []UpdateInfo{}}

	if inst, err := installedInspect(modulesDir); err == nil {
		res.Installed = inst
	}

	staged, err := availableStaged(stageDir)
	if err != nil {
		// Missing staging dir simply means no pending updates.
		if os.IsNotExist(err) {
			return res, nil
		}
		return nil, err
	}

	for _, u := range staged {
		cur, _ := installedVersion(modulesDir, u.ModuleID)
		if cur == "" || VersionCompare(u.Version, cur) > 0 {
			res.Updates = append(res.Updates, u)
		}
	}
	return res, nil
}

// Apply installs a staged update (matching filename) into the active modules
// directory. It validates the package decodes, refuses downgrades, and copies
// the file into place (replacing any same-id module). Returns the installed info.
func Apply(stageDir, modulesDir, filename string) (*InstalledInfo, error) {
	if !strings.EqualFold(filepath.Ext(filename), ".astm") {
		return nil, fmt.Errorf("not a module file: %s", filename)
	}
	src := filepath.Join(stageDir, filename)
	data, err := os.ReadFile(src)
	if err != nil {
		return nil, fmt.Errorf("read staged module: %w", err)
	}
	mod, err := module.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("invalid module package: %w", err)
	}

	// Version check against currently installed same-id module.
	cur, _ := installedVersion(modulesDir, mod.Manifest.ModuleID)
	if cur != "" && VersionCompare(mod.Manifest.Version, cur) <= 0 {
		return nil, fmt.Errorf("refusing downgrade: installed %s >= candidate %s", cur, mod.Manifest.Version)
	}

	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		return nil, err
	}

	// Remove any existing active module with the same id before installing.
	if entries, err := os.ReadDir(modulesDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".astm") {
				continue
			}
			old, lerr := module.Load(filepath.Join(modulesDir, e.Name()))
			if lerr == nil && old.Manifest.ModuleID == mod.Manifest.ModuleID {
				_ = os.Remove(filepath.Join(modulesDir, e.Name()))
			}
		}
	}

	dst := filepath.Join(modulesDir, filename)
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return nil, fmt.Errorf("write active module: %w", err)
	}

	// Best-effort sync to persist the write (skipped where unsupported).
	_ = syncDirIfPossible(filepath.Dir(dst))

	return &InstalledInfo{
		ModuleID:  mod.Manifest.ModuleID,
		Version:   mod.Manifest.Version,
		AssetType: mod.Manifest.AssetType,
		Count:     mod.Manifest.Count,
	}, nil
}

// syncDirIfPossible fsyncs a directory on platforms where that is supported.
var syncDirIfPossible = func(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
