package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/docker"
)

// NetworkCreateField 创建网络表单字段
type NetworkCreateField int

const (
	FieldNetworkName NetworkCreateField = iota
	FieldNetworkDriver
	FieldNetworkSubnet
	FieldNetworkGateway
	FieldNetworkIPRange
	FieldNetworkInternal
	FieldNetworkAttachable
	FieldNetworkIPv6
)

// 驱动选项
var networkDriverOptions = []string{"bridge", "macvlan", "host", "none"}

// 表单样式
var (
	networkFormTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("220")).
				Bold(true)

	networkFormLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("81")).
				Width(14)

	networkFormInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	networkFormInputActiveStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("220")).
					Bold(true)

	networkFormHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

	networkFormCheckboxStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("82"))

	networkFormButtonStyle = lipgloss.NewStyle().
				Padding(0, 2)

	networkFormButtonActiveStyle = lipgloss.NewStyle().
					Padding(0, 2).
					Reverse(true).
					Bold(true)

	networkFormErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196"))
)

// NetworkCreateView 创建网络视图
type NetworkCreateView struct {
	dockerClient docker.Client

	// UI 尺寸
	width  int
	height int

	// 表单字段值
	name       string
	driver     int  // 驱动选项索引
	subnet     string
	gateway    string
	ipRange    string
	internal   bool
	attachable bool
	ipv6       bool

	// 当前焦点字段
	activeField NetworkCreateField

	// 按钮焦点：0=Cancel, 1=Create
	buttonFocus int
	onButtons   bool // 是否在按钮区域

	// 状态
	creating bool
	errorMsg string

	// 回调
	onCreated  func(networkID string) // 创建成功回调
	onCanceled func()                 // 取消回调
}

// NewNetworkCreateView 创建网络创建视图
func NewNetworkCreateView(dockerClient docker.Client) *NetworkCreateView {
	return &NetworkCreateView{
		dockerClient: dockerClient,
		driver:       0, // 默认 bridge
		attachable:   true,
		activeField:  FieldNetworkName,
	}
}

// SetCallbacks 设置回调函数
func (v *NetworkCreateView) SetCallbacks(onCreated func(string), onCanceled func()) {
	v.onCreated = onCreated
	v.onCanceled = onCanceled
}

// Reset 重置表单
func (v *NetworkCreateView) Reset() {
	v.name = ""
	v.driver = 0
	v.subnet = ""
	v.gateway = ""
	v.ipRange = ""
	v.internal = false
	v.attachable = true
	v.ipv6 = false
	v.activeField = FieldNetworkName
	v.buttonFocus = 0
	v.onButtons = false
	v.creating = false
	v.errorMsg = ""
}

// Init 初始化视图
func (v *NetworkCreateView) Init() tea.Cmd {
	return nil
}

// Update 处理消息
func (v *NetworkCreateView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case networkCreateSuccessMsg:
		v.creating = false
		if v.onCreated != nil {
			v.onCreated(msg.networkID)
		}
		return v, nil

	case networkCreateErrorMsg:
		v.creating = false
		v.errorMsg = msg.err.Error()
		return v, nil

	case tea.KeyMsg:
		if v.creating {
			return v, nil // 创建中不处理按键
		}

		switch msg.String() {
		case "esc":
			if v.onCanceled != nil {
				v.onCanceled()
			}
			return v, nil

		case "tab", "down", "j":
			v.moveNext()
			return v, nil

		case "shift+tab", "up", "k":
			v.movePrev()
			return v, nil

		case "enter":
			if v.onButtons {
				if v.buttonFocus == 0 {
					// Cancel
					if v.onCanceled != nil {
						v.onCanceled()
					}
				} else {
					// Create
					return v, v.createNetwork()
				}
			} else {
				// 在输入字段按 Enter，移动到下一个字段
				v.moveNext()
			}
			return v, nil

		case "left", "h":
			if v.onButtons {
				v.buttonFocus = 0
			} else if v.activeField == FieldNetworkDriver {
				// 切换驱动选项
				if v.driver > 0 {
					v.driver--
				}
			}
			return v, nil

		case "right", "l":
			if v.onButtons {
				v.buttonFocus = 1
			} else if v.activeField == FieldNetworkDriver {
				// 切换驱动选项
				if v.driver < len(networkDriverOptions)-1 {
					v.driver++
				}
			}
			return v, nil

		case " ":
			// 空格切换复选框
			switch v.activeField {
			case FieldNetworkInternal:
				v.internal = !v.internal
			case FieldNetworkAttachable:
				v.attachable = !v.attachable
			case FieldNetworkIPv6:
				v.ipv6 = !v.ipv6
			}
			return v, nil

		case "backspace":
			// 删除字符
			v.handleBackspace()
			return v, nil

		default:
			// 输入字符
			if len(msg.String()) == 1 {
				v.handleInput(msg.String())
			}
			return v, nil
		}
	}

	return v, nil
}

