package secrets

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/glieske/forge/internal/platform"
	"github.com/zalando/go-keyring"
)

type Store interface {
	Get(scope, key string) (string, error)
	Set(scope, key, value string) error
	Delete(scope, key string) error
	Backend() string
}

func New(paths platform.Paths, mode string) Store {
	if mode == "" || mode == "auto" || mode == "keyring" {
		ks := KeyringStore{}
		if err := ks.Set("__probe__", "__probe__", "1"); err == nil {
			_ = ks.Delete("__probe__", "__probe__")
			return ks
		}
		if mode == "keyring" {
			return ks
		}
	}
	return FileStore{Path: paths.SecretsPath}
}

type KeyringStore struct{}

func (KeyringStore) Get(scope, key string) (string, error) {
	return keyring.Get(service(scope), key)
}

func (KeyringStore) Set(scope, key, value string) error {
	return keyring.Set(service(scope), key, value)
}

func (KeyringStore) Delete(scope, key string) error {
	return keyring.Delete(service(scope), key)
}

func (KeyringStore) Backend() string { return "keyring" }

type FileStore struct {
	Path string
}

func (s FileStore) Get(scope, key string) (string, error) {
	data, err := s.read()
	if err != nil {
		return "", err
	}
	if data[scope] == nil {
		return "", fmt.Errorf("secret %s/%s not found", scope, key)
	}
	value, ok := data[scope][key]
	if !ok {
		return "", fmt.Errorf("secret %s/%s not found", scope, key)
	}
	return value, nil
}

func (s FileStore) Set(scope, key, value string) error {
	data, err := s.read()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if data == nil {
		data = map[string]map[string]string{}
	}
	if data[scope] == nil {
		data[scope] = map[string]string{}
	}
	data[scope][key] = value
	return s.write(data)
}

func (s FileStore) Delete(scope, key string) error {
	data, err := s.read()
	if err != nil {
		return err
	}
	if data[scope] != nil {
		delete(data[scope], key)
	}
	return s.write(data)
}

func (s FileStore) Backend() string { return "file" }

func (s FileStore) read() (map[string]map[string]string, error) {
	data := map[string]map[string]string{}
	if _, err := os.Stat(s.Path); err != nil {
		return data, err
	}
	if _, err := toml.DecodeFile(s.Path, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (s FileStore) write(data map[string]map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.Path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := toml.NewEncoder(f).Encode(data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func service(scope string) string {
	return "forge:" + scope
}
