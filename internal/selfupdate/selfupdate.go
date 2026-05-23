package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Masterminds/semver/v3"
	"github.com/glieske/forge/internal/config"
	"github.com/glieske/forge/internal/platform"
	"github.com/glieske/forge/internal/plugin"
	"github.com/glieske/forge/internal/repo"
	"github.com/glieske/forge/internal/security"
)

type Manager struct {
	Config  config.Config
	Paths   platform.Paths
	Repo    repo.Client
	Version string
}

type CheckResult struct {
	Current string
	Latest  string
	Update  bool
	Channel string
}

func (m Manager) Check(ctx context.Context) (CheckResult, error) {
	channel := m.Config.Repositories.Channel
	idx, err := m.updateIndex(ctx, channel)
	if err != nil {
		return CheckResult{}, err
	}
	current, err := semver.NewVersion(m.Version)
	if err != nil {
		current, _ = semver.NewVersion("0.0.0")
	}
	latest, err := semver.NewVersion(idx.Latest)
	if err != nil {
		return CheckResult{}, err
	}
	return CheckResult{Current: m.Version, Latest: idx.Latest, Update: latest.GreaterThan(current), Channel: channel}, nil
}

func (m Manager) Apply(ctx context.Context, targetPath string) error {
	res, err := m.Check(ctx)
	if err != nil {
		return err
	}
	if !res.Update {
		return nil
	}
	pkgName := fmt.Sprintf("forge_%s_%s%s", runtime.GOOS, runtime.GOARCH, platform.PackageExt(runtime.GOOS))
	rel := filepath.ToSlash(filepath.Join(res.Channel, res.Latest, pkgName))
	pkg, err := m.Repo.GetBytes(ctx, rel)
	if err != nil {
		return err
	}
	base := filepath.ToSlash(filepath.Join(res.Channel, res.Latest))
	checksums, err := m.Repo.GetBytes(ctx, base+"/checksums.txt")
	if err != nil {
		return err
	}
	if m.Config.Security.RequireSignatures {
		publicKey, err := m.signingPublicKey(ctx)
		if err != nil {
			return err
		}
		sig, err := m.Repo.GetBytes(ctx, base+"/checksums.txt.sig")
		if err != nil {
			return err
		}
		if err := security.VerifyEd25519(publicKey, checksums, sig); err != nil {
			return err
		}
	}
	if m.Config.Security.RequireChecksums {
		if err := security.VerifyChecksum(pkg, string(checksums), pkgName); err != nil {
			return err
		}
	}
	if targetPath == "" {
		targetPath, err = os.Executable()
		if err != nil {
			return err
		}
	}
	tmpDir, err := os.MkdirTemp("", "forge-update-*")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()
	if err := plugin.Extract(pkgName, pkg, tmpDir); err != nil {
		return err
	}
	candidate := filepath.Join(tmpDir, platform.ExeName("forge"))
	newPath := targetPath + ".new"
	data, err := os.ReadFile(candidate)
	if err != nil {
		return err
	}
	if err := os.WriteFile(newPath, data, 0o755); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		return fmt.Errorf("downloaded update to %s; replace the running executable after exit", newPath)
	}
	return os.Rename(newPath, targetPath)
}

func (m Manager) updateIndex(ctx context.Context, channel string) (repo.UpdateIndex, error) {
	rel := filepath.ToSlash(filepath.Join(channel, "index.json"))
	body, err := m.Repo.GetBytes(ctx, rel)
	if err != nil {
		return repo.UpdateIndex{}, err
	}
	if m.Config.Security.RequireSignatures {
		publicKey, err := m.signingPublicKey(ctx)
		if err != nil {
			return repo.UpdateIndex{}, err
		}
		sig, err := m.Repo.GetBytes(ctx, rel+".sig")
		if err != nil {
			return repo.UpdateIndex{}, err
		}
		if err := security.VerifyEd25519(publicKey, body, sig); err != nil {
			return repo.UpdateIndex{}, fmt.Errorf("verify %s: %w", rel, err)
		}
	}
	var idx repo.UpdateIndex
	return idx, json.Unmarshal(body, &idx)
}

func (m Manager) signingPublicKey(ctx context.Context) (string, error) {
	return security.RepositoryPublicKey(ctx, m.Paths, m.Config, m.Repo, "updates")
}
