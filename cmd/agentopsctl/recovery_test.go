package main

import (
	"agentscope/internal/selfhost"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupRecovery(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("ProgramData", root)
	t.Setenv("AGENTOPS_BACKUP_PASSWORD", "test-recovery-password")
	path := filepath.Join(root, "AgentOps", "config", "agentops.env")
	image := "ghcr.io/test/image@sha256:" + strings.Repeat("a", 64)
	if err := selfhost.Configure(path, image, image); err != nil {
		t.Fatal(err)
	}
	return path
}
func TestRecoveryStopsBeforeBackupAndRejectsNonemptyRestore(t *testing.T) {
	env := setupRecovery(t)
	old := recoveryCompose
	defer func() { recoveryCompose = old }()
	recoveryCompose = func(_ string, args []string, _ io.Reader, out io.Writer) error {
		if args[0] == "ps" {
			io.WriteString(out, "running-container\n")
			return nil
		}
		t.Fatal("executed command while writers running")
		return nil
	}
	if err := recovery([]string{"backup"}, env); err == nil {
		t.Fatal("backed up with writers")
	}
	config, _ := os.ReadFile(env)
	file, err := selfhost.SaveBackup(t.TempDir(), os.Getenv("AGENTOPS_BACKUP_PASSWORD"), config, func(w io.Writer) error { _, err := io.WriteString(w, "CREATE TABLE sample (id INT);\n"); return err })
	if err != nil {
		t.Fatal(err)
	}
	recoveryCompose = func(_ string, args []string, in io.Reader, out io.Writer) error {
		if args[0] == "ps" || args[0] == "up" {
			return nil
		}
		if args[len(args)-1] == "DBSIZE" {
			io.WriteString(out, "0\n")
			return nil
		}
		if strings.Contains(args[len(args)-1], "COUNT(*)") {
			io.WriteString(out, "1\n")
			return nil
		}
		t.Fatal("import into nonempty database")
		return nil
	}
	if err = recovery([]string{"restore", file, "--confirm"}, env); err == nil {
		t.Fatal("accepted nonempty DB")
	}
}
func TestRecoveryFreshConfigAndFailedImport(t *testing.T) {
	env := setupRecovery(t)
	config, _ := os.ReadFile(env)
	sql := []byte("CREATE TABLE sample (id INT);\n")
	file, err := selfhost.SaveBackup(t.TempDir(), os.Getenv("AGENTOPS_BACKUP_PASSWORD"), config, func(w io.Writer) error { _, e := w.Write(sql); return e })
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(env)
	old := recoveryCompose
	defer func() { recoveryCompose = old }()
	imported := false
	recoveryCompose = func(_ string, args []string, in io.Reader, out io.Writer) error {
		if args[0] == "ps" || args[0] == "up" {
			return nil
		}
		if args[len(args)-1] == "DBSIZE" {
			io.WriteString(out, "0\n")
			return nil
		}
		if strings.Contains(args[len(args)-1], "COUNT(*)") {
			io.WriteString(out, "0\n")
			return nil
		}
		b, _ := io.ReadAll(in)
		if !bytes.Equal(b, sql) {
			t.Fatal("SQL mismatch")
		}
		imported = true
		return errors.New("import failed")
	}
	if err = recovery([]string{"restore", file, "--confirm"}, env); err == nil || !strings.Contains(err.Error(), "partial destination") {
		t.Fatal("failed import not reported", err)
	}
	restored, _ := os.ReadFile(env)
	if !imported || !bytes.Equal(restored, config) {
		t.Fatal("original config not restored")
	}
}
func TestRestoreRequiresConfirmation(t *testing.T) {
	for _, args := range [][]string{{"restore", "file"}, {"restore", "file", "--force"}} {
		if _, err := commandArgs(args, "env"); err == nil {
			t.Fatal("missing confirmation accepted")
		}
	}
}
