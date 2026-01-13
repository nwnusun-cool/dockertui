package components

import "github.com/charmbracelet/bubbles/key"

// KeyMap 定义全局快捷键映射（使用 bubbles/key 管理）
type KeyMap struct {
	// 全局快捷键
	Quit key.Binding
	Help key.Binding
	Back key.Binding
	
	// 导航快捷键
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding
	
	// 容器列表快捷键
	Enter      key.Binding
	Refresh    key.Binding
	ViewLogs   key.Binding
	ExecShell  key.Binding
	Containers key.Binding
	
	// 日志视图快捷键
	ToggleFollow key.Binding
	ToggleWrap   key.Binding
}

// DefaultKeyMap 返回默认的快捷键映射
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// 全局快捷键
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "Quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "Show Help"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "b"),
			key.WithHelp("esc/b", "Go Back"),
		),
		
		// 导航快捷键（vim 风格）
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "Move Up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "Move Down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("ctrl+u", "pgup"),
			key.WithHelp("ctrl+u", "Page Up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("ctrl+d", "pgdown"),
			key.WithHelp("ctrl+d", "Page Down"),
		),
		Home: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g", "Go to Top"),
		),
		End: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G", "Go to Bottom"),
		),
		
		// 容器列表快捷键
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "View Details"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "Refresh"),
		),
		ViewLogs: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "View Logs"),
		),
		ExecShell: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "Enter Shell"),
		),
		Containers: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "Containers"),
		),
		
		// 日志视图快捷键
		ToggleFollow: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "Toggle Follow"),
		),
		ToggleWrap: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "Toggle Wrap"),
		),
	}
}

// ShortHelp 返回简短的帮助信息（用于底部状态栏）
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp 返回完整的帮助信息（用于帮助面板）
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown},      // 导航
		{k.Enter, k.ViewLogs, k.ExecShell},        // 操作
		{k.Refresh, k.Back, k.Help, k.Quit},       // 其他
	}
}
