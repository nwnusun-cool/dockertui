package container

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/docker"
)

// 编辑视图样式定义
var (
	editBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("81")).
			Padding(1, 2)

	editTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Bold(true)

	editLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Width(14)

	editHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	editValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("81"))

	editSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("81")).
				Bold(true)

	editErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
)

// EditView 容器配置编辑视图
type EditView struct {
	// 容器信息
	container   *docker.Container
	containerID string
	details     *docker.ContainerDetails

	// 输入框
	cpuSharesInput textinput.Model // CPU 份额输入
	memoryInput    textinput.Model // 内存限制输入（MB）

	// 重启策略选择
	restartPolicies  []string        // 可选的重启策略
	restartPolicyIdx int             // 当前选中的重启策略索引
	maxRetriesInput  textinput.Model // on-failure 最大重试次数

	// UI 状态
	visible    bool
	width      int
	focusIndex int // 0=重启策略, 1=最大重试, 2=CPU份额, 3=内存, 4=取消, 5=确认
	errorMsg   string
}

// NewEditView 创建容器编辑视图
func NewEditView() *EditView {
	// CPU 份额输入框
	cpuInput := textinput.New()
	cpuInput.Placeholder = "1024"
	cpuInput.CharLimit = 10
	cpuInput.Width = 12
	cpuInput.Prompt = ""

	// 内存输入框
	memInput := textinput.New()
	memInput.Placeholder = "512"
	memInput.CharLimit = 10
	memInput.Width = 12
	memInput.Prompt = ""

	// 最大重试次数输入框
	retriesInput := textinput.New()
	retriesInput.Placeholder = "3"
	retriesInput.CharLimit = 5
	retriesInput.Width = 8
	retriesInput.Prompt = ""

	return &EditView{
		cpuSharesInput:   cpuInput,
		memoryInput:      memInput,
		maxRetriesInput:  retriesInput,
		restartPolicies:  []string{"no", "always", "on-failure", "unless-stopped"},
		restartPolicyIdx: 0,
		visible:          false,
		focusIndex:       0,
	}
}

// Show 显示编辑视图
func (v *EditView) Show(container *docker.Container, details *docker.ContainerDetails) {
	v.visible = true
	v.container = container
	v.containerID = container.ID
	v.details = details
	v.errorMsg = ""

	// 解析当前重启策略
	currentPolicy := "no"
	maxRetries := 0
	if details != nil && details.RestartPolicy != "" {
		parts := strings.Split(details.RestartPolicy, ":")
		currentPolicy = parts[0]
		if len(parts) > 1 {
			maxRetries, _ = strconv.Atoi(parts[1])
		}
	}

	// 设置重启策略选择
	for i, p := range v.restartPolicies {
		if p == currentPolicy {
			v.restartPolicyIdx = i
			break
		}
	}

	// 设置最大重试次数
	if maxRetries > 0 {
		v.maxRetriesInput.SetValue(strconv.Itoa(maxRetries))
	} else {
		v.maxRetriesInput.SetValue("")
	}

	// 清空资源限制输入（留空表示不修改）
	v.cpuSharesInput.SetValue("")
	v.memoryInput.SetValue("")

	// 聚焦到重启策略
	v.focusIndex = 0
	v.updateInputFocus()
}

// Hide 隐藏编辑视图
func (v *EditView) Hide() {
	v.visible = false
	v.container = nil
	v.details = nil
	v.cpuSharesInput.Blur()
	v.memoryInput.Blur()
	v.maxRetriesInput.Blur()
}

// IsVisible 是否可见
func (v *EditView) IsVisible() bool {
	return v.visible
}

// GetConfig 获取编辑后的配置
func (v *EditView) GetConfig() docker.ContainerUpdateConfig {
	config := docker.ContainerUpdateConfig{
		RestartPolicy: v.restartPolicies[v.restartPolicyIdx],
	}

	// 解析最大重试次数
	if retriesStr := strings.TrimSpace(v.maxRetriesInput.Value()); retriesStr != "" {
		if retries, err := strconv.Atoi(retriesStr); err == nil && retries > 0 {
			config.RestartMaxRetries = retries
		}
	}

	// 解析 CPU 份额
	if cpuStr := strings.TrimSpace(v.cpuSharesInput.Value()); cpuStr != "" {
		if cpu, err := strconv.ParseInt(cpuStr, 10, 64); err == nil && cpu > 0 {
			config.CPUShares = cpu
		}
	}

	// 解析内存限制（MB -> 字节）
	if memStr := strings.TrimSpace(v.memoryInput.Value()); memStr != "" {
		if mem, err := strconv.ParseInt(memStr, 10, 64); err == nil && mem > 0 {
			config.Memory = mem * 1024 * 1024 // MB to bytes
		}
	}

	return config
}

