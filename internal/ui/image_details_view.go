package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/docker"
)

// ImageDetailsTab 镜像详情标签页类型
type ImageDetailsTab int

const (
	TabBasicInfo ImageDetailsTab = iota // 基本信息
	TabUsage                            // 使用状态
	TabConfig                           // 配置信息
	TabEnvVars                          // 环境变量
	TabHistory                          // 构建历史
	TabLabels                           // 标签信息
)

// 标签页名称
var imageTabNames = []string{
	"Basic Info",
	"Usage",
	"Config",
	"Env Vars",
	"History",
	"Labels",
}

// 镜像详情视图样式
var (
	imageDetailsTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	imageDetailsLabelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81")).
		Width(16)

	imageDetailsValueStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	imageDetailsBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	imageTabActiveStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true).
		Underline(true)

	imageTabInactiveStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	imageDetailsHintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	imageDetailsKeyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81"))

	imageContainerRunningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	imageContainerStoppedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
)

// ImageDetailsView 镜像详情视图
type ImageDetailsView struct {
	dockerClient docker.Client

	// UI 尺寸
	width  int
	height int

	// 数据
	image   *docker.Image        // 镜像基本信息
	details *docker.ImageDetails // 镜像详细信息

	// 标签页状态
	activeTab ImageDetailsTab

	// 滚动状态（用于长内容）
	scrollOffset int
	maxScroll    int

	// 加载状态
	loading  bool
	errorMsg string
}

// NewImageDetailsView 创建镜像详情视图
func NewImageDetailsView(dockerClient docker.Client, image *docker.Image) *ImageDetailsView {
	return &ImageDetailsView{
		dockerClient: dockerClient,
		image:        image,
		activeTab:    TabBasicInfo,
		scrollOffset: 0,
	}
}

// Init 初始化视图
func (v *ImageDetailsView) Init() tea.Cmd {
	v.loading = true
	return v.loadImageDetails
}

// Update 处理消息
func (v *ImageDetailsView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case imageDetailsLoadedMsg:
		v.details = msg.details
		v.loading = false
		v.errorMsg = ""
		return v, nil

	case imageDetailsLoadErrorMsg:
		v.loading = false
		v.errorMsg = msg.err.Error()
		return v, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// ESC 返回上一级
			return v, func() tea.Msg { return GoBackMsg{} }
		case "tab", "l", "right":
			// 切换到下一个标签页
			v.activeTab = (v.activeTab + 1) % ImageDetailsTab(len(imageTabNames))
			v.scrollOffset = 0
			return v, nil
		case "shift+tab", "h", "left":
			// 切换到上一个标签页
			if v.activeTab == 0 {
				v.activeTab = ImageDetailsTab(len(imageTabNames) - 1)
			} else {
				v.activeTab--
			}
			v.scrollOffset = 0
			return v, nil
		case "j", "down":
			// 向下滚动
			if v.scrollOffset < v.maxScroll {
				v.scrollOffset++
			}
			return v, nil
		case "k", "up":
			// 向上滚动
			if v.scrollOffset > 0 {
				v.scrollOffset--
			}
			return v, nil
		case "g":
			// 跳转到顶部
			v.scrollOffset = 0
			return v, nil
		case "G":
			// 跳转到底部
			v.scrollOffset = v.maxScroll
			return v, nil
		case "1":
			v.activeTab = TabBasicInfo
			v.scrollOffset = 0
			return v, nil
		case "2":
			v.activeTab = TabUsage
			v.scrollOffset = 0
			return v, nil
		case "3":
			v.activeTab = TabConfig
			v.scrollOffset = 0
			return v, nil
		case "4":
			v.activeTab = TabEnvVars
			v.scrollOffset = 0
			return v, nil
		case "5":
			v.activeTab = TabHistory
			v.scrollOffset = 0
			return v, nil
		case "6":
			v.activeTab = TabLabels
			v.scrollOffset = 0
			return v, nil
		}
	}

	return v, nil
}

