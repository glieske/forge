package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver/v3"
	"github.com/glieske/forge/internal/config"
	"github.com/glieske/forge/internal/platform"
	"github.com/glieske/forge/internal/repo"
	"github.com/glieske/forge/internal/security"
)

type Manager struct {
	Paths   platform.Paths
	Config  config.Config
	Repo    repo.Client
	Version string
}

type Installed struct {
	Name     string
	Version  string
	Current  string
	Manifest Manifest
}

func (m Manager) Available(ctx context.Context) (repo.PluginIndex, error) {
	return m.Repo.PluginIndex(ctx)
}

func (m Manager) Install(ctx context.Context, name, version, channel string) error {
	return m.install(ctx, name, version, channel, map[string]bool{})
}

func (m Manager) install(ctx context.Context, name, version, channel string, visiting map[string]bool) error {
	if channel == "" {
		channel = m.Config.Repositories.Channel
	}
	versions, err := m.Repo.PluginVersions(ctx, name)
	if err != nil {
		return err
	}
	if version == "" {
		version = versions.Channels[channel]
	}
	if version == "" {
		return fmt.Errorf("no version for plugin %s on channel %s", name, channel)
	}
	key := name + "@" + version
	if visiting[key] {
		return fmt.Errorf("plugin dependency cycle detected at %s", key)
	}
	visiting[key] = true
	defer delete(visiting, key)

	pkgName := packageName(name, version, runtime.GOOS, runtime.GOARCH)
	base := filepath.ToSlash(filepath.Join(name, version))
	manifestBytes, err := m.Repo.GetBytes(ctx, base+"/manifest.toml")
	if err != nil {
		return err
	}
	manifest, err := DecodeManifest(manifestBytes)
	if err != nil {
		return err
	}
	if manifest.Name != name {
		return fmt.Errorf("manifest name %q does not match requested plugin %q", manifest.Name, name)
	}
	if manifest.Version != version {
		return fmt.Errorf("manifest version %q does not match requested version %q", manifest.Version, version)
	}
	if err := m.installDependencies(ctx, manifest, channel, visiting); err != nil {
		return err
	}
	pkg, err := m.Repo.GetBytes(ctx, base+"/"+pkgName)
	if err != nil {
		return err
	}
	checksums, err := m.Repo.GetBytes(ctx, base+"/checksums.txt")
	if err != nil {
		return err
	}
	if m.Config.Security.RequireSignatures {
		sig, err := m.Repo.GetBytes(ctx, base+"/checksums.txt.sig")
		if err != nil {
			return err
		}
		if err := security.VerifyEd25519(m.Config.Security.PublicKey, checksums, sig); err != nil {
			return err
		}
	}
	if m.Config.Security.RequireChecksums {
		if err := security.VerifyChecksum(pkg, string(checksums), pkgName); err != nil {
			return err
		}
	}
	dest := filepath.Join(m.Paths.PluginDir, name, "versions", version)
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if err := Extract(pkgName, pkg, dest); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dest, "manifest.toml"), manifestBytes, 0o644); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Join(dest, platform.ExeName(manifest.Entrypoint)), 0o755); err != nil && runtime.GOOS != "windows" {
		return err
	}
	return m.activate(name, version)
}

func (m Manager) installDependencies(ctx context.Context, manifest Manifest, channel string, visiting map[string]bool) error {
	for _, dep := range manifest.Dependencies {
		depChannel := dep.Channel
		if depChannel == "" {
			depChannel = channel
		}
		installed, err := m.Info(dep.Name)
		if err == nil && dependencySatisfied(installed.Version, dep.Version) {
			continue
		}
		if err == nil && dep.Optional {
			continue
		}
		if err != nil && dep.Optional {
			continue
		}
		depVersion, err := m.resolveDependencyVersion(ctx, dep.Name, dep.Version, depChannel)
		if err != nil {
			return fmt.Errorf("resolve dependency %s for %s: %w", dep.Name, manifest.Name, err)
		}
		if err := m.install(ctx, dep.Name, depVersion, depChannel, visiting); err != nil {
			return fmt.Errorf("install dependency %s for %s: %w", dep.Name, manifest.Name, err)
		}
	}
	return nil
}

func (m Manager) resolveDependencyVersion(ctx context.Context, name, constraint, channel string) (string, error) {
	versions, err := m.Repo.PluginVersions(ctx, name)
	if err != nil {
		return "", err
	}
	if constraint == "" {
		if version := versions.Channels[channel]; version != "" {
			return version, nil
		}
		return LatestVersion(versions.Versions)
	}
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return "", err
	}
	var matches []*semver.Version
	for _, raw := range versions.Versions {
		v, err := semver.NewVersion(raw)
		if err != nil {
			return "", err
		}
		if c.Check(v) {
			matches = append(matches, v)
		}
	}
	sort.Sort(semver.Collection(matches))
	if len(matches) == 0 {
		return "", fmt.Errorf("no version of %s satisfies %q", name, constraint)
	}
	return matches[len(matches)-1].Original(), nil
}

func dependencySatisfied(version, constraint string) bool {
	if constraint == "" {
		return true
	}
	v, err := semver.NewVersion(version)
	if err != nil {
		return false
	}
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return false
	}
	return c.Check(v)
}

