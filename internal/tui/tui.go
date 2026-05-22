package tui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/glieske/forge/internal/config"
	"github.com/glieske/forge/internal/platform"
	"github.com/glieske/forge/internal/plugin"
	"github.com/glieske/forge/internal/repo"
	"github.com/glieske/forge/internal/secrets"
)

type Options struct {
	Version string
	Paths   platform.Paths
	Config  config.Config
	Plugins plugin.Manager
	Secrets secrets.Store
}

type screen int

const (
	screenDashboard screen = iota
	screenInstalled
	screenAvailable
	screenCommands
	screenConfig
	screenSecrets
	screenInput
	screenChoice
	screenMessage
)

type item struct {
	title  string
	desc   string
	action string
	value  interface{}
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title + " " + i.desc }

type Model struct {
	opts             Options
	screen           screen
	previous         screen
	list             list.Model
	input            textinput.Model
	inputTitle       string
	inputKey         string
	inputScope       string
	inputPlugin      string
	inputPluginField plugin.ConfigSpec
	inputStep        int
	secretKey        string
	status           string
	err              string
	installed        []plugin.Installed
	available        []repo.PluginSummary
	wizard           *commandWizard
	width            int
	height           int
}

type commandWizard struct {
	plugin  plugin.Installed
	command plugin.CommandSpec
	args    []string
	nextArg int
}

type pluginConfigSelection struct {
	plugin plugin.Installed
	field  plugin.ConfigSpec
}

type installedMsg struct {
	items []plugin.Installed
	err   error
}

type availableMsg struct {
	items []repo.PluginSummary
	err   error
}

type actionMsg struct {
	status string
	err    error
}

type runDoneMsg struct {
	err error
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true)
	statusStyle = lipgloss.NewStyle().Faint(true)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

func New(opts Options) Model {
	l := list.New(nil, list.NewDefaultDelegate(), 90, 24)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	in := textinput.New()
	in.CharLimit = 4096
	m := Model{opts: opts, screen: screenDashboard, list: l, input: in}
	m.showDashboard()
	return m
}

func (m Model) Init() tea.Cmd {
	return m.loadInstalled()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(max(40, msg.Width), max(10, msg.Height-5))
	case tea.KeyMsg:
		if m.screen == screenInput {
			return m.updateInput(msg)
		}
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.screen == screenDashboard {
				return m, tea.Quit
			}
			m.showDashboard()
			return m, nil
		case "esc", "backspace":
			m.showDashboard()
			return m, nil
		case "r":
			if strings.TrimSpace(m.opts.Config.Repositories.PluginsURL) == "" {
				return m, m.loadInstalled()
			}
			return m, tea.Batch(m.loadInstalled(), m.loadAvailable())
		case "x":
			if m.screen == screenInstalled {
				selected, ok := m.list.SelectedItem().(item)
				if ok && selected.action == "plugin" {
					p := selected.value.(plugin.Installed)
					return m, m.removePlugin(p.Name)
				}
			}
		case "enter":
			return m.activateSelected()
		}
	case installedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.installed = msg.items
			if m.screen == screenDashboard || m.screen == screenInstalled || m.screen == screenCommands {
				m.refreshScreen()
			}
		}
	case availableMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.available = msg.items
			if m.screen == screenAvailable {
				m.showAvailable()
			}
		}
	case actionMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
			m.status = msg.status
		}
		if m.screen == screenInput {
			m.screen = m.previous
			m.input.SetValue("")
			m.refreshScreen()
		}
		if strings.TrimSpace(m.opts.Config.Repositories.PluginsURL) == "" {
			return m, m.loadInstalled()
		}
		return m, tea.Batch(m.loadInstalled(), m.loadAvailable())
	case runDoneMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.status = "command finished"
		}
		m.showDashboard()
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	header := titleStyle.Render(fmt.Sprintf("forge %s", m.opts.Version))
	body := m.list.View()
	if m.screen == screenInput {
		body = fmt.Sprintf("%s\n\n%s", m.inputTitle, m.input.View())
	}
	footer := "enter: select  /: fuzzy  r: refresh  q: back/quit"
	lines := []string{header, body, statusStyle.Render(footer)}
	if m.status != "" {
		lines = append(lines, statusStyle.Render(m.status))
	}
	if m.err != "" {
		lines = append(lines, errorStyle.Render(m.err))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) showDashboard() {
	m.screen = screenDashboard
	m.list.Title = "Dashboard"
	items := make([]list.Item, 0, 8)
	for _, missing := range config.MissingRepositorySettings(m.opts.Config) {
		items = append(items, item{"Configuration warning", "Missing " + missing + " - open Config > Global settings to set it", "config", nil})
	}
	items = append(items,
		item{"Installed plugins", fmt.Sprintf("%d installed", len(m.installed)), "installed", nil},
		item{"Available plugins", fmt.Sprintf("%d loaded from repository", len(m.available)), "available", nil},
		item{"Commands", "Fuzzy select an installed plugin command", "commands", nil},
		item{"Config", "Edit global settings", "config", nil},
		item{"Secrets", "Set or delete secrets", "secrets", nil},
	)
	m.list.SetItems(items)
}

