package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/glieske/forge/internal/config"
	"github.com/glieske/forge/internal/platform"
	"github.com/glieske/forge/internal/plugin"
	"github.com/glieske/forge/internal/repo"
	"github.com/glieske/forge/internal/secrets"
	"github.com/glieske/forge/internal/selfupdate"
	"github.com/glieske/forge/internal/tui"
	"github.com/spf13/cobra"
)

type runtimeState struct {
	version       string
	commit        string
	paths         platform.Paths
	cfg           config.Config
	noInteractive bool
}

func Execute(version, commit string) error {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-v" || os.Args[1] == "version") {
		fmt.Println(version + " (" + commit + ")")
		return nil
	}
	root, err := NewRootCommand(version, commit)
	if err != nil {
		return err
	}
	return root.Execute()
}

func NewRootCommand(version, commit string) (*cobra.Command, error) {
	paths, err := platform.ResolvePaths()
	if err != nil {
		return nil, err
	}
	if err := config.EnsureFile(paths); err != nil {
		return nil, err
	}
	cfg, err := config.Load(paths)
	if err != nil {
		return nil, err
	}
	st := &runtimeState{version: version, commit: commit, paths: paths, cfg: cfg}
	root := &cobra.Command{
		Use:           "forge",
		Short:         "Developer support CLI with installable plugins",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version + " (" + commit + ")",
		RunE: func(cmd *cobra.Command, args []string) error {
			return st.runTUI()
		},
	}
	root.PersistentFlags().BoolVar(&st.noInteractive, "no-interactive", false, "disable interactive prompts and TUI fallback")
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error { return err })
	root.AddCommand(st.pluginCommand())
	root.AddCommand(st.configCommand())
	root.AddCommand(st.secretCommand())
	root.AddCommand(st.selfUpdateCommand())
	root.AddCommand(&cobra.Command{Use: "tui", Short: "Open interactive TUI", RunE: func(cmd *cobra.Command, args []string) error { return st.runTUI() }})
	if installed, err := st.pluginManager().List(); err == nil {
		for _, item := range installed {
			name := item.Name
			root.AddCommand(&cobra.Command{
				Use:                name + " [args...]",
				Short:              "Run installed plugin " + name,
				Args:               cobra.ArbitraryArgs,
				DisableFlagParsing: true,
				RunE: func(cmd *cobra.Command, args []string) error {
					return st.pluginManager().Run(cmd.Context(), name, args, st.noInteractive)
				},
			})
		}
	}
	root.SetHelpCommand(&cobra.Command{Use: "help [command]", Run: func(cmd *cobra.Command, args []string) { _ = cmd.Root().Help() }})
	root.Args = cobra.ArbitraryArgs
	root.SetArgs(os.Args[1:])
	root.SetVersionTemplate("{{.Version}}\n")
	root.ValidArgsFunction = cobra.NoFileCompletions
	root.TraverseChildren = true
	root.FParseErrWhitelist.UnknownFlags = true
	root.PreRunE = nil
	root.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return st.runTUI()
		}
		return st.pluginManager().Run(cmd.Context(), args[0], args[1:], st.noInteractive)
	}
	return root, nil
}

func (s *runtimeState) pluginManager() plugin.Manager {
	return plugin.Manager{
		Paths:   s.paths,
		Config:  s.cfg,
		Repo:    repo.New(s.cfg.Repositories.PluginsURL),
		Version: s.version,
	}
}

func (s *runtimeState) runTUI() error {
	return tui.Run(tui.Options{
		Version: s.version,
		Paths:   s.paths,
		Config:  s.cfg,
		Plugins: s.pluginManager(),
		Secrets: secrets.New(s.paths, s.cfg.Security.SecretsBackend),
	})
}

