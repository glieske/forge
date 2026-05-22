package platform

import (
	"os"
	"path/filepath"
	"runtime"
)

type Paths struct {
	ConfigDir   string
	ConfigPath  string
	DataDir     string
	PluginDir   string
	CacheDir    string
	SecretsPath string
}

func ResolvePaths() (Paths, error) {
	if root := os.Getenv("FORGE_HOME"); root != "" {
		configDir := filepath.Join(root, "config")
		dataDir := filepath.Join(root, "data")
		cacheDir := filepath.Join(root, "cache")
		return Paths{
			ConfigDir:   configDir,
			ConfigPath:  filepath.Join(configDir, "config.toml"),
			DataDir:     dataDir,
			PluginDir:   filepath.Join(dataDir, "plugins"),
			CacheDir:    cacheDir,
			SecretsPath: filepath.Join(configDir, "secrets.toml"),
		}, nil
	}
	configRoot, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, err
	}
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return Paths{}, err
	}
	dataRoot := dataRoot(configRoot)
	configDir := filepath.Join(configRoot, "forge")
	dataDir := filepath.Join(dataRoot, "forge")
	cacheDir := filepath.Join(cacheRoot, "forge")
	return Paths{
		ConfigDir:   configDir,
		ConfigPath:  filepath.Join(configDir, "config.toml"),
		DataDir:     dataDir,
		PluginDir:   filepath.Join(dataDir, "plugins"),
		CacheDir:    cacheDir,
		SecretsPath: filepath.Join(configDir, "secrets.toml"),
	}, nil
}

func dataRoot(configRoot string) string {
	if override := os.Getenv("XDG_DATA_HOME"); override != "" && runtime.GOOS == "linux" {
		return override
	}
	if runtime.GOOS == "linux" {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, ".local", "share")
		}
	}
	return configRoot
}

func Ensure(paths Paths) error {
	for _, dir := range []string{paths.ConfigDir, paths.DataDir, paths.PluginDir, paths.CacheDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func ExeName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func PackageExt(goos string) string {
	if goos == "windows" {
		return ".zip"
	}
	return ".tar.gz"
}