func (m *Model) showInstalled() {
	m.screen = screenInstalled
	m.list.Title = "Installed Plugins"
	items := make([]list.Item, 0, len(m.installed))
	for _, p := range m.installed {
		items = append(items, item{p.Name, p.Version + " - enter: update, x: remove via actions", "plugin", p})
	}
	if len(items) == 0 {
		items = append(items, item{"No plugins installed", "Install one from Available plugins", "noop", nil})
	}
	m.list.SetItems(items)
}

func (m *Model) showAvailable() {
	m.screen = screenAvailable
	m.list.Title = "Available Plugins"
	if strings.TrimSpace(m.opts.Config.Repositories.PluginsURL) == "" {
		m.list.SetItems([]list.Item{
			item{"Missing plugins repository", "Set repositories.plugins_url in Config > Global settings", "config-global", nil},
		})
		return
	}
	items := make([]list.Item, 0, len(m.available))
	for _, p := range m.available {
		items = append(items, item{p.Name, p.Latest + " - " + p.Description, "install", p})
	}
	if len(items) == 0 {
		items = append(items, item{"No repository data", "Press r to retry or configure repositories.plugins_url", "noop", nil})
	}
	m.list.SetItems(items)
}

func (m *Model) showCommands() {
	m.screen = screenCommands
	m.list.Title = "Commands"
	var items []list.Item
	for _, p := range m.installed {
		for _, c := range p.Manifest.Commands {
			items = append(items, item{p.Name + " " + c.Name, c.Description, "command", commandWizard{plugin: p, command: c}})
		}
	}
	if len(items) == 0 {
		items = append(items, item{"No commands", "Install a plugin first", "noop", nil})
	}
	m.list.SetItems(items)
}

func (m *Model) showConfig() {
	m.screen = screenConfig
	m.list.Title = "Config"
	m.list.SetItems([]list.Item{
		item{"Global settings", "Repository URLs, channel, UI, security, and shared globals", "config-global", nil},
		item{"Plugin settings", "Settings declared by installed plugin manifests", "config-plugins", nil},
	})
}

func (m *Model) showGlobalConfig() {
	m.screen = screenConfig
	m.list.Title = "Global Settings"
	items := []list.Item{
		item{"config file", m.opts.Paths.ConfigPath, "noop", nil},
		configItem("repositories.plugins_url", m.opts.Config.Repositories.PluginsURL),
		configItem("repositories.updates_url", m.opts.Config.Repositories.UpdatesURL),
		configItem("repositories.channel", m.opts.Config.Repositories.Channel),
		configItem("ui.interactive", fmt.Sprint(m.opts.Config.UI.Interactive)),
		configItem("ui.fuzzy_limit", fmt.Sprint(m.opts.Config.UI.FuzzyLimit)),
		configItem("security.public_key", mask(m.opts.Config.Security.PublicKey)),
		configItem("security.secrets_backend", m.opts.Config.Security.SecretsBackend),
	}
	keys := make([]string, 0, len(m.opts.Config.Globals))
	for key := range m.opts.Config.Globals {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		items = append(items, configItem("globals."+key, strings.Join(m.opts.Config.Globals[key], ",")))
	}
	m.list.SetItems(items)
}