// GetContainerID 获取容器 ID
func (v *EditView) GetContainerID() string {
	return v.containerID
}

// GetContainerName 获取容器名称
func (v *EditView) GetContainerName() string {
	if v.container != nil {
		return v.container.Name
	}
	return ""
}

// SetWidth 设置宽度
func (v *EditView) SetWidth(width int) {
	v.width = width
}

// Update 处理输入
// 返回值: (confirmed bool, handled bool, cmd tea.Cmd)
func (v *EditView) Update(msg tea.Msg) (bool, bool, tea.Cmd) {
	if !v.visible {
		return false, false, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		keyStr := msg.String()

		// Enter 键
		if msg.Type == tea.KeyEnter || keyStr == "enter" {
			if v.focusIndex == 5 { // 确认按钮
				return true, true, nil
			}
			if v.focusIndex == 4 { // 取消按钮
				v.Hide()
				return false, true, nil
			}
			// 其他情况，移动到下一个焦点
			v.nextFocus()
			return false, true, nil
		}

		// Esc 键
		if msg.Type == tea.KeyEsc || keyStr == "esc" {
			v.Hide()
			return false, true, nil
		}

		// Tab 键切换焦点
		if msg.Type == tea.KeyTab {
			v.nextFocus()
			return false, true, nil
		}

		// Shift+Tab 反向切换
		if msg.Type == tea.KeyShiftTab {
			v.prevFocus()
			return false, true, nil
		}

		// 上下键
		if msg.Type == tea.KeyUp || keyStr == "up" {
			v.prevFocus()
			return false, true, nil
		}
		if msg.Type == tea.KeyDown || keyStr == "down" {
			v.nextFocus()
			return false, true, nil
		}

		// 左右键在重启策略和按钮区域切换
		if v.focusIndex == 0 {
			// 重启策略选择
			if msg.Type == tea.KeyLeft || keyStr == "left" || keyStr == "h" {
				if v.restartPolicyIdx > 0 {
					v.restartPolicyIdx--
				}
				return false, true, nil
			}
			if msg.Type == tea.KeyRight || keyStr == "right" || keyStr == "l" {
				if v.restartPolicyIdx < len(v.restartPolicies)-1 {
					v.restartPolicyIdx++
				}
				return false, true, nil
			}
		}

		if v.focusIndex >= 4 {
			// 按钮区域
			if msg.Type == tea.KeyLeft || keyStr == "left" {
				if v.focusIndex == 5 {
					v.focusIndex = 4
				}
				return false, true, nil
			}
			if msg.Type == tea.KeyRight || keyStr == "right" {
				if v.focusIndex == 4 {
					v.focusIndex = 5
				}
				return false, true, nil
			}
		}
	}

	// 传递给当前聚焦的输入框
	var cmd tea.Cmd
	switch v.focusIndex {
	case 1:
		v.maxRetriesInput, cmd = v.maxRetriesInput.Update(msg)
	case 2:
		v.cpuSharesInput, cmd = v.cpuSharesInput.Update(msg)
	case 3:
		v.memoryInput, cmd = v.memoryInput.Update(msg)
	}

	return false, true, cmd
}

// nextFocus 切换到下一个焦点
func (v *EditView) nextFocus() {
	// 如果不是 on-failure，跳过最大重试次数
	if v.focusIndex == 0 && v.restartPolicies[v.restartPolicyIdx] != "on-failure" {
		v.focusIndex = 2 // 跳到 CPU 份额
	} else {
		v.focusIndex = (v.focusIndex + 1) % 6
	}
	v.updateInputFocus()
}

// prevFocus 切换到上一个焦点
func (v *EditView) prevFocus() {
	// 如果不是 on-failure，跳过最大重试次数
	if v.focusIndex == 2 && v.restartPolicies[v.restartPolicyIdx] != "on-failure" {
		v.focusIndex = 0 // 跳回重启策略
	} else {
		v.focusIndex = (v.focusIndex + 5) % 6
	}
	v.updateInputFocus()
}

// updateInputFocus 更新输入框焦点状态
func (v *EditView) updateInputFocus() {
	v.cpuSharesInput.Blur()
	v.memoryInput.Blur()
	v.maxRetriesInput.Blur()

	switch v.focusIndex {
	case 1:
		v.maxRetriesInput.Focus()
	case 2:
		v.cpuSharesInput.Focus()
	case 3:
		v.memoryInput.Focus()
	}
}