// moveNext 移动到下一个字段
func (v *NetworkCreateView) moveNext() {
	if v.onButtons {
		// 已经在按钮区域，切换按钮
		v.buttonFocus = 1 - v.buttonFocus
		return
	}

	if v.activeField == FieldNetworkIPv6 {
		// 最后一个字段，移动到按钮区域
		v.onButtons = true
		v.buttonFocus = 1 // 默认选中 Create
	} else {
		v.activeField++
	}
}

// movePrev 移动到上一个字段
func (v *NetworkCreateView) movePrev() {
	if v.onButtons {
		if v.buttonFocus == 0 {
			// 从 Cancel 返回到最后一个字段
			v.onButtons = false
			v.activeField = FieldNetworkIPv6
		} else {
			v.buttonFocus = 0
		}
		return
	}

	if v.activeField > 0 {
		v.activeField--
	}
}

// handleInput 处理输入
func (v *NetworkCreateView) handleInput(char string) {
	switch v.activeField {
	case FieldNetworkName:
		v.name += char
	case FieldNetworkSubnet:
		v.subnet += char
	case FieldNetworkGateway:
		v.gateway += char
	case FieldNetworkIPRange:
		v.ipRange += char
	}
}

// handleBackspace 处理退格
func (v *NetworkCreateView) handleBackspace() {
	switch v.activeField {
	case FieldNetworkName:
		if len(v.name) > 0 {
			v.name = v.name[:len(v.name)-1]
		}
	case FieldNetworkSubnet:
		if len(v.subnet) > 0 {
			v.subnet = v.subnet[:len(v.subnet)-1]
		}
	case FieldNetworkGateway:
		if len(v.gateway) > 0 {
			v.gateway = v.gateway[:len(v.gateway)-1]
		}
	case FieldNetworkIPRange:
		if len(v.ipRange) > 0 {
			v.ipRange = v.ipRange[:len(v.ipRange)-1]
		}
	}
}

// View 渲染视图
func (v *NetworkCreateView) View() string {
	var s strings.Builder

	// 标题
	s.WriteString("\n  " + networkFormTitleStyle.Render("🌐 Create Network") + "\n\n")

	// 错误信息
	if v.errorMsg != "" {
		s.WriteString("  " + networkFormErrorStyle.Render("❌ "+v.errorMsg) + "\n\n")
	}

	// 创建中状态
	if v.creating {
		s.WriteString("  " + networkFormHintStyle.Render("⏳ 正在创建网络...") + "\n")
		return s.String()
	}

	// 表单字段
	s.WriteString(v.renderField(FieldNetworkName, "Name", v.name, "网络名称（必填）"))
	s.WriteString(v.renderDriverField())
	s.WriteString(v.renderField(FieldNetworkSubnet, "Subnet", v.subnet, "子网 CIDR，如 172.20.0.0/16"))
	s.WriteString(v.renderField(FieldNetworkGateway, "Gateway", v.gateway, "网关地址，如 172.20.0.1"))
	s.WriteString(v.renderField(FieldNetworkIPRange, "IP Range", v.ipRange, "IP 范围（可选）"))
	s.WriteString(v.renderCheckbox(FieldNetworkInternal, "Internal", v.internal, "内部网络（不能访问外部）"))
	s.WriteString(v.renderCheckbox(FieldNetworkAttachable, "Attachable", v.attachable, "允许手动连接容器"))
	s.WriteString(v.renderCheckbox(FieldNetworkIPv6, "IPv6", v.ipv6, "启用 IPv6"))

	// 按钮
	s.WriteString("\n" + v.renderButtons())

	// 快捷键提示
	s.WriteString("\n\n" + v.renderHints())

	return s.String()
}

