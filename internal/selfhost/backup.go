package selfhost

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/scrypt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxBundle = 256 << 20
const magic = "AgentOpsBackup1\n"

type Manifest struct {
	Version      int    `json:"version"`
	SQLSHA256    string `json:"sql_sha256"`
	ConfigSHA256 string `json:"config_sha256"`
	Redis        string `json:"redis_policy"`
}

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func bundle(sql, config []byte) ([]byte, error) {
	if len(sql) == 0 || len(sql)+len(config) > maxBundle {
		return nil, fmt.Errorf("empty or oversized backup (limit 256 MiB)")
	}
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	m, _ := json.Marshal(Manifest{1, digest(sql), digest(config), "rebuild ephemeral Redis state; sessions/nonces/rate-limit counters are reset"})
	for _, entry := range []struct {
		name string
		data []byte
	}{{"database.sql", sql}, {"agentops.env", config}, {"manifest.json", m}} {
		w, err := z.Create(entry.name)
		if err != nil {
			return nil, err
		}
		if _, err = w.Write(entry.data); err != nil {
			return nil, err
		}
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
func backupCipher(password string, salt []byte) (cipher.AEAD, error) {
	if len(password) < 16 {
		return nil, fmt.Errorf("AGENTOPS_BACKUP_PASSWORD must have at least 16 characters")
	}
	key, err := scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// SaveBackup publishes only a complete, authenticated bundle. Random names avoid collisions.
func SaveBackup(directory, password string, config []byte, dump func(io.Writer) error) (string, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	if err := Restrict(directory); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(directory, ".pending-*")
	if err != nil {
		return "", err
	}
	name := f.Name()
	defer os.Remove(name)
	defer f.Close()
	if err = Restrict(name); err != nil {
		return "", err
	}
	// Dump into a bounded buffer: this version deliberately rejects backups over 256 MiB.
	var sql bytes.Buffer
	if err = dump(&limitedWriter{&sql, maxBundle}); err != nil {
		return "", err
	}
	payload, err := bundle(sql.Bytes(), config)
	if err != nil {
		return "", err
	}
	salt := make([]byte, 16)
	if _, err = rand.Read(salt); err != nil {
		return "", err
	}
	a, err := backupCipher(password, salt)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, a.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}
	header := append(append([]byte(magic), salt...), nonce...)
	data := append(header, a.Seal(nil, nonce, payload, header)...)
	if _, err = f.Write(data); err != nil {
		return "", err
	}
	if err = f.Sync(); err != nil {
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	if _, _, err = ReadBackup(name, password); err != nil {
		return "", err
	}
	final := filepath.Join(directory, "agentops-"+strings.TrimPrefix(filepath.Base(name), ".pending-")+".aob")
	if err = os.Rename(name, final); err != nil {
		return "", err
	}
	return final, nil
}

type limitedWriter struct {
	w         io.Writer
	remaining int
}

func (w *limitedWriter) Write(b []byte) (int, error) {
	if len(b) > w.remaining {
		return 0, fmt.Errorf("backup exceeds 256 MiB limit")
	}
	n, e := w.w.Write(b)
	w.remaining -= n
	return n, e
}
func ReadBackup(path, password string) (sql, config []byte, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxBundle+(1<<20)+1))
	if err != nil {
		return nil, nil, err
	}
	h := len(magic) + 16 + 12
	if len(data) < h+16 || len(data) > maxBundle+(1<<20) || string(data[:len(magic)]) != magic {
		return nil, nil, fmt.Errorf("invalid backup format or size")
	}
	a, err := backupCipher(password, data[len(magic):len(magic)+16])
	if err != nil {
		return nil, nil, err
	}
	payload, err := a.Open(nil, data[h-12:h], data[h:], data[:h])
	if err != nil {
		return nil, nil, fmt.Errorf("backup authentication failed")
	}
	z, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return nil, nil, err
	}
	entries := map[string][]byte{}
	total := 0
	for _, file := range z.File {
		if file.Name != "database.sql" && file.Name != "agentops.env" && file.Name != "manifest.json" {
			return nil, nil, fmt.Errorf("unexpected backup entry")
		}
		if _, ok := entries[file.Name]; ok {
			return nil, nil, fmt.Errorf("duplicate backup entry")
		}
		r, e := file.Open()
		if e != nil {
			return nil, nil, e
		}
		b, e := io.ReadAll(io.LimitReader(r, int64(maxBundle-total+1)))
		r.Close()
		if e != nil {
			return nil, nil, e
		}
		total += len(b)
		if total > maxBundle {
			return nil, nil, fmt.Errorf("oversized backup")
		}
		entries[file.Name] = b
	}
	var m Manifest
	if err = json.Unmarshal(entries["manifest.json"], &m); err != nil {
		return nil, nil, err
	}
	sql, config = entries["database.sql"], entries["agentops.env"]
	if m.Version != 1 || len(sql) == 0 || digest(sql) != m.SQLSHA256 || digest(config) != m.ConfigSHA256 {
		return nil, nil, fmt.Errorf("backup checksum mismatch")
	}
	c, err := ParseConfig(bytes.NewReader(config), false)
	if err != nil {
		return nil, nil, err
	}
	if err = c.Validate(); err != nil {
		return nil, nil, err
	}
	return sql, config, nil
}
