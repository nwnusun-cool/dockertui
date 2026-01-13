package image

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"docktui/internal/docker"
	"docktui/internal/ui/components"
)

// DetailsTab 镜像详情标签页类型
type DetailsTab int

const (
	TabBasicInfo DetailsTab = iota
	TabUsage
	TabConfig
	TabEnvVars
	TabHistory
	TabLabels
)

var tabNames = []string{"Basic Info", "Usage", "Config", "Env Vars", "History", "Labels"}

// DetailsView 镜像详情视图
type DetailsView struct {
	dockerClient docker.Client
	width, height int
	image *docker.Image
	details *docker.ImageDetails
	activeTab DetailsTab
	scrollOffset, maxScroll int
	loading bool
	errorMsg string
}

// NewDetailsView 创建镜像详情视图
func NewDetailsView(dockerClient docker.Client, image *docker.Image) *DetailsView {
	return &DetailsView{dockerClient: dockerClient, image: image, activeTab: TabBasicInfo}
}

// Init 初始化视图
func (v *DetailsView) Init() tea.Cmd {
	v.loading = true
	return v.loadImageDetails
}

// Update 处理消息
func (v *DetailsView) Update(msg tea.Msg) (*DetailsView, tea.Cmd) {
	switch msg := msg.(type) {
	case ImageDetailsLoadedMsg:
		v.details = msg.Details
		v.loading = false
		v.errorMsg = ""
		return v, nil
	case ImageDetailsLoadErrorMsg:
		v.loading = false
		v.errorMsg = msg.Err.Error()
		return v, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc": return v, func() tea.Msg { return GoBackMsg{} }
		case "tab", "l", "right":
			v.activeTab = (v.activeTab + 1) % DetailsTab(len(tabNames))
			v.scrollOffset = 0
		case "shift+tab", "h", "left":
			if v.activeTab == 0 { v.activeTab = DetailsTab(len(tabNames) - 1) } else { v.activeTab-- }
			v.scrollOffset = 0
		case "j", "down": if v.scrollOffset < v.maxScroll { v.scrollOffset++ }
		case "k", "up": if v.scrollOffset > 0 { v.scrollOffset-- }
		case "g": v.scrollOffset = 0
		case "G": v.scrollOffset = v.maxScroll
		case "1": v.activeTab = TabBasicInfo; v.scrollOffset = 0
		case "2": v.activeTab = TabUsage; v.scrollOffset = 0
		case "3": v.activeTab = TabConfig; v.scrollOffset = 0
		case "4": v.activeTab = TabEnvVars; v.scrollOffset = 0
		case "5": v.activeTab = TabHistory; v.scrollOffset = 0
		case "6": v.activeTab = TabLabels; v.scrollOffset = 0
		}
	}
	return v, nil
}