// renderField 渲染输入字段
func (v *NetworkCreateView) renderField(field NetworkCreateField, label, value, hint string) string {
	isActive := !v.onButtons && v.activeField == field

	labelStr := networkFormLabelStyle.Render(label + ":")

	// 显示值
	displayValue := value
	if displayValue == "" {
		displayValue = "(empty)"
	}

	// 根据是否活动选择样式
	var valueStr string
	if isActive {
		// 活动状态：显示光标，使用高亮样式
		valueStr = networkFormInputActiveStyle.Render(value + "█")
	} else {
		if value == "" {
			valueStr = networkFormHintStyle.Render("(empty)")
		} else {
			valueStr = networkFormInputStyle.Render(value)
		}
	}

	hintStr := networkFormHintStyle.Render(hint)

	return fmt.Sprintf("  %s %s  %s\n", labelStr, valueStr, hintStr)
}

// renderDriverField 渲染驱动选择字段
func (v *NetworkCreateView) renderDriverField() string {
	isActive := !v.onButtons && v.activeField == FieldNetworkDriver

	labelStr := networkFormLabelStyle.Render("Driver:")

	// 构建驱动选项显示
	var options []string
	for i, opt := range networkDriverOptions {
		if i == v.driver {
			if isActive {
				options = append(options, lipgloss.NewStyle().Reverse(true).Bold(true).Render(" "+opt+" "))
			} else {
				options = append(options, lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true).Render("["+opt+"]"))
			}
		} else {
			options = append(options, networkFormHintStyle.Render(opt))
		}
	}

	optionsStr := strings.Join(options, "  ")
	hintStr := networkFormHintStyle.Render("← → 切换")

	return fmt.Sprintf("  %s %s  %s\n", labelStr, optionsStr, hintStr)
}

// renderCheckbox 渲染复选框
func (v *NetworkCreateView) renderCheckbox(field NetworkCreateField, label string, checked bool, hint string) string {
	isActive := !v.onButtons && v.activeField == field

	labelStr := networkFormLabelStyle.Render(label + ":")

	checkStr := "[ ]"
	if checked {
		checkStr = networkFormCheckboxStyle.Render("[✓]")
	}

	if isActive {
		checkStr = lipgloss.NewStyle().Reverse(true).Render(checkStr)
	}

	hintStr := networkFormHintStyle.Render(hint + " (空格切换)")

	return fmt.Sprintf("  %s %s  %s\n", labelStr, checkStr, hintStr)
}

// renderButtons 渲染按钮
func (v *NetworkCreateView) renderButtons() string {
	cancelStyle := networkFormButtonStyle
	createStyle := networkFormButtonStyle

	if v.onButtons {
		if v.buttonFocus == 0 {
			cancelStyle = networkFormButtonActiveStyle
		} else {
			createStyle = networkFormButtonActiveStyle
		}
	}

	cancelBtn := cancelStyle.Render("[ Cancel ]")
	createBtn := createStyle.Render("[ Create ]")

	return "  " + strings.Repeat(" ", 14) + cancelBtn + "    " + createBtn
}

// renderHints 渲染快捷键提示
func (v *NetworkCreateView) renderHints() string {
	hints := []string{
		networkFormHintStyle.Render("Tab/↑↓") + " 切换字段",
		networkFormHintStyle.Render("Space") + " 切换复选框",
		networkFormHintStyle.Render("Enter") + " 确认",
		networkFormHintStyle.Render("Esc") + " 取消",
	}

	return "  " + strings.Join(hints, "  │  ")
}

// SetSize 设置视图尺寸
func (v *NetworkCreateView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// createNetwork 创建网络
func (v *NetworkCreateView) createNetwork() tea.Cmd {
	// 验证
	if strings.TrimSpace(v.name) == "" {
		v.errorMsg = "网络名称不能为空"
		return nil
	}

	v.creating = true
	v.errorMsg = ""

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		opts := docker.NetworkCreateOptions{
			Name:       strings.TrimSpace(v.name),
			Driver:     networkDriverOptions[v.driver],
			Subnet:     strings.TrimSpace(v.subnet),
			Gateway:    strings.TrimSpace(v.gateway),
			IPRange:    strings.TrimSpace(v.ipRange),
			Internal:   v.internal,
			Attachable: v.attachable,
			IPv6:       v.ipv6,
		}

		networkID, err := v.dockerClient.CreateNetwork(ctx, opts)
		if err != nil {
			return networkCreateErrorMsg{err: err}
		}

		return networkCreateSuccessMsg{networkID: networkID}
	}
}

// 消息类型
type networkCreateSuccessMsg struct {
	networkID string
}

type networkCreateErrorMsg struct {
	err error
}
