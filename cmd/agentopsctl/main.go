package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"agentscope/internal/selfhost"
)

func commandArgs(arguments []string, envFile string) ([]string, error) {
	if len(arguments) != 1 { return nil, fmt.Errorf("usage: agentopsctl <configure|start|stop|status|logs|diagnose|backup>") }
	action := arguments[0]
	if action == "configure" || action == "diagnose" { return nil, nil }
	return selfhost.BuildComposeArgs(envFile, action, "")
}

func checkDocker() error {
	for _, args := range [][]string{{"version", "--format", "{{.Server.Version}}"}, {"compose", "version"}} {
		if output, err := exec.Command("docker", args...).CombinedOutput(); err != nil { return fmt.Errorf("Docker Desktop must be running: %s", string(output)) }
	}
	return nil
}

func waitReady(url string) error {
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		response, err := http.Get(url)
		if err == nil { response.Body.Close(); if response.StatusCode == http.StatusOK { return nil } }
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("API readiness check timed out")
}

func backupOutput() (*os.File, error) {
	directory := filepath.Join(os.Getenv("ProgramData"), "AgentOps", "backups")
	if err := os.MkdirAll(directory, 0700); err != nil { return nil, err }
	return os.Create(filepath.Join(directory, "agentscope-"+time.Now().UTC().Format("20060102T150405Z")+".sql"))
}

func main() {
	envFile := os.Getenv("AGENTOPS_ENV_FILE")
	if envFile == "" { envFile = filepath.Join(os.Getenv("ProgramData"), "AgentOps", "config", "agentops.env") }
	args, err := commandArgs(os.Args[1:], envFile)
	if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(2) }
	if os.Args[1] == "configure" { fmt.Println("Configure writes validated settings through the installer wizard."); return }
	if err := checkDocker(); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
	if os.Args[1] == "diagnose" { fmt.Println("Docker Desktop and Docker Compose are available."); return }
	command := exec.Command("docker", args...)
	if os.Args[1] == "backup" {
		output, err := backupOutput(); if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
		defer output.Close(); command.Stdout = output
	} else { command.Stdout, command.Stderr = os.Stdout, os.Stderr }
	if err := command.Run(); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
	if os.Args[1] == "start" { if err := waitReady("http://127.0.0.1:8080/health/ready"); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }; fmt.Println("AgentOps is ready at http://127.0.0.1:3300") }
}
