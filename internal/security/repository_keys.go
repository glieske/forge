package security

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/glieske/forge/internal/config"
	"github.com/glieske/forge/internal/platform"
	"github.com/glieske/forge/internal/repo"
)

type pinnedRepositories struct {
	Repositories map[string]pinnedRepository `toml:"repositories"`
}

type pinnedRepository struct {
	URL         string `toml:"url"`
	Fingerprint string `toml:"fingerprint"`
}

func RepositoryPublicKey(ctx context.Context, paths platform.Paths, cfg config.Config, client repo.Client, scope string) (string, error) {
	if cfg.Security.PublicKey != "" {
		return cfg.Security.PublicKey, nil
	}
	key, err := client.GetBytes(ctx, "public-key.ed25519")
	if err != nil {
		return "", fmt.Errorf("load signing public key from %s repository: %w", scope, err)
	}
	publicKey := strings.TrimSpace(string(key))
	fingerprint, err := PublicKeyFingerprint(publicKey)
	if err != nil {
		return "", fmt.Errorf("invalid %s repository public key: %w", scope, err)
	}
	if err := verifyPinnedRepositoryKey(paths, scope, client.BaseURL, fingerprint); err != nil {
		return "", err
	}
	return publicKey, nil
}

func verifyPinnedRepositoryKey(paths platform.Paths, scope, repoURL, fingerprint string) error {
	configDir := paths.ConfigDir
	if configDir == "" && paths.ConfigPath != "" {
		configDir = filepath.Dir(paths.ConfigPath)
	}
	if configDir == "" {
		configDir = paths.DataDir
	}
	path := filepath.Join(configDir, "trusted-repositories.toml")
	pinned, err := loadPinnedRepositories(path)
	if err != nil {
		return err
	}
	current, ok := pinned.Repositories[scope]
	if ok && current.Fingerprint != "" {
		if current.Fingerprint != fingerprint {
			return fmt.Errorf("%s repository signing key changed: got %s, trusted %s; set security.public_key or remove %s after verifying the repository", scope, fingerprint, current.Fingerprint, path)
		}
		return nil
	}
	if pinned.Repositories == nil {
		pinned.Repositories = map[string]pinnedRepository{}
	}
	pinned.Repositories[scope] = pinnedRepository{URL: repoURL, Fingerprint: fingerprint}
	return savePinnedRepositories(path, pinned)
}

func loadPinnedRepositories(path string) (pinnedRepositories, error) {
	var pinned pinnedRepositories
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return pinnedRepositories{Repositories: map[string]pinnedRepository{}}, nil
		}
		return pinnedRepositories{}, err
	}
	if _, err := toml.DecodeFile(path, &pinned); err != nil {
		return pinnedRepositories{}, err
	}
	if pinned.Repositories == nil {
		pinned.Repositories = map[string]pinnedRepository{}
	}
	return pinned, nil
}

func savePinnedRepositories(path string, pinned pinnedRepositories) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := toml.NewEncoder(f).Encode(pinned); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