// View 渲染视图
func (v *ImageDetailsView) View() string {
	var s strings.Builder

	// 标题
	title := "🖼️  Image Details"
	if v.image != nil {
		imageName := v.image.Repository + ":" + v.image.Tag
		if len(imageName) > 40 {
			imageName = imageName[:37] + "..."
		}
		title = "🖼️  " + imageName
	}
	s.WriteString("\n  " + imageDetailsTitleStyle.Render(title) + "\n\n")

	// 标签页导航
	s.WriteString(v.renderTabs())
	s.WriteString("\n")

	// 加载中状态
	if v.loading {
		s.WriteString("\n  " + imageDetailsHintStyle.Render("⏳ 正在加载镜像详情...") + "\n")
		return s.String()
	}

	// 错误状态
	if v.errorMsg != "" {
		s.WriteString("\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("❌ "+v.errorMsg) + "\n")
		return s.String()
	}

	// 渲染当前标签页内容
	s.WriteString(v.renderCurrentTab())

	// 底部快捷键提示
	s.WriteString("\n" + v.renderHints())

	return s.String()
}

// SetSize 设置视图尺寸
func (v *ImageDetailsView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// renderTabs 渲染标签页导航
func (v *ImageDetailsView) renderTabs() string {
	var tabs []string

	for i, name := range imageTabNames {
		tabNum := fmt.Sprintf("[%d]", i+1)
		if ImageDetailsTab(i) == v.activeTab {
			tabs = append(tabs, imageTabActiveStyle.Render(tabNum+" "+name))
		} else {
			tabs = append(tabs, imageTabInactiveStyle.Render(tabNum+" "+name))
		}
	}

	return "  " + strings.Join(tabs, "  │  ")
}

// renderCurrentTab 渲染当前标签页内容
func (v *ImageDetailsView) renderCurrentTab() string {
	switch v.activeTab {
	case TabBasicInfo:
		return v.renderBasicInfo()
	case TabUsage:
		return v.renderUsage()
	case TabConfig:
		return v.renderConfig()
	case TabEnvVars:
		return v.renderEnvVars()
	case TabHistory:
		return v.renderHistory()
	case TabLabels:
		return v.renderLabels()
	default:
		return ""
	}
}

// renderBasicInfo 渲染基本信息
func (v *ImageDetailsView) renderBasicInfo() string {
	if v.image == nil {
		return "\n  " + imageDetailsHintStyle.Render("无镜像信息")
	}

	var lines []string

	// 使用详情信息（如果有）
	if v.details != nil {
		lines = append(lines, v.formatLine("IMAGE ID", v.details.ID))
		lines = append(lines, v.formatLine("REPOSITORY", v.details.Repository))
		lines = append(lines, v.formatLine("TAG", v.details.Tag))
		if v.details.Digest != "" {
			digest := v.details.Digest
			if len(digest) > 50 {
				digest = digest[:47] + "..."
			}
			lines = append(lines, v.formatLine("DIGEST", digest))
		}
		lines = append(lines, v.formatLine("SIZE", formatSize(v.details.Size)))
		lines = append(lines, v.formatLine("CREATED", v.details.Created.Format("2006-01-02 15:04:05")+" ("+formatCreatedTime(v.details.Created)+")"))
		lines = append(lines, v.formatLine("ARCHITECTURE", v.details.Architecture))
		lines = append(lines, v.formatLine("OS", v.details.OS))
		if v.details.Author != "" {
			lines = append(lines, v.formatLine("AUTHOR", v.details.Author))
		}
	} else {
		// 使用基本信息
		lines = append(lines, v.formatLine("IMAGE ID", v.image.ID))
		lines = append(lines, v.formatLine("REPOSITORY", v.image.Repository))
		lines = append(lines, v.formatLine("TAG", v.image.Tag))
		lines = append(lines, v.formatLine("SIZE", formatSize(v.image.Size)))
		lines = append(lines, v.formatLine("CREATED", v.image.Created.Format("2006-01-02 15:04:05")))
	}

	content := strings.Join(lines, "\n")
	boxWidth := v.width - 6
	if boxWidth < 60 {
		boxWidth = 60
	}

	return "\n" + v.wrapInBox("Basic Information", content, boxWidth)
}

// renderUsage 渲染使用状态
func (v *ImageDetailsView) renderUsage() string {
	var lines []string

	// 使用状态
	if v.image != nil {
		if v.image.InUse {
			lines = append(lines, v.formatLine("STATUS", "🟢 In Use"))
		} else if v.image.Dangling {
			lines = append(lines, v.formatLine("STATUS", "🟡 Dangling (无标签)"))
		} else {
			lines = append(lines, v.formatLine("STATUS", "🔴 Unused"))
		}
	}

	// 使用此镜像的容器
	if v.details != nil && len(v.details.Containers) > 0 {
		lines = append(lines, "")
		lines = append(lines, imageDetailsLabelStyle.Render("CONTAINERS:")+" ("+fmt.Sprintf("%d", len(v.details.Containers))+")")
		for i, containerRef := range v.details.Containers {
			if i >= 10 {
				lines = append(lines, "  "+imageDetailsHintStyle.Render(fmt.Sprintf("... and %d more", len(v.details.Containers)-10)))
				break
			}
			shortID := containerRef.ID
			if len(shortID) > 12 {
				shortID = shortID[:12]
			}

			// 根据状态设置样式
			var stateStyle lipgloss.Style
			var stateIcon string
			switch containerRef.State {
			case "running":
				stateStyle = imageContainerRunningStyle
				stateIcon = "🟢"
			case "exited":
				stateStyle = imageContainerStoppedStyle
				stateIcon = "🔴"
			case "paused":
				stateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
				stateIcon = "🟡"
			default:
				stateStyle = imageDetailsHintStyle
				stateIcon = "⚪"
			}

			// 格式化显示：ID (名称) [状态]
			containerInfo := fmt.Sprintf("%s (%s) %s %s",
				imageDetailsKeyStyle.Render(shortID),
				imageDetailsValueStyle.Render(containerRef.Name),
				stateIcon,
				stateStyle.Render(containerRef.State))

			lines = append(lines, "  • "+containerInfo)
		}
	} else if v.image != nil && len(v.image.Containers) > 0 {
		lines = append(lines, "")
		lines = append(lines, imageDetailsLabelStyle.Render("CONTAINERS:")+" ("+fmt.Sprintf("%d", len(v.image.Containers))+")")
		for i, containerID := range v.image.Containers {
			if i >= 10 {
				lines = append(lines, "  "+imageDetailsHintStyle.Render(fmt.Sprintf("... and %d more", len(v.image.Containers)-10)))
				break
			}
			shortID := containerID
			if len(shortID) > 12 {
				shortID = shortID[:12]
			}
			lines = append(lines, "  • "+shortID)
		}
	} else {
		lines = append(lines, "")
		lines = append(lines, imageDetailsHintStyle.Render("没有容器使用此镜像"))
	}

	content := strings.Join(lines, "\n")
	boxWidth := v.width - 6
	if boxWidth < 60 {
		boxWidth = 60
	}

	return "\n" + v.wrapInBox("Usage Status", content, boxWidth)
}

// renderConfig 渲染配置信息
func (v *ImageDetailsView) renderConfig() string {
	if v.details == nil {
		return "\n  " + imageDetailsHintStyle.Render("无配置信息")
	}

	var lines []string

	// 入口点
	if len(v.details.Entrypoint) > 0 {
		lines = append(lines, v.formatLine("ENTRYPOINT", strings.Join(v.details.Entrypoint, " ")))
	} else {
		lines = append(lines, v.formatLine("ENTRYPOINT", "(none)"))
	}

	// 命令
	if len(v.details.Cmd) > 0 {
		lines = append(lines, v.formatLine("CMD", strings.Join(v.details.Cmd, " ")))
	} else {
		lines = append(lines, v.formatLine("CMD", "(none)"))
	}

	// 工作目录
	if v.details.WorkingDir != "" {
		lines = append(lines, v.formatLine("WORKING DIR", v.details.WorkingDir))
	} else {
		lines = append(lines, v.formatLine("WORKING DIR", "/"))
	}

	// 用户
	if v.details.User != "" {
		lines = append(lines, v.formatLine("USER", v.details.User))
	} else {
		lines = append(lines, v.formatLine("USER", "root"))
	}

	// 暴露端口
	if len(v.details.ExposedPorts) > 0 {
		lines = append(lines, v.formatLine("EXPOSED PORTS", strings.Join(v.details.ExposedPorts, ", ")))
	}

	// 卷
	if len(v.details.Volumes) > 0 {
		lines = append(lines, v.formatLine("VOLUMES", strings.Join(v.details.Volumes, ", ")))
	}

	content := strings.Join(lines, "\n")
	boxWidth := v.width - 6
	if boxWidth < 60 {
		boxWidth = 60
	}

	return "\n" + v.wrapInBox("Configuration", content, boxWidth)
}

// renderEnvVars 渲染环境变量
func (v *ImageDetailsView) renderEnvVars() string {
	if v.details == nil || len(v.details.Env) == 0 {
		return "\n  " + imageDetailsHintStyle.Render("无环境变量")
	}

	var lines []string
	envCount := len(v.details.Env)

	// 计算可显示的行数
	maxLines := v.height - 15
	if maxLines < 5 {
		maxLines = 5
	}

	// 应用滚动
	startIdx := v.scrollOffset
	endIdx := startIdx + maxLines
	if endIdx > envCount {
		endIdx = envCount
	}
	v.maxScroll = envCount - maxLines
	if v.maxScroll < 0 {
		v.maxScroll = 0
	}

	for i := startIdx; i < endIdx; i++ {
		env := v.details.Env[i]
		// 截断过长的环境变量
		if len(env) > v.width-10 {
			env = env[:v.width-13] + "..."
		}
		lines = append(lines, "  "+env)
	}

	// 滚动提示
	if v.maxScroll > 0 {
		scrollInfo := fmt.Sprintf("(%d/%d) ", v.scrollOffset+1, envCount)
		if v.scrollOffset > 0 {
			scrollInfo += "↑ "
		}
		if v.scrollOffset < v.maxScroll {
			scrollInfo += "↓"
		}
		lines = append(lines, "")
		lines = append(lines, imageDetailsHintStyle.Render(scrollInfo+"  j/k 滚动"))
	}

	content := strings.Join(lines, "\n")
	boxWidth := v.width - 6
	if boxWidth < 60 {
		boxWidth = 60
	}

	title := fmt.Sprintf("Environment Variables (%d)", envCount)
	return "\n" + v.wrapInBox(title, content, boxWidth)
}

// renderHistory 渲染构建历史（类似 docker history）
func (v *ImageDetailsView) renderHistory() string {
	if v.details == nil || len(v.details.History) == 0 {
		return "\n  " + imageDetailsHintStyle.Render("无构建历史信息")
	}

	var lines []string
	historyCount := len(v.details.History)

	// 计算可显示的行数（每条历史记录占 2-3 行）
	maxItems := (v.height - 15) / 3
	if maxItems < 3 {
		maxItems = 3
	}

	// 应用滚动
	startIdx := v.scrollOffset
	endIdx := startIdx + maxItems
	if endIdx > historyCount {
		endIdx = historyCount
	}
	v.maxScroll = historyCount - maxItems
	if v.maxScroll < 0 {
		v.maxScroll = 0
	}

	for i := startIdx; i < endIdx; i++ {
		h := v.details.History[i]

		// 格式化 ID
		idStr := h.ID
		if len(idStr) > 12 && idStr != "<missing>" {
			if strings.HasPrefix(idStr, "sha256:") {
				idStr = idStr[7:19]
			} else {
				idStr = idStr[:12]
			}
		}

		// 格式化创建时间
		createdStr := formatCreatedTime(h.Created)

		// 格式化命令（截断过长的命令）
		cmdStr := h.CreatedBy
		// 移除 /bin/sh -c 前缀
		cmdStr = strings.TrimPrefix(cmdStr, "/bin/sh -c ")
		cmdStr = strings.TrimPrefix(cmdStr, "#(nop) ")
		// 截断过长的命令
		maxCmdLen := v.width - 20
		if maxCmdLen < 40 {
			maxCmdLen = 40
		}
		if len(cmdStr) > maxCmdLen {
			cmdStr = cmdStr[:maxCmdLen-3] + "..."
		}

		// 格式化大小
		sizeStr := ""
		if h.Size > 0 {
			sizeStr = formatSize(h.Size)
		} else {
			sizeStr = "0B"
		}

		// 构建显示行
		// 第一行：ID + 创建时间 + 大小
		line1 := fmt.Sprintf("  %s  %s  %s",
			imageDetailsKeyStyle.Render(idStr),
			imageDetailsHintStyle.Render(createdStr),
			imageDetailsValueStyle.Render(sizeStr))

		// 第二行：命令
		line2 := "    " + imageDetailsValueStyle.Render(cmdStr)

		lines = append(lines, line1)
		lines = append(lines, line2)

		// 添加分隔线（除了最后一条）
		if i < endIdx-1 {
			lines = append(lines, "")
		}
	}

	// 滚动提示
	if v.maxScroll > 0 {
		scrollInfo := fmt.Sprintf("(%d/%d) ", v.scrollOffset+1, historyCount)
		if v.scrollOffset > 0 {
			scrollInfo += "↑ "
		}
		if v.scrollOffset < v.maxScroll {
			scrollInfo += "↓"
		}
		lines = append(lines, "")
		lines = append(lines, imageDetailsHintStyle.Render(scrollInfo+"  j/k 滚动"))
	}

	content := strings.Join(lines, "\n")
	boxWidth := v.width - 6
	if boxWidth < 60 {
		boxWidth = 60
	}

	title := fmt.Sprintf("Build History (%d)", historyCount)
	return "\n" + v.wrapInBox(title, content, boxWidth)
}

// renderLabels 渲染标签信息
func (v *ImageDetailsView) renderLabels() string {
	if v.details == nil || len(v.details.Labels) == 0 {
		return "\n  " + imageDetailsHintStyle.Render("无标签信息")
	}

	var lines []string

	// 将 map 转换为切片以便排序和滚动
	var labelPairs []string
	for k, val := range v.details.Labels {
		labelPairs = append(labelPairs, k+"="+val)
	}

	labelCount := len(labelPairs)

	// 计算可显示的行数
	maxLines := v.height - 15
	if maxLines < 5 {
		maxLines = 5
	}

	// 应用滚动
	startIdx := v.scrollOffset
	endIdx := startIdx + maxLines
	if endIdx > labelCount {
		endIdx = labelCount
	}
	v.maxScroll = labelCount - maxLines
	if v.maxScroll < 0 {
		v.maxScroll = 0
	}

	for i := startIdx; i < endIdx; i++ {
		label := labelPairs[i]
		// 截断过长的标签
		if len(label) > v.width-10 {
			label = label[:v.width-13] + "..."
		}
		lines = append(lines, "  "+label)
	}

	// 滚动提示
	if v.maxScroll > 0 {
		scrollInfo := fmt.Sprintf("(%d/%d) ", v.scrollOffset+1, labelCount)
		if v.scrollOffset > 0 {
			scrollInfo += "↑ "
		}
		if v.scrollOffset < v.maxScroll {
			scrollInfo += "↓"
		}
		lines = append(lines, "")
		lines = append(lines, imageDetailsHintStyle.Render(scrollInfo+"  j/k 滚动"))
	}

	content := strings.Join(lines, "\n")
	boxWidth := v.width - 6
	if boxWidth < 60 {
		boxWidth = 60
	}

	title := fmt.Sprintf("Labels (%d)", labelCount)
	return "\n" + v.wrapInBox(title, content, boxWidth)
}

// renderHints 渲染底部快捷键提示
func (v *ImageDetailsView) renderHints() string {
	hints := []string{
		imageDetailsKeyStyle.Render("<Tab/←/→>") + " 切换标签",
		imageDetailsKeyStyle.Render("<1-6>") + " 快速跳转",
		imageDetailsKeyStyle.Render("<j/k>") + " 滚动",
		imageDetailsKeyStyle.Render("<Esc>") + " 返回",
	}

	return "  " + imageDetailsHintStyle.Render(strings.Join(hints, "  │  "))
}

// formatLine 格式化一行信息
func (v *ImageDetailsView) formatLine(label, value string) string {
	return imageDetailsLabelStyle.Render(label+":") + " " + imageDetailsValueStyle.Render(value)
}

// wrapInBox 将内容包装在边框中
func (v *ImageDetailsView) wrapInBox(title, content string, width int) string {
	boxStyle := imageDetailsBoxStyle.Width(width)
	titleLine := "  " + imageDetailsTitleStyle.Render("─ "+title+" ") + imageDetailsHintStyle.Render(strings.Repeat("─", width-len(title)-6))
	return titleLine + "\n" + boxStyle.Render(content)
}

// loadImageDetails 加载镜像详情
func (v *ImageDetailsView) loadImageDetails() tea.Msg {
	if v.image == nil {
		return imageDetailsLoadErrorMsg{err: fmt.Errorf("镜像信息为空")}
	}

	ctx := context.Background()
	details, err := v.dockerClient.ImageDetails(ctx, v.image.ID)
	if err != nil {
		return imageDetailsLoadErrorMsg{err: err}
	}

	return imageDetailsLoadedMsg{details: details}
}

// 消息类型
type imageDetailsLoadedMsg struct {
	details *docker.ImageDetails
}

type imageDetailsLoadErrorMsg struct {
	err error
}
