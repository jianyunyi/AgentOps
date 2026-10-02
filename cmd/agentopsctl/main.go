package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"agentscope/internal/selfhost"
)

func commandArgs(arguments []string, envFile string) ([]string, error) {
	if len(arguments) == 3 && arguments[0] == "configure" {
		return nil, nil
	}
	if len(arguments) == 3 && arguments[0] == "restore" && arguments[2] == "--confirm" {
		return nil, nil
	}
	if len(arguments) == 2 && arguments[0] == "verify-backup" {
		return nil, nil
	}
	if len(arguments) != 1 {
		return nil, fmt.Errorf("usage: agentopsctl <configure [api-digest web-digest]|start|stop|status|logs|diagnose|backup|verify-backup FILE|restore FILE --confirm>")
	}
	action := arguments[0]
	if action == "configure" || action == "diagnose" {
		return nil, nil
	}
	return selfhost.BuildComposeArgs(envFile, action, "")
}

func checkDocker() error {
	for _, args := range [][]string{{"version", "--format", "{{.Server.Version}}"}, {"compose", "version"}} {
		if output, err := exec.Command("docker", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("Docker Desktop must be running: %s", string(output))
		}
	}
	return nil
}

func waitReady(url string) error {
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		response, err := http.Get(url)
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("API readiness check timed out")
}