// View 渲染编辑视图
func (v *EditView) View() string {
	if !v.visible {
		return ""
	}

	// 标题
	containerName := ""
	if v.container != nil {
		containerName = v.container.Name
		if len(containerName) > 25 {
			containerName = containerName[:22] + "..."
		}
	}
	title := editTitleStyle.Render("⚙️  Edit Container Config: " + containerName)

	// 当前配置信息
	currentPolicy := "no"
	if v.details != nil && v.details.RestartPolicy != "" {
		currentPolicy = v.details.RestartPolicy
	}
	currentInfo := editHintStyle.Render("Current restart policy: ") + editValueStyle.Render(currentPolicy)

	// 重启策略选择
	restartLabel := editLabelStyle.Render("Restart Policy:")
	var policyOptions []string
	for i, p := range v.restartPolicies {
		if i == v.restartPolicyIdx {
			if v.focusIndex == 0 {
				policyOptions = append(policyOptions, editSelectedStyle.Render("["+p+"]"))
			} else {
				policyOptions = append(policyOptions, editValueStyle.Render("["+p+"]"))
			}
		} else {
			policyOptions = append(policyOptions, editHintStyle.Render(" "+p+" "))
		}
	}
	restartLine := restartLabel + " " + strings.Join(policyOptions, " ")

	// 最大重试次数（仅 on-failure 时显示）
	var retriesLine string
	if v.restartPolicies[v.restartPolicyIdx] == "on-failure" {
		retriesLabel := editLabelStyle.Render("Max Retries:")
		retriesInputStyle := lipgloss.NewStyle()
		if v.focusIndex == 1 {
			retriesInputStyle = retriesInputStyle.Foreground(lipgloss.Color("81"))
		}
		retriesLine = retriesLabel + " " + retriesInputStyle.Render(v.maxRetriesInput.View()) +
			editHintStyle.Render(" (times, 0=unlimited)")
	}

	// CPU 份额
	cpuLabel := editLabelStyle.Render("CPU Shares:")
	cpuInputStyle := lipgloss.NewStyle()
	if v.focusIndex == 2 {
		cpuInputStyle = cpuInputStyle.Foreground(lipgloss.Color("81"))
	}
	cpuLine := cpuLabel + " " + cpuInputStyle.Render(v.cpuSharesInput.View()) +
		editHintStyle.Render(" (default 1024, leave empty to keep unchanged)")

	// 内存限制
	memLabel := editLabelStyle.Render("Memory Limit:")
	memInputStyle := lipgloss.NewStyle()
	if v.focusIndex == 3 {
		memInputStyle = memInputStyle.Foreground(lipgloss.Color("81"))
	}
	memLine := memLabel + " " + memInputStyle.Render(v.memoryInput.View()) +
		editHintStyle.Render(" MB (must be greater than current usage)")

	// 按钮
	cancelBtnStyle := lipgloss.NewStyle().Padding(0, 2)
	okBtnStyle := lipgloss.NewStyle().Padding(0, 2)

	if v.focusIndex == 4 {
		cancelBtnStyle = cancelBtnStyle.Reverse(true).Bold(true)
	} else {
		cancelBtnStyle = cancelBtnStyle.Foreground(lipgloss.Color("245"))
	}

	if v.focusIndex == 5 {
		okBtnStyle = okBtnStyle.Reverse(true).Bold(true)
	} else {
		okBtnStyle = okBtnStyle.Foreground(lipgloss.Color("245"))
	}

	cancelBtn := cancelBtnStyle.Render("< Cancel >")
	okBtn := okBtnStyle.Render("< Confirm >")
	buttons := cancelBtn + "    " + okBtn

	// 错误信息
	var errorLine string
	if v.errorMsg != "" {
		errorLine = editErrorStyle.Render("❌ " + v.errorMsg)
	}

	// 提示
	hints := editHintStyle.Render("[Tab/↑↓=Switch] [←→=Select policy] [Enter=Confirm] [Esc=Cancel]")
	note := editHintStyle.Render("Note: CPU/Memory limits only supported on Linux native Docker")

	// 组合内容
	var contentParts []string
	contentParts = append(contentParts, title, "", currentInfo, "", restartLine)

	if retriesLine != "" {
		contentParts = append(contentParts, retriesLine)
	}

	contentParts = append(contentParts, "", cpuLine, memLine)

	if errorLine != "" {
		contentParts = append(contentParts, "", errorLine)
	}

	contentParts = append(contentParts, "", buttons, "", hints, note)

	content := lipgloss.JoinVertical(lipgloss.Left, contentParts...)

	// 计算框宽度
	boxWidth := v.width - 10
	if boxWidth < 65 {
		boxWidth = 65
	}
	if boxWidth > 75 {
		boxWidth = 75
	}

	box := editBoxStyle.Width(boxWidth).Render(content)

	// 居中
	if v.width > boxWidth+10 {
		leftPadding := (v.width - boxWidth - 4) / 2
		lines := strings.Split(box, "\n")
		for i, line := range lines {
			lines[i] = strings.Repeat(" ", leftPadding) + line
		}
		return strings.Join(lines, "\n")
	}

	return box
}
