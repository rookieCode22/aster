package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"aster/internal/module"
	"aster/internal/vault"
)

// buildASTM builds a valid .astm package bytes for the given manifest/assets.
func buildASTM(t *testing.T, id, version, assetType string) []byte {
	t.Helper()

	// Build asset tar.gz
	var tarBuf bytes.Buffer
	gz := gzip.NewWriter(&tarBuf)
	tw := tar.NewWriter(gz)
	body := []byte("# skill\n\ncontent version " + version)
	hdr := &tar.Header{Name: id + "/SKILL.md", Mode: 0o644, Size: int64(len(body))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gz.Close()

	enc, err := vault.Encrypt(tarBuf.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	manifest, _ := json.Marshal(module.Manifest{
		ModuleID:  id,
		Version:   version,
		AssetType: assetType,
		Count:     1,
	})

	var out bytes.Buffer
	out.WriteString("ASTM")                    // magic
	binary.Write(&out, binary.BigEndian, uint16(1))
	binary.Write(&out, binary.BigEndian, uint16(len(manifest)))
	out.Write(manifest)
	out.Write(enc)
	return out.Bytes()
}

func writeASTM(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestVersionCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.2.0", "1.9.0", -1},
		{"1.10.0", "1.9.0", 1}, // numeric, not lexicographic
		{"2.0.0", "1.99.0", 1},
		{"v1.2", "1.2", 0},
		{"1.0", "1.0.0", 0},
	}
	for _, c := range cases {
		if got := VersionCompare(c.a, c.b); got != c.want {
			t.Errorf("VersionCompare(%s,%s)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestCheckAndApply(t *testing.T) {
	stage := t.TempDir()
	mods := t.TempDir()

	// installed v1
	writeASTM(t, filepath.Join(mods, "code-audit.astm"), buildASTM(t, "code-audit", "1.0.0", "skills"))
	// staged: newer, same, and unrelated-new
	writeASTM(t, filepath.Join(stage, "code-audit-1.1.0.astm"), buildASTM(t, "code-audit", "1.1.0", "skills"))
	writeASTM(t, filepath.Join(stage, "code-audit-1.0.0.astm"), buildASTM(t, "code-audit", "1.0.0", "skills")) // equal -> not update
	writeASTM(t, filepath.Join(stage, "waf-rules-2.0.astm"), buildASTM(t, "waf-rules", "2.0.0", "rules"))

	res, err := Check(stage, mods)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Installed) != 1 || res.Installed[0].ModuleID != "code-audit" {
		t.Fatalf("installed wrong: %+v", res.Installed)
	}
	if len(res.Updates) != 2 {
		t.Fatalf("expected 2 updates (new+unrelated), got %d: %+v", len(res.Updates), res.Updates)
	}

	// apply the newer one
	got, err := Apply(stage, mods, "code-audit-1.1.0.astm")
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "1.1.0" {
		t.Fatalf("apply version = %s want 1.1.0", got.Version)
	}

	// after apply, installed is now 1.1.0, and code-audit-1.1.0 no longer an update
	res2, _ := Check(stage, mods)
	for _, u := range res2.Updates {
		if u.ModuleID == "code-audit" {
			t.Fatalf("code-audit still listed as update after apply: %+v", u)
		}
	}

	// downgrade refused
	_, err = Apply(stage, mods, "code-audit-1.0.0.astm")
	if err == nil {
		t.Fatal("expected downgrade refusal")
	}
}

func TestApplyInvalidPackage(t *testing.T) {
	stage := t.TempDir()
	mods := t.TempDir()
	writeASTM(t, filepath.Join(stage, "bad.astm"), []byte("not a real module"))
	if _, err := Apply(stage, mods, "bad.astm"); err == nil {
		t.Fatal("expected error for invalid package")
	}
	// non-astm filename rejected
	if _, err := Apply(stage, mods, "evil.sh"); err == nil {
		t.Fatal("expected error for non-astm filename")
	}
}
