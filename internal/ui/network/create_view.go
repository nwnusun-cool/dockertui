package network

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/docker"
)

// CreateField 创建网络表单字段
type CreateField int

const (
	FieldName CreateField = iota
	FieldDriver
	FieldSubnet
	FieldGateway
	FieldIPRange
	FieldInternal
	FieldAttachable
	FieldIPv6
)

// 驱动选项
var driverOptions = []string{"bridge", "macvlan", "host", "none"}

// CreateView 创建网络视图
type CreateView struct {
	dockerClient docker.Client
	width, height int
	name, subnet, gateway, ipRange string
	driver int
	internal, attachable, ipv6 bool
	activeField CreateField
	buttonFocus int
	onButtons, creating bool
	errorMsg string
	onCreated func(networkID string)
	onCanceled func()
}

// NewCreateView 创建网络创建视图
func NewCreateView(dockerClient docker.Client) *CreateView {
	return &CreateView{dockerClient: dockerClient, driver: 0, attachable: true, activeField: FieldName}
}

// SetCallbacks 设置回调函数
func (v *CreateView) SetCallbacks(onCreated func(string), onCanceled func()) {
	v.onCreated = onCreated
	v.onCanceled = onCanceled
}

// Reset 重置表单
func (v *CreateView) Reset() {
	v.name, v.subnet, v.gateway, v.ipRange = "", "", "", ""
	v.driver = 0
	v.internal, v.attachable, v.ipv6 = false, true, false
	v.activeField = FieldName
	v.buttonFocus = 0
	v.onButtons, v.creating = false, false
	v.errorMsg = ""
}

// Init 初始化视图
func (v *CreateView) Init() tea.Cmd { return nil }

// Update 处理消息
func (v *CreateView) Update(msg tea.Msg) (*CreateView, tea.Cmd) {
	switch msg := msg.(type) {
	case NetworkCreateSuccessMsg:
		v.creating = false
		if v.onCreated != nil { v.onCreated(msg.NetworkID) }
		return v, nil
	case NetworkCreateErrorMsg:
		v.creating = false
		v.errorMsg = msg.Err.Error()
		return v, nil
	case tea.KeyMsg:
		if v.creating { return v, nil }
		switch msg.String() {
		case "esc":
			if v.onCanceled != nil { v.onCanceled() }
			return v, nil
		case "tab", "down", "j": v.moveNext()
		case "shift+tab", "up", "k": v.movePrev()
		case "enter":
			if v.onButtons {
				if v.buttonFocus == 0 {
					if v.onCanceled != nil { v.onCanceled() }
				} else {
					return v, v.createNetwork()
				}
			} else { v.moveNext() }
		case "left", "h":
			if v.onButtons { v.buttonFocus = 0 } else if v.activeField == FieldDriver && v.driver > 0 { v.driver-- }
		case "right", "l":
			if v.onButtons { v.buttonFocus = 1 } else if v.activeField == FieldDriver && v.driver < len(driverOptions)-1 { v.driver++ }
		case " ":
			switch v.activeField {
			case FieldInternal: v.internal = !v.internal
			case FieldAttachable: v.attachable = !v.attachable
			case FieldIPv6: v.ipv6 = !v.ipv6
			}
		case "backspace": v.handleBackspace()
		default:
			if len(msg.String()) == 1 { v.handleInput(msg.String()) }
		}
	}
	return v, nil
}

func (v *CreateView) moveNext() {
	if v.onButtons { v.buttonFocus = 1 - v.buttonFocus; return }
	if v.activeField == FieldIPv6 { v.onButtons = true; v.buttonFocus = 1 } else { v.activeField++ }
}

func (v *CreateView) movePrev() {
	if v.onButtons {
		if v.buttonFocus == 0 { v.onButtons = false; v.activeField = FieldIPv6 } else { v.buttonFocus = 0 }
		return
	}
	if v.activeField > 0 { v.activeField-- }
}

func (v *CreateView) handleInput(char string) {
	switch v.activeField {
	case FieldName: v.name += char
	case FieldSubnet: v.subnet += char
	case FieldGateway: v.gateway += char
	case FieldIPRange: v.ipRange += char
	}
}

func (v *CreateView) handleBackspace() {
	switch v.activeField {
	case FieldName: if len(v.name) > 0 { v.name = v.name[:len(v.name)-1] }
	case FieldSubnet: if len(v.subnet) > 0 { v.subnet = v.subnet[:len(v.subnet)-1] }
	case FieldGateway: if len(v.gateway) > 0 { v.gateway = v.gateway[:len(v.gateway)-1] }
	case FieldIPRange: if len(v.ipRange) > 0 { v.ipRange = v.ipRange[:len(v.ipRange)-1] }
	}
}