func (m Manager) Update(ctx context.Context, name string) error {
	if name != "" {
		return m.Install(ctx, name, "", m.Config.Repositories.Channel)
	}
	installed, err := m.List()
	if err != nil {
		return err
	}
	for _, item := range installed {
		if err := m.Install(ctx, item.Name, "", m.Config.Repositories.Channel); err != nil {
			return err
		}
	}
	return nil
}

func (m Manager) Remove(name string) error {
	return os.RemoveAll(filepath.Join(m.Paths.PluginDir, name))
}

func (m Manager) Info(name string) (Installed, error) {
	items, err := m.List()
	if err != nil {
		return Installed{}, err
	}
	for _, item := range items {
		if item.Name == name {
			return item, nil
		}
	}
	return Installed{}, fmt.Errorf("plugin %s is not installed", name)
}

func (m Manager) List() ([]Installed, error) {
	entries, err := os.ReadDir(m.Paths.PluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Installed
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		item, err := m.loadInstalled(entry.Name())
		if err == nil {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (m Manager) Run(ctx context.Context, name string, args []string, noInteractive bool) error {
	cmd, err := m.Command(ctx, name, args, noInteractive)
	if err != nil {
		return err
	}
	return cmd.Run()
}

func (m Manager) Command(ctx context.Context, name string, args []string, noInteractive bool) (*exec.Cmd, error) {
	item, err := m.Info(name)
	if err != nil {
		return nil, err
	}
	if err := validateArgs(item.Manifest, args, noInteractive); err != nil {
		return nil, err
	}
	exe := filepath.Join(item.Current, platform.ExeName(item.Manifest.Entrypoint))
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = m.pluginEnv(item)
	return cmd, nil
}

func (m Manager) pluginEnv(item Installed) []string {
	globals, _ := json.Marshal(m.Config.Globals)
	return append(os.Environ(),
		"FORGE_PLUGIN_NAME="+item.Name,
		"FORGE_PLUGIN_VERSION="+item.Version,
		"FORGE_CONFIG_PATH="+m.Paths.ConfigPath,
		"FORGE_DATA_DIR="+m.Paths.DataDir,
		"FORGE_PLUGIN_DIR="+filepath.Join(m.Paths.PluginDir, item.Name),
		"FORGE_SECRETS_MODE="+m.Config.Security.SecretsBackend,
		"FORGE_GLOBAL_CONFIG_JSON="+string(globals),
	)
}

func (m Manager) activate(name, version string) error {
	root := filepath.Join(m.Paths.PluginDir, name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	current := filepath.Join(root, "current")
	if runtime.GOOS == "windows" {
		return os.WriteFile(filepath.Join(root, "current.toml"), []byte("version = \""+version+"\"\n"), 0o644)
	}
	_ = os.Remove(current)
	if err := os.Symlink(filepath.Join("versions", version), current); err != nil {
		return err
	}
	return nil
}

func (m Manager) loadInstalled(name string) (Installed, error) {
	root := filepath.Join(m.Paths.PluginDir, name)
	current := filepath.Join(root, "current")
	if runtime.GOOS == "windows" {
		data, err := os.ReadFile(filepath.Join(root, "current.toml"))
		if err != nil {
			return Installed{}, err
		}
		var cur struct {
			Version string `toml:"version"`
		}
		if err := tomlDecode(data, &cur); err != nil {
			return Installed{}, err
		}
		current = filepath.Join(root, "versions", cur.Version)
	}
	realCurrent, err := filepath.EvalSymlinks(current)
	if err != nil && runtime.GOOS != "windows" {
		return Installed{}, err
	}
	if runtime.GOOS == "windows" {
		realCurrent = current
	}
	manifest, err := LoadManifest(filepath.Join(realCurrent, "manifest.toml"))
	if err != nil {
		return Installed{}, err
	}
	return Installed{Name: manifest.Name, Version: manifest.Version, Current: realCurrent, Manifest: manifest}, nil
}

func packageName(name, version, goos, goarch string) string {
	return fmt.Sprintf("forge-%s_%s_%s%s", name, goos, goarch, platform.PackageExt(goos))
}

func validateArgs(manifest Manifest, args []string, noInteractive bool) error {
	if len(manifest.Commands) == 0 {
		return nil
	}
	cmd := manifest.Commands[0]
	required := 0
	for _, arg := range cmd.Args {
		if arg.Required && arg.Position > required {
			required = arg.Position
		}
	}
	if len(args) < required && noInteractive {
		return fmt.Errorf("plugin %s requires %d positional arguments, got %d", manifest.Name, required, len(args))
	}
	return nil
}

func tomlDecode(data []byte, v interface{}) error {
	_, err := toml.Decode(string(data), v)
	return err
}

func LatestVersion(versions []string) (string, error) {
	var parsed []*semver.Version
	for _, raw := range versions {
		v, err := semver.NewVersion(raw)
		if err != nil {
			return "", err
		}
		parsed = append(parsed, v)
	}
	sort.Sort(semver.Collection(parsed))
	if len(parsed) == 0 {
		return "", fmt.Errorf("no versions")
	}
	return parsed[len(parsed)-1].Original(), nil
}