func (m *Model) showPluginConfigList() {
	m.screen = screenConfig
	m.list.Title = "Plugin Settings"
	items := make([]list.Item, 0, len(m.installed))
	for _, p := range m.installed {
		desc := fmt.Sprintf("%d settings declared", len(p.Manifest.Config))
		items = append(items, item{p.Name, desc, "config-plugin", p})
	}
	if len(items) == 0 {
		items = append(items, item{"No installed plugins", "Install a plugin with configurable settings first", "noop", nil})
	}
	m.list.SetItems(items)
}

func (m *Model) showPluginConfig(pluginItem plugin.Installed) {
	m.screen = screenConfig
	m.list.Title = "Plugin Settings: " + pluginItem.Name
	items := make([]list.Item, 0, len(pluginItem.Manifest.Config))
	for _, field := range pluginItem.Manifest.Config {
		value := config.GetPlugin(m.opts.Config, pluginItem.Name, field.Key)
		if field.Secret {
			value = "secret stored outside config"
		}
		if value == "" && !field.Secret {
			value = "(unset)"
		}
		desc := fmt.Sprintf("%s - %s", field.Type, value)
		if field.Description != "" {
			desc += " - " + field.Description
		}
		items = append(items, item{field.Key, desc, "config-plugin-set", pluginConfigSelection{plugin: pluginItem, field: field}})
	}
	if len(items) == 0 {
		items = append(items, item{"No settings", "This plugin does not declare configurable fields", "noop", nil})
	}
	m.list.SetItems(items)
}

func (m *Model) showSecrets() {
	m.screen = screenSecrets
	m.list.Title = "Secrets"
	m.list.SetItems([]list.Item{
		item{"Set global secret", "Prompt for key and value", "secret-set-global", nil},
		item{"Set plugin secret", "Prompt for plugin scope, key and value", "secret-set-plugin-scope", nil},
		item{"Delete global secret", "Prompt for key", "secret-delete-global", nil},
		item{"Delete plugin secret", "Prompt for plugin scope and key", "secret-delete-plugin-scope", nil},
	})
}

func (m *Model) refreshScreen() {
	switch m.screen {
	case screenInstalled:
		m.showInstalled()
	case screenAvailable:
		m.showAvailable()
	case screenCommands:
		m.showCommands()
	case screenConfig:
		m.showConfig()
	default:
		m.showDashboard()
	}
}

func (m Model) activateSelected() (tea.Model, tea.Cmd) {
	selected, ok := m.list.SelectedItem().(item)
	if !ok {
		return m, nil
	}
	switch selected.action {
	case "installed":
		m.showInstalled()
	case "available":
		m.showAvailable()
		if strings.TrimSpace(m.opts.Config.Repositories.PluginsURL) == "" {
			return m, nil
		}
		return m, m.loadAvailable()
	case "commands":
		m.showCommands()
	case "config":
		m.showConfig()
	case "config-global":
		m.showGlobalConfig()
	case "config-plugins":
		m.showPluginConfigList()
	case "config-plugin":
		m.showPluginConfig(selected.value.(plugin.Installed))
	case "secrets":
		m.showSecrets()
	case "install":
		p := selected.value.(repo.PluginSummary)
		return m, m.installPlugin(p.Name)
	case "plugin":
		p := selected.value.(plugin.Installed)
		return m, m.updatePlugin(p.Name)
	case "command":
		w := selected.value.(commandWizard)
		w.args = []string{}
		w.nextArg = 0
		m.wizard = &w
		return m.nextCommandArg()
	case "arg-value":
		if m.wizard != nil {
			m.wizard.args = append(m.wizard.args, selected.value.(string))
			return m.nextCommandArg()
		}
	case "config-set":
		m.previous = screenConfig
		m.screen = screenInput
		m.inputKey = selected.value.(string)
		m.inputPlugin = ""
		m.inputScope = ""
		current, _ := config.Get(m.opts.Config, m.inputKey)
		m.inputTitle = "Set " + m.inputKey
		m.input.SetValue(current)
		m.input.Focus()
	case "config-plugin-set":
		selection := selected.value.(pluginConfigSelection)
		m.previous = screenConfig
		m.screen = screenInput
		m.inputKey = ""
		m.inputScope = ""
		m.inputPlugin = selection.plugin.Name
		m.inputPluginField = selection.field
		current := config.GetPlugin(m.opts.Config, selection.plugin.Name, selection.field.Key)
		if selection.field.Secret {
			current = ""
		}
		m.inputTitle = "Set " + selection.plugin.Name + "." + selection.field.Key
		if selection.field.Secret {
			m.inputTitle += " (secret)"
		}
		m.input.SetValue(current)
		m.input.Focus()
	case "secret-set-global":
		m.startSecretInput("global", "", "Secret key", 1)
	case "secret-set-plugin-scope":
		m.startSecretInput("", "", "Plugin name", 10)
	case "secret-delete-global":
		m.startSecretInput("global", "", "Secret key to delete", 3)
	case "secret-delete-plugin-scope":
		m.startSecretInput("", "", "Plugin name", 30)
	}
	return m, nil
}

