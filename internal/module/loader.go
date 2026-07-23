// module provides loading and registration of encrypted .astm module files.
package module

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"aster/internal/vault"
)

const (
	magicStr  = "ASTM"
	minHeader = 4 + 2 + 2 // magic + version + hdr_len
)

// Manifest describes a loaded module's metadata.
type Manifest struct {
	ModuleID    string `json:"module_id"`
	Version     string `json:"version"`
	AssetType   string `json:"asset_type"` // "skills" | "rules"
	Description string `json:"description,omitempty"`
	Count       int    `json:"count"`
}

// Asset represents a single file inside a module archive.
type Asset struct {
	Path    string // relative path inside archive (e.g. "code-audit/sast-scan/SKILL.md")
	Content []byte // file content
}

// Module is a loaded and decrypted module ready for registration.
type Module struct {
	Manifest Manifest
	Assets   []Asset // decrypted, in-memory
}

// Load reads and decrypts a .astm file, returning the parsed module.
func Load(path string) (*Module, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return Decode(data)
}

// Decode parses and decrypts raw .astm bytes.
func Decode(data []byte) (*Module, error) {
	if len(data) < minHeader {
		return nil, fmt.Errorf("file too short: %d bytes", len(data))
	}

	// Magic
	if string(data[:4]) != magicStr {
		return nil, fmt.Errorf("not a valid .astm file (bad magic)")
	}

	// Version
	ver := binary.BigEndian.Uint16(data[4:6])
	if ver != 1 {
		return nil, fmt.Errorf("unsupported version: %d", ver)
	}

	// Header length
	hdrLen := int(binary.BigEndian.Uint16(data[6:8]))
	manifestStart := 8
	manifestEnd := manifestStart + hdrLen
	if manifestEnd > len(data) {
		return nil, fmt.Errorf("manifest extends past file end")
	}

	// Manifest
	var manifest Manifest
	if err := json.Unmarshal(data[manifestStart:manifestEnd], &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	// Decrypt assets
	encrypted := data[manifestEnd:]
	if len(encrypted) < 12 {
		return nil, fmt.Errorf("encrypted data too short")
	}

	plaintext, err := vault.Decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	// Untar
	assets, err := untarGz(plaintext)
	if err != nil {
		return nil, fmt.Errorf("untar: %w", err)
	}

	return &Module{
		Manifest: manifest,
		Assets:   assets,
	}, nil
}

func untarGz(data []byte) ([]Asset, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	var assets []Asset

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag == tar.TypeDir {
			continue
		}

		content, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", hdr.Name, err)
		}

		assets = append(assets, Asset{
			Path:    hdr.Name,
			Content: content,
		})
	}

	return assets, nil
}

// Validate checks manifest fields are coherent with asset contents.
func (m *Module) Validate() error {
	if m.Manifest.ModuleID == "" {
		return fmt.Errorf("module_id is empty")
	}
	if m.Manifest.Version == "" {
		return fmt.Errorf("version is empty")
	}
	at := strings.ToLower(m.Manifest.AssetType)
	if at != "skills" && at != "rules" {
		return fmt.Errorf("unknown asset_type: %s", m.Manifest.AssetType)
	}
	if len(m.Assets) == 0 {
		return fmt.Errorf("no assets in module")
	}
	return nil
}