// View 渲染视图
func (v *DetailsView) View() string {
	var s strings.Builder
	title := "🖼️  Image Details"
	if v.image != nil {
		imageName := v.image.Repository + ":" + v.image.Tag
		if len(imageName) > 40 { imageName = imageName[:37] + "..." }
		title = "🖼️  " + imageName
	}
	s.WriteString("\n  " + DetailsTitleStyle.Render(title) + "\n\n")
	s.WriteString(v.renderTabs() + "\n")
	if v.loading { s.WriteString("\n  " + DetailsHintStyle.Render("⏳ Loading image details...") + "\n"); return s.String() }
	if v.errorMsg != "" { s.WriteString("\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("❌ "+v.errorMsg) + "\n"); return s.String() }
	s.WriteString(v.renderCurrentTab())
	s.WriteString("\n" + v.renderHints())
	return s.String()
}

// SetSize 设置视图尺寸
func (v *DetailsView) SetSize(width, height int) { v.width = width; v.height = height }

func (v *DetailsView) renderTabs() string {
	var tabs []string
	for i, name := range tabNames {
		tabNum := fmt.Sprintf("[%d]", i+1)
		if DetailsTab(i) == v.activeTab {
			tabs = append(tabs, TabActiveStyle.Render(tabNum+" "+name))
		} else {
			tabs = append(tabs, TabInactiveStyle.Render(tabNum+" "+name))
		}
	}
	return "  " + strings.Join(tabs, "  │  ")
}

func (v *DetailsView) renderCurrentTab() string {
	switch v.activeTab {
	case TabBasicInfo: return v.renderBasicInfo()
	case TabUsage: return v.renderUsage()
	case TabConfig: return v.renderConfig()
	case TabEnvVars: return v.renderEnvVars()
	case TabHistory: return v.renderHistory()
	case TabLabels: return v.renderLabels()
	default: return ""
	}
}

func (v *DetailsView) renderBasicInfo() string {
	if v.image == nil { return "\n  " + DetailsHintStyle.Render("No image info") }
	var lines []string
	if v.details != nil {
		lines = append(lines, v.formatLine("IMAGE ID", v.details.ID))
		lines = append(lines, v.formatLine("REPOSITORY", v.details.Repository))
		lines = append(lines, v.formatLine("TAG", v.details.Tag))
		if v.details.Digest != "" {
			digest := v.details.Digest; if len(digest) > 50 { digest = digest[:47] + "..." }
			lines = append(lines, v.formatLine("DIGEST", digest))
		}
		lines = append(lines, v.formatLine("SIZE", FormatSize(v.details.Size)))
		lines = append(lines, v.formatLine("CREATED", v.details.Created.Format("2006-01-02 15:04:05")+" ("+FormatCreatedTime(v.details.Created)+")"))
		lines = append(lines, v.formatLine("ARCHITECTURE", v.details.Architecture))
		lines = append(lines, v.formatLine("OS", v.details.OS))
		if v.details.Author != "" { lines = append(lines, v.formatLine("AUTHOR", v.details.Author)) }
	} else {
		lines = append(lines, v.formatLine("IMAGE ID", v.image.ID))
		lines = append(lines, v.formatLine("REPOSITORY", v.image.Repository))
		lines = append(lines, v.formatLine("TAG", v.image.Tag))
		lines = append(lines, v.formatLine("SIZE", FormatSize(v.image.Size)))
		lines = append(lines, v.formatLine("CREATED", v.image.Created.Format("2006-01-02 15:04:05")))
	}
	boxWidth := v.width - 6; if boxWidth < 60 { boxWidth = 60 }
	return "\n" + v.wrapInBox("Basic Information", strings.Join(lines, "\n"), boxWidth)
}

func (v *DetailsView) renderUsage() string {
	var lines []string
	if v.image != nil {
		if v.image.InUse { lines = append(lines, v.formatLine("STATUS", "🟢 In Use")) } else if v.image.Dangling { lines = append(lines, v.formatLine("STATUS", "🟡 Dangling (no tag)")) } else { lines = append(lines, v.formatLine("STATUS", "🔴 Unused")) }
	}
	if v.details != nil && len(v.details.Containers) > 0 {
		lines = append(lines, "", DetailsLabelStyle.Render("CONTAINERS:")+" ("+fmt.Sprintf("%d", len(v.details.Containers))+")")
		for i, containerRef := range v.details.Containers {
			if i >= 10 { lines = append(lines, "  "+DetailsHintStyle.Render(fmt.Sprintf("... and %d more", len(v.details.Containers)-10))); break }
			shortID := containerRef.ID; if len(shortID) > 12 { shortID = shortID[:12] }
			var stateStyle lipgloss.Style; var stateIcon string
			switch containerRef.State {
			case "running": stateStyle = ContainerRunningStyle; stateIcon = "🟢"
			case "exited": stateStyle = ContainerStoppedStyle; stateIcon = "🔴"
			case "paused": stateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220")); stateIcon = "🟡"
			default: stateStyle = DetailsHintStyle; stateIcon = "⚪"
			}
			containerInfo := fmt.Sprintf("%s (%s) %s %s", DetailsKeyStyle.Render(shortID), DetailsValueStyle.Render(containerRef.Name), stateIcon, stateStyle.Render(containerRef.State))
			lines = append(lines, "  • "+containerInfo)
		}
	} else if v.image != nil && len(v.image.Containers) > 0 {
		lines = append(lines, "", DetailsLabelStyle.Render("CONTAINERS:")+" ("+fmt.Sprintf("%d", len(v.image.Containers))+")")
		for i, containerID := range v.image.Containers {
			if i >= 10 { lines = append(lines, "  "+DetailsHintStyle.Render(fmt.Sprintf("... and %d more", len(v.image.Containers)-10))); break }
			shortID := containerID; if len(shortID) > 12 { shortID = shortID[:12] }
			lines = append(lines, "  • "+shortID)
		}
	} else { lines = append(lines, "", DetailsHintStyle.Render("No containers using this image")) }
	boxWidth := v.width - 6; if boxWidth < 60 { boxWidth = 60 }
	return "\n" + v.wrapInBox("Usage Status", strings.Join(lines, "\n"), boxWidth)
}

func (v *DetailsView) renderConfig() string {
	if v.details == nil { return "\n  " + DetailsHintStyle.Render("No config info") }
	var lines []string
	if len(v.details.Entrypoint) > 0 { lines = append(lines, v.formatLine("ENTRYPOINT", strings.Join(v.details.Entrypoint, " "))) } else { lines = append(lines, v.formatLine("ENTRYPOINT", "(none)")) }
	if len(v.details.Cmd) > 0 { lines = append(lines, v.formatLine("CMD", strings.Join(v.details.Cmd, " "))) } else { lines = append(lines, v.formatLine("CMD", "(none)")) }
	if v.details.WorkingDir != "" { lines = append(lines, v.formatLine("WORKING DIR", v.details.WorkingDir)) } else { lines = append(lines, v.formatLine("WORKING DIR", "/")) }
	if v.details.User != "" { lines = append(lines, v.formatLine("USER", v.details.User)) } else { lines = append(lines, v.formatLine("USER", "root")) }
	if len(v.details.ExposedPorts) > 0 { lines = append(lines, v.formatLine("EXPOSED PORTS", strings.Join(v.details.ExposedPorts, ", "))) }
	if len(v.details.Volumes) > 0 { lines = append(lines, v.formatLine("VOLUMES", strings.Join(v.details.Volumes, ", "))) }
	boxWidth := v.width - 6; if boxWidth < 60 { boxWidth = 60 }
	return "\n" + v.wrapInBox("Configuration", strings.Join(lines, "\n"), boxWidth)
}

func (v *DetailsView) renderEnvVars() string {
	if v.details == nil || len(v.details.Env) == 0 { return "\n  " + DetailsHintStyle.Render("No environment variables") }
	var lines []string
	envCount := len(v.details.Env)
	maxLines := v.height - 15; if maxLines < 5 { maxLines = 5 }
	startIdx := v.scrollOffset
	endIdx := startIdx + maxLines; if endIdx > envCount { endIdx = envCount }
	v.maxScroll = envCount - maxLines; if v.maxScroll < 0 { v.maxScroll = 0 }
	for i := startIdx; i < endIdx; i++ {
		env := v.details.Env[i]
		if len(env) > v.width-10 { env = env[:v.width-13] + "..." }
		lines = append(lines, "  "+env)
	}
	if v.maxScroll > 0 {
		scrollInfo := fmt.Sprintf("(%d/%d) ", v.scrollOffset+1, envCount)
		if v.scrollOffset > 0 { scrollInfo += "↑ " }
		if v.scrollOffset < v.maxScroll { scrollInfo += "↓" }
		lines = append(lines, "", DetailsHintStyle.Render(scrollInfo+"  j/k scroll"))
	}
	boxWidth := v.width - 6; if boxWidth < 60 { boxWidth = 60 }
	return "\n" + v.wrapInBox(fmt.Sprintf("Environment Variables (%d)", envCount), strings.Join(lines, "\n"), boxWidth)
}

func (v *DetailsView) renderHistory() string {
	if v.details == nil || len(v.details.History) == 0 { return "\n  " + DetailsHintStyle.Render("No build history") }
	var lines []string
	historyCount := len(v.details.History)
	maxItems := (v.height - 15) / 3; if maxItems < 3 { maxItems = 3 }
	startIdx := v.scrollOffset
	endIdx := startIdx + maxItems; if endIdx > historyCount { endIdx = historyCount }
	v.maxScroll = historyCount - maxItems; if v.maxScroll < 0 { v.maxScroll = 0 }
	for i := startIdx; i < endIdx; i++ {
		h := v.details.History[i]
		idStr := h.ID
		if len(idStr) > 12 && idStr != "<missing>" {
			if strings.HasPrefix(idStr, "sha256:") { idStr = idStr[7:19] } else { idStr = idStr[:12] }
		}
		createdStr := FormatCreatedTime(h.Created)
		cmdStr := h.CreatedBy
		cmdStr = strings.TrimPrefix(cmdStr, "/bin/sh -c ")
		cmdStr = strings.TrimPrefix(cmdStr, "#(nop) ")
		maxCmdLen := v.width - 20; if maxCmdLen < 40 { maxCmdLen = 40 }
		if len(cmdStr) > maxCmdLen { cmdStr = cmdStr[:maxCmdLen-3] + "..." }
		sizeStr := "0B"; if h.Size > 0 { sizeStr = FormatSize(h.Size) }
		line1 := fmt.Sprintf("  %s  %s  %s", DetailsKeyStyle.Render(idStr), DetailsHintStyle.Render(createdStr), DetailsValueStyle.Render(sizeStr))
		line2 := "    " + DetailsValueStyle.Render(cmdStr)
		lines = append(lines, line1, line2)
		if i < endIdx-1 { lines = append(lines, "") }
	}
	if v.maxScroll > 0 {
		scrollInfo := fmt.Sprintf("(%d/%d) ", v.scrollOffset+1, historyCount)
		if v.scrollOffset > 0 { scrollInfo += "↑ " }
		if v.scrollOffset < v.maxScroll { scrollInfo += "↓" }
		lines = append(lines, "", DetailsHintStyle.Render(scrollInfo+"  j/k scroll"))
	}
	boxWidth := v.width - 6; if boxWidth < 60 { boxWidth = 60 }
	return "\n" + v.wrapInBox(fmt.Sprintf("Build History (%d)", historyCount), strings.Join(lines, "\n"), boxWidth)
}

func (v *DetailsView) renderLabels() string {
	if v.details == nil || len(v.details.Labels) == 0 { return "\n  " + DetailsHintStyle.Render("No labels") }
	var lines []string
	var labelPairs []string
	for k, val := range v.details.Labels { labelPairs = append(labelPairs, k+"="+val) }
	labelCount := len(labelPairs)
	maxLines := v.height - 15; if maxLines < 5 { maxLines = 5 }
	startIdx := v.scrollOffset
	endIdx := startIdx + maxLines; if endIdx > labelCount { endIdx = labelCount }
	v.maxScroll = labelCount - maxLines; if v.maxScroll < 0 { v.maxScroll = 0 }
	for i := startIdx; i < endIdx; i++ {
		label := labelPairs[i]
		if len(label) > v.width-10 { label = label[:v.width-13] + "..." }
		lines = append(lines, "  "+label)
	}
	if v.maxScroll > 0 {
		scrollInfo := fmt.Sprintf("(%d/%d) ", v.scrollOffset+1, labelCount)
		if v.scrollOffset > 0 { scrollInfo += "↑ " }
		if v.scrollOffset < v.maxScroll { scrollInfo += "↓" }
		lines = append(lines, "", DetailsHintStyle.Render(scrollInfo+"  j/k scroll"))
	}
	boxWidth := v.width - 6; if boxWidth < 60 { boxWidth = 60 }
	return "\n" + v.wrapInBox(fmt.Sprintf("Labels (%d)", labelCount), strings.Join(lines, "\n"), boxWidth)
}

func (v *DetailsView) renderHints() string {
	hints := []string{
		DetailsKeyStyle.Render("<Tab/←/→>") + " Switch tabs",
		DetailsKeyStyle.Render("<1-6>") + " Quick jump",
		DetailsKeyStyle.Render("<j/k>") + " Scroll",
		DetailsKeyStyle.Render("<Esc>") + " Back",
	}
	return "  " + DetailsHintStyle.Render(strings.Join(hints, "  │  "))
}

func (v *DetailsView) formatLine(label, value string) string {
	return DetailsLabelStyle.Render(label+":") + " " + DetailsValueStyle.Render(value)
}

func (v *DetailsView) wrapInBox(title, content string, width int) string {
	return components.WrapInBox(title, content, width)
}

func (v *DetailsView) loadImageDetails() tea.Msg {
	if v.image == nil { return ImageDetailsLoadErrorMsg{Err: fmt.Errorf("image info is empty")} }
	ctx := context.Background()
	details, err := v.dockerClient.ImageDetails(ctx, v.image.ID)
	if err != nil { return ImageDetailsLoadErrorMsg{Err: err} }
	return ImageDetailsLoadedMsg{Details: details}
}
