package main

import (\n\t"os"\n\t"path/filepath"\n\t"testing"\n)

func TestCommandArgsRejectUnknownAction(t *testing.T) {
	if _, err := commandArgs([]string{"destroy"}, "config.env"); err == nil {
		t.Fatal("expected rejection")
	}
}

func TestCommandArgsMapsStart(t *testing.T) {
	args, err := commandArgs([]string{"start"}, "config.env")
	if err != nil {
		t.Fatal(err)
	}
	if len(args) == 0 || args[len(args)-1] != "-d" {
		t.Fatalf("unexpected args %v", args)
	}
}

func TestEnsureStartAllowedBlocksIncompleteRestore(t *testing.T) {
	env := filepath.Join(t.TempDir(), "config", "agentops.env")
	if err := os.MkdirAll(filepath.Dir(env), 0700); err != nil { t.Fatal(err) }
	if err := os.WriteFile(recoveryIncompleteMarker(env), []byte("incomplete"), 0600); err != nil { t.Fatal(err) }
	if err := ensureStartAllowed(env); err == nil { t.Fatal("start allowed after incomplete restore") }
	if err := os.Remove(recoveryIncompleteMarker(env)); err != nil { t.Fatal(err) }
	if err := ensureStartAllowed(env); err != nil { t.Fatal(err) }
}
