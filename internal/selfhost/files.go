package selfhost

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ConfigBytes(c Config) []byte {
	keys := make([]string, 0, len(c.Values))
	for key := range c.Values {
		if allowed[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&b, "%s=%s\n", key, c.Values[key])
	}
	return []byte(b.String())
}
func LoadConfig(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()
	c, err := ParseConfig(f, false)
	if err == nil {
		err = c.Validate()
	}
	return c, err
}
func Configure(path, apiImage, webImage string) error {
	if _, err := os.Stat(path); err == nil {
		_, err = LoadConfig(path)
		return err
	} else if !os.IsNotExist(err) {
		return err
	}
	c, err := ParseConfig(strings.NewReader(""), true)
	if err != nil {
		return err
	}
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return err
	}
	c.Values["MYSQL_PASSWORD"] = base64.RawURLEncoding.EncodeToString(b)
	c.Values["AGENTOPS_API_IMAGE"] = apiImage
	c.Values["AGENTOPS_WEB_IMAGE"] = webImage
	if err = c.Validate(); err != nil {
		return err
	}
	return WriteNewConfig(path, ConfigBytes(c))
}
func WriteNewConfig(path string, data []byte) error {
	c, err := ParseConfig(strings.NewReader(string(data)), false)
	if err != nil {
		return err
	}
	if err = c.Validate(); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if err = Restrict(dir); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".config-pending-*")
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		f.Close()
		if !ok {
			os.Remove(f.Name())
		}
	}()
	if err = Restrict(f.Name()); err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Link(f.Name(), path); err != nil {
		return err
	}
	os.Remove(f.Name())
	ok = true
	return nil
}