// View 渲染视图
func (v *CreateView) View() string {
	var s strings.Builder
	s.WriteString("\n  " + FormTitleStyle.Render("🌐 Create Network") + "\n\n")
	if v.errorMsg != "" { s.WriteString("  " + FormErrorStyle.Render("❌ "+v.errorMsg) + "\n\n") }
	if v.creating { s.WriteString("  " + FormHintStyle.Render("⏳ 正在创建网络...") + "\n"); return s.String() }
	s.WriteString(v.renderField(FieldName, "Name", v.name, "网络名称（必填）"))
	s.WriteString(v.renderDriverField())
	s.WriteString(v.renderField(FieldSubnet, "Subnet", v.subnet, "子网 CIDR，如 172.20.0.0/16"))
	s.WriteString(v.renderField(FieldGateway, "Gateway", v.gateway, "网关地址，如 172.20.0.1"))
	s.WriteString(v.renderField(FieldIPRange, "IP Range", v.ipRange, "IP 范围（可选）"))
	s.WriteString(v.renderCheckbox(FieldInternal, "Internal", v.internal, "内部网络（不能访问外部）"))
	s.WriteString(v.renderCheckbox(FieldAttachable, "Attachable", v.attachable, "允许手动连接容器"))
	s.WriteString(v.renderCheckbox(FieldIPv6, "IPv6", v.ipv6, "启用 IPv6"))
	s.WriteString("\n" + v.renderButtons())
	s.WriteString("\n\n" + v.renderHints())
	return s.String()
}

// SetSize 设置视图尺寸
func (v *CreateView) SetSize(width, height int) { v.width = width; v.height = height }

func (v *CreateView) renderField(field CreateField, label, value, hint string) string {
	isActive := !v.onButtons && v.activeField == field
	labelStr := FormLabelStyle.Render(label + ":")
	var valueStr string
	if isActive {
		valueStr = FormInputActiveStyle.Render(value + "█")
	} else if value == "" {
		valueStr = FormHintStyle.Render("(empty)")
	} else {
		valueStr = FormInputStyle.Render(value)
	}
	return fmt.Sprintf("  %s %s  %s\n", labelStr, valueStr, FormHintStyle.Render(hint))
}

func (v *CreateView) renderDriverField() string {
	isActive := !v.onButtons && v.activeField == FieldDriver
	labelStr := FormLabelStyle.Render("Driver:")
	var options []string
	for i, opt := range driverOptions {
		if i == v.driver {
			if isActive {
				options = append(options, lipgloss.NewStyle().Reverse(true).Bold(true).Render(" "+opt+" "))
			} else {
				options = append(options, lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true).Render("["+opt+"]"))
			}
		} else {
			options = append(options, FormHintStyle.Render(opt))
		}
	}
	return fmt.Sprintf("  %s %s  %s\n", labelStr, strings.Join(options, "  "), FormHintStyle.Render("← → 切换"))
}

func (v *CreateView) renderCheckbox(field CreateField, label string, checked bool, hint string) string {
	isActive := !v.onButtons && v.activeField == field
	labelStr := FormLabelStyle.Render(label + ":")
	checkStr := "[ ]"
	if checked { checkStr = FormCheckboxStyle.Render("[✓]") }
	if isActive { checkStr = lipgloss.NewStyle().Reverse(true).Render(checkStr) }
	return fmt.Sprintf("  %s %s  %s\n", labelStr, checkStr, FormHintStyle.Render(hint+" (空格切换)"))
}

func (v *CreateView) renderButtons() string {
	cancelStyle, createStyle := FormButtonStyle, FormButtonStyle
	if v.onButtons {
		if v.buttonFocus == 0 { cancelStyle = FormButtonActiveStyle } else { createStyle = FormButtonActiveStyle }
	}
	return "  " + strings.Repeat(" ", 14) + cancelStyle.Render("[ Cancel ]") + "    " + createStyle.Render("[ Create ]")
}

func (v *CreateView) renderHints() string {
	hints := []string{
		FormHintStyle.Render("Tab/↑↓") + " 切换字段",
		FormHintStyle.Render("Space") + " 切换复选框",
		FormHintStyle.Render("Enter") + " 确认",
		FormHintStyle.Render("Esc") + " 取消",
	}
	return "  " + strings.Join(hints, "  │  ")
}

func (v *CreateView) createNetwork() tea.Cmd {
	if strings.TrimSpace(v.name) == "" { v.errorMsg = "网络名称不能为空"; return nil }
	v.creating = true
	v.errorMsg = ""
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		opts := docker.NetworkCreateOptions{
			Name: strings.TrimSpace(v.name), Driver: driverOptions[v.driver],
			Subnet: strings.TrimSpace(v.subnet), Gateway: strings.TrimSpace(v.gateway),
			IPRange: strings.TrimSpace(v.ipRange), Internal: v.internal, Attachable: v.attachable, IPv6: v.ipv6,
		}
		networkID, err := v.dockerClient.CreateNetwork(ctx, opts)
		if err != nil { return NetworkCreateErrorMsg{Err: err} }
		return NetworkCreateSuccessMsg{NetworkID: networkID}
	}
}
