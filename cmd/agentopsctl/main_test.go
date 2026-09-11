package main

import "testing"

func TestCommandArgsRejectUnknownAction(t *testing.T) {
	if _, err := commandArgs([]string{"destroy"}, "config.env"); err == nil { t.Fatal("expected rejection") }
}

func TestCommandArgsMapsStart(t *testing.T) {
	args, err := commandArgs([]string{"start"}, "config.env")
	if err != nil { t.Fatal(err) }
	if len(args) == 0 || args[len(args)-1] != "-d" { t.Fatalf("unexpected args %v", args) }
}
