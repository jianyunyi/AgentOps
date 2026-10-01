package selfhost

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testConfig(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join(t.TempDir(), "agentops.env")
	image := "ghcr.io/test/image@sha256:" + strings.Repeat("a", 64)
	if err := Configure(path, image, image); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func TestConfigurePreservesSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "agentops.env")
	image := "ghcr.io/test/image@sha256:" + strings.Repeat("a", 64)
	if err := Configure(path, image, image); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	if err := Configure(path, "", ""); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("configuration rotated")
	}
	if _, err := LoadConfig(path); err != nil {
		t.Fatal(err)
	}
}
func TestConfigureRejectsBadImagesWithoutFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "env")
	if err := Configure(path, "latest", "latest"); err == nil {
		t.Fatal("accepted mutable image")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("invalid config written")
	}
}
func TestBackupRoundTripAndAuthentication(t *testing.T) {
	dir := t.TempDir()
	config := testConfig(t)
	password := "test-only-password-123"
	sql := []byte("CREATE TABLE sample (id INT); INSERT INTO sample VALUES (1);\n")
	dump := func(w io.Writer) error { _, e := w.Write(sql); return e }
	path, err := SaveBackup(dir, password, config, dump)
	if err != nil {
		t.Fatal(err)
	}
	got, c, err := ReadBackup(path, password)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, sql) || !bytes.Equal(c, config) {
		t.Fatal("recovery differs")
	}
	another, err := SaveBackup(dir, password, config, dump)
	if err != nil || another == path {
		t.Fatal("backup collision", err)
	}
	if _, _, err = ReadBackup(path, "wrong-password-long"); err == nil {
		t.Fatal("accepted wrong password")
	}
	b, _ := os.ReadFile(path)
	b[len(b)-1] ^= 1
	os.WriteFile(path, b, 0600)
	if _, _, err = ReadBackup(path, password); err == nil {
		t.Fatal("accepted tampered backup")
	}
}
func TestBackupFailureLeavesNoArtifact(t *testing.T) {
	for _, dump := range []func(io.Writer) error{func(w io.Writer) error { w.Write([]byte("partial")); return errors.New("dump failed") }, func(io.Writer) error { return nil }} {
		dir := t.TempDir()
		if _, err := SaveBackup(dir, "test-password-123456", testConfig(t), dump); err == nil {
			t.Fatal("expected failure")
		}
		files, _ := os.ReadDir(dir)
		if len(files) != 0 {
			t.Fatal("left failed artifact")
		}
	}
}
func TestBoundedDumpAndDuplicateConfig(t *testing.T) {
	w := &limitedWriter{io.Discard, 1}
	if _, err := w.Write([]byte("too big")); err == nil {
		t.Fatal("unbounded dump")
	}
	if _, err := ParseConfig(strings.NewReader("MYSQL_PASSWORD=\nMYSQL_PASSWORD=x\n"), false); err == nil {
		t.Fatal("duplicate accepted")
	}
}