func (m Model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = m.previous
		m.refreshScreen()
		return m, nil
	case "enter":
		value := m.input.Value()
		switch {
		case m.inputKey != "" && m.inputScope == "":
			key := m.inputKey
			cfg := m.opts.Config
			if err := config.Set(&cfg, key, value); err != nil {
				m.err = err.Error()
				return m, nil
			}
			m.opts.Config = cfg
			m.opts.Plugins.Config = cfg
			m.opts.Plugins.Repo = repo.New(cfg.Repositories.PluginsURL)
			m.opts.Secrets = secrets.New(m.opts.Paths, cfg.Security.SecretsBackend)
			return m, func() tea.Msg {
				err := config.Save(m.opts.Paths.ConfigPath, cfg)
				return actionMsg{status: "saved " + key, err: err}
			}
		case m.inputPlugin != "":
			pluginName := m.inputPlugin
			field := m.inputPluginField
			if field.Secret {
				return m, func() tea.Msg {
					err := m.opts.Secrets.Set("plugin:"+pluginName, field.Key, value)
					return actionMsg{status: "saved secret " + pluginName + "." + field.Key, err: err}
				}
			}
			if err := validateConfigValue(field, value); err != nil {
				m.err = err.Error()
				return m, nil
			}
			cfg := m.opts.Config
			config.SetPlugin(&cfg, pluginName, field.Key, value)
			m.opts.Config = cfg
			m.opts.Plugins.Config = cfg
			return m, func() tea.Msg {
				err := config.Save(m.opts.Paths.ConfigPath, cfg)
				return actionMsg{status: "saved " + pluginName + "." + field.Key, err: err}
			}
		case m.wizard != nil:
			m.wizard.args = append(m.wizard.args, value)
			m.input.SetValue("")
			return m.nextCommandArg()
		default:
			return m.handleSecretInput(value)
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) nextCommandArg() (tea.Model, tea.Cmd) {
	if m.wizard == nil {
		m.showCommands()
		return m, nil
	}
	args := m.wizard.command.Args
	for m.wizard.nextArg < len(args) && !args[m.wizard.nextArg].Required {
		m.wizard.nextArg++
	}
	if m.wizard.nextArg >= len(args) {
		pluginName := m.wizard.plugin.Name
		runArgs := append([]string{}, m.wizard.args...)
		m.status = "running forge " + pluginName + " " + strings.Join(runArgs, " ")
		return m, m.runPlugin(pluginName, runArgs)
	}
	arg := args[m.wizard.nextArg]
	m.wizard.nextArg++
	values := m.valuesFor(arg)
	if len(values) > 0 {
		m.previous = screenCommands
		m.screen = screenChoice
		m.list.Title = "Choose " + arg.Name
		items := make([]list.Item, 0, len(values))
		for _, value := range values {
			items = append(items, item{value, arg.Description, "arg-value", value})
		}
		m.list.SetItems(items)
		return m, nil
	}
	m.previous = screenCommands
	m.screen = screenInput
	m.inputTitle = "Enter " + arg.Name
	m.input.SetValue("")
	m.input.Focus()
	return m, nil
}

func (m Model) valuesFor(arg plugin.ArgSpec) []string {
	switch {
	case strings.HasPrefix(arg.ValueProvider, "global."):
		values := m.opts.Config.Globals[strings.TrimPrefix(arg.ValueProvider, "global.")]
		if len(values) > 0 {
			return values
		}
	case strings.HasPrefix(arg.ValueProvider, "static:"):
		return splitCSV(strings.TrimPrefix(arg.ValueProvider, "static:"))
	}
	return arg.FallbackValues
}

func (m Model) handleSecretInput(value string) (tea.Model, tea.Cmd) {
	switch m.inputStep {
	case 1:
		m.secretKey = value
		m.startSecretInput(m.inputScope, m.secretKey, "Secret value", 2)
		return m, nil
	case 2:
		scope, key := m.inputScope, m.secretKey
		return m, func() tea.Msg {
			return actionMsg{status: "saved secret " + scope + "/" + key, err: m.opts.Secrets.Set(scope, key, value)}
		}
	case 3:
		scope, key := m.inputScope, value
		return m, func() tea.Msg {
			return actionMsg{status: "deleted secret " + scope + "/" + key, err: m.opts.Secrets.Delete(scope, key)}
		}
	case 10:
		m.startSecretInput("plugin:"+value, "", "Secret key", 1)
		return m, nil
	case 30:
		m.startSecretInput("plugin:"+value, "", "Secret key to delete", 3)
		return m, nil
	}
	m.showSecrets()
	return m, nil
}

func (m *Model) startSecretInput(scope, key, title string, step int) {
	m.previous = screenSecrets
	m.screen = screenInput
	m.inputScope = scope
	m.secretKey = key
	m.inputKey = ""
	m.inputPlugin = ""
	m.inputPluginField = plugin.ConfigSpec{}
	m.inputStep = step
	m.inputTitle = title
	m.input.SetValue("")
	m.input.Focus()
}

func (m Model) installPlugin(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.opts.Plugins.Install(context.Background(), name, "", m.opts.Config.Repositories.Channel)
		return actionMsg{status: "installed " + name, err: err}
	}
}

