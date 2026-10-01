package selfhost

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Opt-in CI test against a real MySQL 8.4 instance. Test credentials are disposable.
func TestMySQLRecovery(t *testing.T) {
	if os.Getenv("AGENTOPS_RECOVERY_INTEGRATION") != "1" {
		t.Skip("set AGENTOPS_RECOVERY_INTEGRATION=1")
	}
	name := fmt.Sprintf("agentops-recovery-%d", time.Now().UnixNano())
	docker := func(args ...string) []byte {
		t.Helper()
		out, err := exec.Command("docker", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("docker failed: %v %s", err, out)
		}
		return out
	}
	docker("run", "-d", "--name", name, "-e", "MYSQL_ROOT_PASSWORD=test-only-password", "-e", "MYSQL_DATABASE=agentscope", "-e", "MYSQL_USER=agentscope", "-e", "MYSQL_PASSWORD=test-only-password", "mysql:8.4")
	defer exec.Command("docker", "rm", "-f", name).Run()
	deadline := time.Now().Add(2 * time.Minute)
	for {
		err := exec.Command("docker", "exec", name, "sh", "-c", `MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot -e 'SELECT 1'`).Run()
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("mysql readiness timed out")
		}
		time.Sleep(time.Second)
	}
	config := testConfig(t)
	c, err := ParseConfig(bytes.NewReader(config), false)
	if err != nil {
		t.Fatal(err)
	}
	key, _ := base64.StdEncoding.DecodeString(c.Values["AGENT_SIGNING_ENCRYPTION_KEY"])
	block, _ := aes.NewCipher(key)
	a, _ := cipher.NewGCM(block)
	nonce := make([]byte, a.NonceSize())
	encrypted := base64.StdEncoding.EncodeToString(a.Seal(nonce, nonce, []byte("original-agent-signing-secret"), nil))
	docker("exec", name, "sh", "-c", `MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot -e "CREATE DATABASE restored; CREATE TABLE agentscope.credentials (id INT PRIMARY KEY, secret TEXT); INSERT INTO agentscope.credentials VALUES (1,'`+encrypted+`'); CREATE TABLE agentscope.events (id INT PRIMARY KEY, payload TEXT); INSERT INTO agentscope.events VALUES (7,'event-payload');"`)
	path, err := SaveBackup(t.TempDir(), "integration-backup-password", config, func(w io.Writer) error {
		cmd := exec.Command("docker", "exec", name, "sh", "-c", `MYSQL_PWD="$MYSQL_PASSWORD" exec mysqldump --single-transaction --quick --no-tablespaces --set-gtid-purged=OFF --hex-blob -u agentscope agentscope`)
		cmd.Stdout = w
		return cmd.Run()
	})
	if err != nil {
		t.Fatal(err)
	}
	sql, recovered, err := ReadBackup(path, "integration-backup-password")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("docker", "exec", "-i", name, "sh", "-c", `MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysql -uroot restored`)
	cmd.Stdin = bytes.NewReader(sql)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("import: %v %s", err, out)
	}
	out := docker("exec", name, "sh", "-c", `MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -N -B -uroot restored -e 'SELECT secret FROM credentials WHERE id=1; SELECT payload FROM events WHERE id=7'`)
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 || lines[1] != "event-payload" {
		t.Fatalf("restored rows differ: %q", out)
	}
	restoredPath := filepath.Join(t.TempDir(), "agentops.env")
	if err = WriteNewConfig(restoredPath, recovered); err != nil {
		t.Fatal(err)
	}
	restored, err := LoadConfig(restoredPath)
	if err != nil {
		t.Fatal(err)
	}
	restoredKey, _ := base64.StdEncoding.DecodeString(restored.Values["AGENT_SIGNING_ENCRYPTION_KEY"])
	block, _ = aes.NewCipher(restoredKey)
	a, _ = cipher.NewGCM(block)
	blob, _ := base64.StdEncoding.DecodeString(lines[0])
	plain, err := a.Open(nil, blob[:a.NonceSize()], blob[a.NonceSize():], nil)
	if err != nil || string(plain) != "original-agent-signing-secret" {
		t.Fatal("original credential no longer decrypts", err)
	}
}
