// encrypt encrypts a skill or rule directory into a .astm module archive.
//
// File format: ASTM (Aster Module)
//
//	┌──────────────────────────────────────┐
//	│  Header (plaintext)                   │
//	│  ├── magic:    "ASTM" (4B)           │
//	│  ├── version:  uint16 (2B, big)      │
//	│  ├── hdr_len:  uint16 (2B, big)      │
//	│  └── manifest JSON (hdr_len bytes)   │
//	├──────────────────────────────────────┤
//	│  Assets (AES-256-GCM encrypted)       │
//	│  ├── nonce (12B)                     │
//	│  └── tar.gz of assets (encrypted)    │
//	└──────────────────────────────────────┘
//
// Usage:
//
//	go run cmd/encrypt/main.go \
//	  --in=modules/aster-sast/skills \
//	  --out=build/aster-sast-skills.astm \
//	  --module=aster-sast \
//	  --version=1.0.0 \
//	  --type=skills \
//	  --key=<hex-encoded-key>
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

const (
	magicBytes  = "ASTM"
	fileVersion = 1
)

// Manifest describes the encrypted module.
type Manifest struct {
	ModuleID    string `json:"module_id"`
	Version     string `json:"version"`
	AssetType   string `json:"asset_type"` // "skills" | "rules"
	Description string `json:"description,omitempty"`
	Count       int    `json:"count"`
}

func main() {
	inDir := flag.String("in", "", "input directory containing assets")
	outFile := flag.String("out", "", "output .astm file path")
	moduleID := flag.String("module", "", "module identifier (e.g. aster-sast)")
	version := flag.String("version", "1.0.0", "module version")
	assetType := flag.String("type", "skills", "asset type: skills | rules")
	desc := flag.String("desc", "", "module description")
	keyHex := flag.String("key", "", "AES-256 key (hex encoded, 64 chars)")
	flag.Parse()

	if *inDir == "" || *outFile == "" || *moduleID == "" || *keyHex == "" {
		fmt.Fprintf(os.Stderr, "Usage: encrypt --in=<dir> --out=<file> --module=<id> --key=<hex>\n")
		os.Exit(1)
	}

	key, err := hex.DecodeString(*keyHex)
	if err != nil || len(key) != 32 {
		fmt.Fprintf(os.Stderr, "invalid key: must be 64 hex chars (32 bytes)\n")
		os.Exit(1)
	}

	count := countFiles(*inDir)

	manifest := Manifest{
		ModuleID:    *moduleID,
		Version:     *version,
		AssetType:   *assetType,
		Description: *desc,
		Count:       count,
	}

	tarData, err := tarGzDir(*inDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tar failed: %v\n", err)
		os.Exit(1)
	}

	encrypted, err := encryptAESGCM(tarData, key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "encrypt failed: %v\n", err)
		os.Exit(1)
	}

	manifestJSON, _ := json.Marshal(manifest)
	if len(manifestJSON) > 65535 {
		fmt.Fprintf(os.Stderr, "manifest too large\n")
		os.Exit(1)
	}

	var buf bytes.Buffer
	buf.WriteString(magicBytes)

	ver := make([]byte, 2)
	binary.BigEndian.PutUint16(ver, fileVersion)
	buf.Write(ver)

	hdrLen := make([]byte, 2)
	binary.BigEndian.PutUint16(hdrLen, uint16(len(manifestJSON)))
	buf.Write(hdrLen)

	buf.Write(manifestJSON)
	buf.Write(encrypted)

	if err := os.MkdirAll(filepath.Dir(*outFile), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outFile, buf.Bytes(), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created %s (%d bytes, %d %s)\n", *outFile, buf.Len(), count, *assetType)
}

func encryptAESGCM(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func tarGzDir(dir string) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)

		info, err := d.Info()
		if err != nil {
			return err
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = rel

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if !d.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if _, err := tw.Write(data); err != nil {
				return err
			}
		}
		return nil
	})

	_ = tw.Close()
	_ = gw.Close()

	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func countFiles(dir string) int {
	n := 0
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		n++
		return nil
	})
	return n
}

// Ensure unused imports don't cause issues
var _ = sort.Strings
var _ = io.Discard
