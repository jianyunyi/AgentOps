package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agentscope/internal/selfhost"
)

func awaitOperation(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("operation did not finish")
		return nil
	}
}

func awaitBarrier(t *testing.T, entered <-chan struct{}) {
	t.Helper()
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		t.Fatal("operation did not reach barrier")
	}
}

func TestStartRestoreMutualExclusion(t *testing.T) {
	for _, phase := range []string{"before-marker", "during-import"} {
		for _, failed := range []bool{false, true} {
			name := phase + "/success"
			if failed {
				name = phase + "/failure"
			}
			t.Run(name, func(t *testing.T) {
				env := setupRecovery(t)
				config, err := os.ReadFile(env)
				if err != nil {
					t.Fatal(err)
				}
				file, err := selfhost.SaveBackup(t.TempDir(), os.Getenv("AGENTOPS_BACKUP_PASSWORD"), config, func(w io.Writer) error {
					_, err := io.WriteString(w, "CREATE TABLE sample (id INT);\n")
					return err
				})
				if err != nil {
					t.Fatal(err)
				}
				entered, resume := make(chan struct{}), make(chan struct{})
				done := make(chan error, 1)
				old := recoveryCompose
				defer func() { recoveryCompose = old }()
				recoveryCompose = func(_ string, args []string, in io.Reader, out io.Writer) error {
					if (phase == "before-marker" && args[0] == "ps") || (phase == "during-import" && in != nil) {
						close(entered)
						<-resume
					}
					if in != nil {
						if failed {
							return errors.New("import failed")
						}
						return nil
					}
					if args[len(args)-1] == "DBSIZE" || strings.Contains(args[len(args)-1], "COUNT(*)") {
						_, err := io.WriteString(out, "0\n")
						return err
					}
					return nil
				}
				go func() { done <- recovery([]string{"restore", file, "--confirm"}, env) }()
				// Ensure the goroutine finishes before restoring the injected runner, even on failure.
				released, joined := false, false
				defer func() {
					if !released {
						close(resume)
					}
					if !joined {
						awaitOperation(t, done)
					}
				}()
				awaitBarrier(t, entered)
				if phase == "before-marker" {
					if _, err := os.Stat(recoveryIncompleteMarker(env)); !os.IsNotExist(err) {
						t.Fatalf("marker already exists: %v", err)
					}
				}
				called := false
				if err := start(env, func() error { called = true; return nil }); err == nil || !strings.Contains(err.Error(), ".recovery-lock") || called {
					t.Fatalf("start reached Docker during restore: called=%v err=%v", called, err)
				}
				// Release and join explicitly for post-restore assertions.
				close(resume)
				released = true
				err = awaitOperation(t, done)
				joined = true
				if (err != nil) != failed {
					t.Fatalf("restore result: %v", err)
				}
				if _, err := os.Stat(filepath.Join(filepath.Dir(env), ".recovery-lock")); !os.IsNotExist(err) {
					t.Fatalf("lock not released: %v", err)
				}
				called = false
				err = start(env, func() error { called = true; return nil })
				if failed {
					if err == nil || called || !strings.Contains(err.Error(), "restore is incomplete") {
						t.Fatalf("failed restore allowed startup: %v", err)
					}
				} else if err != nil || !called {
					t.Fatalf("successful restore blocked startup: %v", err)
				}
			})
		}
	}
}

func TestRestoreCannotEnterWhileStartRuns(t *testing.T) {
	env := setupRecovery(t)
	entered, resume := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	old := recoveryCompose
	defer func() { recoveryCompose = old }()
	called := false
	recoveryCompose = func(string, []string, io.Reader, io.Writer) error { called = true; return nil }
	go func() {
		done <- start(env, func() error { close(entered); <-resume; return errors.New("startup failed") })
	}()
	released, joined := false, false
	defer func() {
		if !released {
			close(resume)
		}
		if !joined {
			awaitOperation(t, done)
		}
	}()
	awaitBarrier(t, entered)
	if err := recovery([]string{"restore", "unused", "--confirm"}, env); err == nil || !strings.Contains(err.Error(), ".recovery-lock") || called {
		t.Fatalf("restore entered during startup: called=%v err=%v", called, err)
	}
	close(resume)
	released = true
	err := awaitOperation(t, done)
	joined = true
	if err == nil {
		t.Fatal("startup failure lost")
	}
	if err := start(env, func() error { return nil }); err != nil {
		t.Fatalf("failed startup leaked lock: %v", err)
	}
}

func TestStartRejectsStaleRecoveryLock(t *testing.T) {
	env := setupRecovery(t)
	lock := filepath.Join(filepath.Dir(env), ".recovery-lock")
	if err := os.Mkdir(lock, 0700); err != nil {
		t.Fatal(err)
	}
	if err := start(env, func() error { t.Fatal("startup with stale lock"); return nil }); err == nil {
		t.Fatal("stale lock ignored")
	}
	if _, err := os.Stat(lock); err != nil {
		t.Fatalf("another owner's lock removed: %v", err)
	}
}