func (s *runtimeState) pluginCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "plugin", Short: "Manage plugins"}
	cmd.AddCommand(&cobra.Command{Use: "list", Short: "List installed plugins", RunE: func(cmd *cobra.Command, args []string) error {
		items, err := s.pluginManager().List()
		if err != nil {
			return err
		}
		for _, item := range items {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", item.Name, item.Version)
		}
		return nil
	}})
	cmd.AddCommand(&cobra.Command{Use: "available", Short: "List available plugins", RunE: func(cmd *cobra.Command, args []string) error {
		idx, err := s.pluginManager().Available(cmd.Context())
		if err != nil {
			return err
		}
		for _, p := range idx.Plugins {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", p.Name, p.Latest, p.Description)
		}
		return nil
	}})
	var version, channel string
	install := &cobra.Command{Use: "install <name>", Short: "Install a plugin", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return s.pluginManager().Install(cmd.Context(), args[0], version, channel)
	}}
	install.Flags().StringVar(&version, "version", "", "plugin version")
	install.Flags().StringVar(&channel, "channel", "", "release channel")
	cmd.AddCommand(install)
	cmd.AddCommand(&cobra.Command{Use: "update [name]", Short: "Update plugins", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) == 1 {
			name = args[0]
		}
		return s.pluginManager().Update(cmd.Context(), name)
	}})
	cmd.AddCommand(&cobra.Command{Use: "remove <name>", Short: "Remove a plugin", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return s.pluginManager().Remove(args[0])
	}})
	cmd.AddCommand(&cobra.Command{Use: "info <name>", Short: "Show plugin info", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		item, err := s.pluginManager().Info(args[0])
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n%s\n", item.Name, item.Version, item.Manifest.Description)
		for _, c := range item.Manifest.Commands {
			fmt.Fprintf(cmd.OutOrStdout(), "command: %s\t%s\n", c.Name, c.Description)
		}
		return nil
	}})
	return cmd
}

func (s *runtimeState) configCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Manage configuration"}
	cmd.AddCommand(&cobra.Command{Use: "get <key>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		value, err := config.Get(s.cfg, args[0])
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), value)
		return nil
	}})
	cmd.AddCommand(&cobra.Command{Use: "set <key> <value>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Set(&s.cfg, args[0], args[1]); err != nil {
			return err
		}
		return config.Save(s.paths.ConfigPath, s.cfg)
	}})
	cmd.AddCommand(&cobra.Command{Use: "edit", Short: "Open config in $EDITOR", RunE: func(cmd *cobra.Command, args []string) error {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			return fmt.Errorf("EDITOR is not set; config path: %s", s.paths.ConfigPath)
		}
		parts := strings.Fields(editor)
		c := exec.Command(parts[0], append(parts[1:], s.paths.ConfigPath)...)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}})
	return cmd
}

func (s *runtimeState) secretCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "secret", Short: "Manage secrets"}
	store := func() secrets.Store { return secrets.New(s.paths, s.cfg.Security.SecretsBackend) }
	cmd.AddCommand(&cobra.Command{Use: "get <scope> <key>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		value, err := store().Get(args[0], args[1])
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), value)
		return nil
	}})
	cmd.AddCommand(&cobra.Command{Use: "set <scope> <key>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprint(cmd.OutOrStdout(), "value: ")
		var value string
		if _, err := fmt.Fscan(os.Stdin, &value); err != nil {
			return err
		}
		return store().Set(args[0], args[1], value)
	}})
	cmd.AddCommand(&cobra.Command{Use: "delete <scope> <key>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		return store().Delete(args[0], args[1])
	}})
	return cmd
}

func (s *runtimeState) selfUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "self-update", Short: "Manage forge updates"}
	mgr := func() selfupdate.Manager {
		return selfupdate.Manager{Config: s.cfg, Repo: repo.New(s.cfg.Repositories.UpdatesURL), Version: s.version}
	}
	cmd.AddCommand(&cobra.Command{Use: "check", RunE: func(cmd *cobra.Command, args []string) error {
		res, err := mgr().Check(cmd.Context())
		if err != nil {
			return err
		}
		if res.Update {
			fmt.Fprintf(cmd.OutOrStdout(), "update available: %s -> %s (%s)\n", res.Current, res.Latest, res.Channel)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "up to date: %s (%s)\n", res.Current, res.Channel)
		}
		return nil
	}})
	cmd.AddCommand(&cobra.Command{Use: "apply", RunE: func(cmd *cobra.Command, args []string) error {
		return mgr().Apply(cmd.Context(), "")
	}})
	cmd.AddCommand(&cobra.Command{Use: "channel <channel>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if args[0] != "stable" && args[0] != "beta" && args[0] != "dev" {
			return fmt.Errorf("unsupported channel %q", args[0])
		}
		s.cfg.Repositories.Channel = args[0]
		return config.Save(s.paths.ConfigPath, s.cfg)
	}})
	return cmd
}

func Context() context.Context {
	return context.Background()
}