func (m Model) updatePlugin(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.opts.Plugins.Update(context.Background(), name)
		return actionMsg{status: "updated " + name, err: err}
	}
}

func (m Model) removePlugin(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.opts.Plugins.Remove(name)
		return actionMsg{status: "removed " + name, err: err}
	}
}

func (m Model) runPlugin(name string, args []string) tea.Cmd {
	cmd, err := m.opts.Plugins.Command(context.Background(), name, args, true)
	if err != nil {
		return func() tea.Msg { return runDoneMsg{err: err} }
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return runDoneMsg{err: err}
	})
}

func (m Model) loadInstalled() tea.Cmd {
	return func() tea.Msg {
		items, err := m.opts.Plugins.List()
		return installedMsg{items: items, err: err}
	}
}

func (m Model) loadAvailable() tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(m.opts.Config.Repositories.PluginsURL) == "" {
			return availableMsg{err: config.RequirePluginsURL(m.opts.Config)}
		}
		idx, err := m.opts.Plugins.Available(context.Background())
		return availableMsg{items: idx.Plugins, err: err}
	}
}

func configItem(key, value string) list.Item {
	return item{key, value, "config-set", key}
}

func validateConfigValue(field plugin.ConfigSpec, value string) error {
	switch field.Type {
	case "", "string", "secret_string":
		return nil
	case "bool":
		_, err := strconv.ParseBool(value)
		return err
	case "int":
		_, err := strconv.Atoi(value)
		return err
	case "string_list":
		return nil
	default:
		return fmt.Errorf("unsupported config type %q for %s", field.Type, field.Key)
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func mask(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "********"
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Run(opts Options) error {
	_, err := tea.NewProgram(New(opts), tea.WithAltScreen()).Run()
	return err
}