func recoveryIncompleteMarker(env string) string {\n\treturn filepath.Join(filepath.Dir(env), ".restore-incomplete")\n}\n\nfunc ensureStartAllowed(env string) error {\n\tif _, err := os.Stat(recoveryIncompleteMarker(env)); err == nil {\n\t\treturn fmt.Errorf("restore is incomplete; discard the partial destination and retry restore against fresh MySQL/Redis volumes before start")\n\t} else if !os.IsNotExist(err) {\n\t\treturn fmt.Errorf("cannot verify restore state: %w", err)\n\t}\n\treturn nil\n}\n\nfunc main() {
	envFile := os.Getenv("AGENTOPS_ENV_FILE")
	if envFile == "" {
		envFile = filepath.Join(os.Getenv("ProgramData"), "AgentOps", "config", "agentops.env")
	}
	envFile, err := filepath.Abs(envFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	args, err := commandArgs(os.Args[1:], envFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if os.Args[1] == "configure" {
		api, web := os.Getenv("AGENTOPS_API_IMAGE"), os.Getenv("AGENTOPS_WEB_IMAGE")
		if len(os.Args) == 4 {
			api, web = os.Args[2], os.Args[3]
		}
		if err := selfhost.Configure(envFile, api, web); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("Configuration validated and saved; existing secrets preserved.")
		return
	}
	if os.Args[1] == "verify-backup" {
		if _, _, err := selfhost.ReadBackup(os.Args[2], os.Getenv("AGENTOPS_BACKUP_PASSWORD")); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("Backup authentication, checksums and configuration verified.")
		return
	}
	if os.Args[1] == "restore" {
		absolute, e := filepath.Abs(os.Args[2])
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
		os.Args[2] = absolute
	}
	root, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	root = filepath.Dir(root)
	if err = os.Chdir(root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := checkDocker(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if os.Args[1] == "diagnose" {
		fmt.Println("Docker Desktop and Docker Compose are available.")
		return
	}
	if os.Args[1] == "backup" || os.Args[1] == "restore" {
		if err := recovery(os.Args[1:], envFile); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	command := exec.Command("docker", args...)
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if os.Args[1] == "start" {
		if err := waitReady("http://127.0.0.1:8080/health/ready"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("AgentOps is ready at http://127.0.0.1:3300")
	}
}

func composeRun(env string, args []string, in io.Reader, out io.Writer) error {
	base, err := selfhost.BuildComposeArgs(env, "status", "")
	if err != nil {
		return err
	}
	base = base[:len(base)-1]
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", append(base, args...)...)
	cmd.Stdin = in
	cmd.Stdout = out
	// Docker/MySQL error text can contain credentials. Return a stable, secret-free error.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("Docker recovery command failed: %w", err)
	}
	return nil
}

var recoveryCompose = composeRun

func requireStopped(env string) error {
	var b bytes.Buffer
	if err := recoveryCompose(env, []string{"ps", "--status", "running", "-q", "api", "worker", "web"}, nil, &b); err != nil {
		return err
	}
	if strings.TrimSpace(b.String()) != "" {
		return fmt.Errorf("stop api, worker and web before backup/restore; prohibit external database writers and schema changes")
	}
	return nil
}
func recovery(args []string, env string) error {
	password := os.Getenv("AGENTOPS_BACKUP_PASSWORD")
	if len(password) < 16 {
		return fmt.Errorf("set AGENTOPS_BACKUP_PASSWORD (at least 16 characters); keep it separately from the backup")
	}
	dir := filepath.Dir(env)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	lock := filepath.Join(dir, ".recovery-lock")
	if err := os.Mkdir(lock, 0700); err != nil {
		return fmt.Errorf("another recovery operation is active (or a stale .recovery-lock needs operator inspection): %w", err)
	}
	defer os.Remove(lock)
	if args[0] == "backup" {
		if _, err := selfhost.LoadConfig(env); err != nil {
			return err
		}
		if err := requireStopped(env); err != nil {
			return err
		}
		if err := recoveryCompose(env, []string{"up", "-d", "--wait", "mysql", "redis"}, nil, os.Stdout); err != nil {
			return err
		}
		config, err := os.ReadFile(env)
		if err != nil {
			return err
		}
		path, err := selfhost.SaveBackup(filepath.Join(os.Getenv("ProgramData"), "AgentOps", "backups"), password, config, func(w io.Writer) error {
			return recoveryCompose(env, []string{"exec", "-T", "mysql", "sh", "-c", `MYSQL_PWD="$MYSQL_PASSWORD" exec mysqldump --single-transaction --quick --no-tablespaces --set-gtid-purged=OFF --hex-blob -u agentscope agentscope`}, nil, w)
		})
		if err != nil {
			return err
		}
		fmt.Println("Verified encrypted backup:", path)
		return nil
	}
	sql, config, err := selfhost.ReadBackup(args[1], password)
	if err != nil {
		return err
	}
	// Restore requires the original config. Never replace an existing deployment's keys.
	existing, err := os.ReadFile(env)
	if os.IsNotExist(err) {
		if e := selfhost.WriteNewConfig(env, config); e != nil {
			return e
		}
	} else if err != nil {
		return err
	} else if !bytes.Equal(existing, config) {
		return fmt.Errorf("restore requires the exact original agentops.env; existing configuration was not changed")
	}
	if err = requireStopped(env); err != nil {
		return err
	}
	if err = recoveryCompose(env, []string{"up", "-d", "--wait", "mysql", "redis"}, nil, os.Stdout); err != nil {
		return err
	}
	var count bytes.Buffer
	if err = recoveryCompose(env, []string{"exec", "-T", "mysql", "sh", "-c", `MYSQL_PWD="$MYSQL_PASSWORD" exec mysql -N -B -u agentscope agentscope -e "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='agentscope'"`}, nil, &count); err != nil {
		return err
	}
	if strings.TrimSpace(count.String()) != "0" {
		return fmt.Errorf("restore refused: destination database is not empty")
	}
	var redisState bytes.Buffer
	if err = recoveryCompose(env, []string{"exec", "-T", "redis", "redis-cli", "DBSIZE"}, nil, &redisState); err != nil {
		return err
	}
	if strings.TrimSpace(redisState.String()) != "0" {
		return fmt.Errorf("restore refused: destination Redis is not empty; use fresh volumes")
	}
	if err = recoveryCompose(env, []string{"exec", "-T", "mysql", "sh", "-c", `MYSQL_PWD="$MYSQL_PASSWORD" exec mysql -u agentscope agentscope`}, bytes.NewReader(sql), os.Stdout); err != nil {
		return fmt.Errorf("restore failed; discard the partial destination and retry against a fresh database: %w", err)
	}
	fmt.Println("Database restored with original keys. Redis ephemeral state resets on a fresh instance. Run start and verify login, Agent credentials and events.")
	return nil
}
