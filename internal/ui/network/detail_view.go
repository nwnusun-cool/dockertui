package network

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"docktui/internal/docker"
)

// DetailTab 网络详情标签页类型
type DetailTab int

const (
	TabBasicInfo DetailTab = iota
	TabIPAM
	TabContainers
	TabLabels
)

var tabNames = []string{"Basic Info", "IPAM Config", "Containers", "Labels"}

// DetailView 网络详情视图
type DetailView struct {
	dockerClient docker.Client
	width, height int
	network *docker.Network
	details *docker.NetworkDetails
	activeTab DetailTab
	scrollOffset, maxScroll int
	loading bool
	errorMsg string
}

// NewDetailView 创建网络详情视图
func NewDetailView(dockerClient docker.Client, network *docker.Network) *DetailView {
	return &DetailView{dockerClient: dockerClient, network: network, activeTab: TabBasicInfo}
}

// Init 初始化视图
func (v *DetailView) Init() tea.Cmd {
	v.loading = true
	return v.loadNetworkDetails
}

// Update 处理消息
func (v *DetailView) Update(msg tea.Msg) (*DetailView, tea.Cmd) {
	switch msg := msg.(type) {
	case NetworkDetailLoadedMsg:
		v.details = msg.Details
		v.loading = false
		v.errorMsg = ""
		return v, nil
	case NetworkDetailLoadErrorMsg:
		v.loading = false
		v.errorMsg = msg.Err.Error()
		return v, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc": return v, func() tea.Msg { return GoBackMsg{} }
		case "tab", "l", "right":
			v.activeTab = (v.activeTab + 1) % DetailTab(len(tabNames))
			v.scrollOffset = 0
		case "shift+tab", "h", "left":
			if v.activeTab == 0 { v.activeTab = DetailTab(len(tabNames) - 1) } else { v.activeTab-- }
			v.scrollOffset = 0
		case "j", "down": if v.scrollOffset < v.maxScroll { v.scrollOffset++ }
		case "k", "up": if v.scrollOffset > 0 { v.scrollOffset-- }
		case "g": v.scrollOffset = 0
		case "G": v.scrollOffset = v.maxScroll
		case "1": v.activeTab = TabBasicInfo; v.scrollOffset = 0
		case "2": v.activeTab = TabIPAM; v.scrollOffset = 0
		case "3": v.activeTab = TabContainers; v.scrollOffset = 0
		case "4": v.activeTab = TabLabels; v.scrollOffset = 0
		case "r": v.loading = true; return v, v.loadNetworkDetails
		}
	}
	return v, nil
}

// View 渲染视图
func (v *DetailView) View() string {
	var s strings.Builder
	title := "🌐 Network Details"
	if v.network != nil {
		name := v.network.Name
		if len(name) > 40 { name = name[:37] + "..." }
		title = "🌐 " + name
	}
	s.WriteString("\n  " + DetailTitleStyle.Render(title) + "\n\n")
	s.WriteString(v.renderTabs() + "\n")
	if v.loading { s.WriteString("\n  " + DetailHintStyle.Render("⏳ 正在加载网络详情...") + "\n"); return s.String() }
	if v.errorMsg != "" { s.WriteString("\n  " + FormErrorStyle.Render("❌ "+v.errorMsg) + "\n"); return s.String() }
	s.WriteString(v.renderCurrentTab())
	s.WriteString("\n" + v.renderHints())
	return s.String()
}

// SetSize 设置视图尺寸
func (v *DetailView) SetSize(width, height int) { v.width = width; v.height = height }

func (v *DetailView) renderTabs() string {
	var tabs []string
	for i, name := range tabNames {
		tabNum := fmt.Sprintf("[%d]", i+1)
		if DetailTab(i) == v.activeTab {
			tabs = append(tabs, TabActiveStyle.Render(tabNum+" "+name))
		} else {
			tabs = append(tabs, TabInactiveStyle.Render(tabNum+" "+name))
		}
	}
	return "  " + strings.Join(tabs, "  │  ")
}

func (v *DetailView) renderCurrentTab() string {
	switch v.activeTab {
	case TabBasicInfo: return v.renderBasicInfo()
	case TabIPAM: return v.renderIPAMConfig()
	case TabContainers: return v.renderContainers()
	case TabLabels: return v.renderLabels()
	default: return ""
	}
}

func (v *DetailView) renderBasicInfo() string {
	if v.details == nil { return "\n  " + DetailHintStyle.Render("无网络信息") }
	var lines []string
	lines = append(lines, v.formatLine("NETWORK ID", v.details.ID))
	lines = append(lines, v.formatLine("NAME", v.details.Name))
	lines = append(lines, v.formatLine("DRIVER", v.details.Driver))
	lines = append(lines, v.formatLine("SCOPE", v.details.Scope))
	lines = append(lines, v.formatLine("CREATED", v.details.Created.Format("2006-01-02 15:04:05")+" ("+formatCreatedTime(v.details.Created)+")"))
	internalStr := "No"; if v.details.Internal { internalStr = "Yes (不能访问外部网络)" }
	lines = append(lines, v.formatLine("INTERNAL", internalStr))
	ipv6Str := "No"; if v.details.IPv6 { ipv6Str = "Yes" }
	lines = append(lines, v.formatLine("IPv6", ipv6Str))
	attachableStr := "No"; if v.details.Attachable { attachableStr = "Yes (可手动连接容器)" }
	lines = append(lines, v.formatLine("ATTACHABLE", attachableStr))
	ingressStr := "No"; if v.details.Ingress { ingressStr = "Yes" }
	lines = append(lines, v.formatLine("INGRESS", ingressStr))
	if len(v.details.Options) > 0 {
		lines = append(lines, "", DetailLabelStyle.Render("DRIVER OPTIONS:"))
		for k, val := range v.details.Options {
			lines = append(lines, "  "+DetailKeyStyle.Render(k)+" = "+DetailValueStyle.Render(val))
		}
	}
	boxWidth := v.width - 6; if boxWidth < 60 { boxWidth = 60 }
	return "\n" + v.wrapInBox("Basic Information", strings.Join(lines, "\n"), boxWidth)
}

func (v *DetailView) renderIPAMConfig() string {
	if v.details == nil { return "\n  " + DetailHintStyle.Render("无 IPAM 配置信息") }
	var lines []string
	driver := v.details.IPAM.Driver; if driver == "" { driver = "default" }
	lines = append(lines, v.formatLine("IPAM DRIVER", driver))
	if len(v.details.IPAM.Options) > 0 {
		lines = append(lines, "", DetailLabelStyle.Render("IPAM OPTIONS:"))
		for k, val := range v.details.IPAM.Options {
			lines = append(lines, "  "+DetailKeyStyle.Render(k)+" = "+DetailValueStyle.Render(val))
		}
	}
	if len(v.details.IPAM.Configs) > 0 {
		lines = append(lines, "", DetailLabelStyle.Render("IP POOLS:")+" ("+fmt.Sprintf("%d", len(v.details.IPAM.Configs))+")")
		for i, cfg := range v.details.IPAM.Configs {
			lines = append(lines, "", DetailKeyStyle.Render(fmt.Sprintf("  Pool #%d:", i+1)))
			if cfg.Subnet != "" { lines = append(lines, "    "+DetailLabelStyle.Render("Subnet:")+" "+DetailValueStyle.Render(cfg.Subnet)) }
			if cfg.Gateway != "" { lines = append(lines, "    "+DetailLabelStyle.Render("Gateway:")+" "+DetailValueStyle.Render(cfg.Gateway)) }
			if cfg.IPRange != "" { lines = append(lines, "    "+DetailLabelStyle.Render("IP Range:")+" "+DetailValueStyle.Render(cfg.IPRange)) }
		}
	} else {
		lines = append(lines, "", DetailHintStyle.Render("无 IP 池配置（使用默认配置）"))
	}
	boxWidth := v.width - 6; if boxWidth < 60 { boxWidth = 60 }
	return "\n" + v.wrapInBox("IPAM Configuration", strings.Join(lines, "\n"), boxWidth)
}

func (v *DetailView) renderContainers() string {
	if v.details == nil || len(v.details.Containers) == 0 {
		return "\n  " + DetailHintStyle.Render("没有容器连接到此网络")
	}
	var lines []string
	containerCount := len(v.details.Containers)
	maxItems := (v.height - 15) / 5; if maxItems < 2 { maxItems = 2 }
	startIdx := v.scrollOffset
	endIdx := startIdx + maxItems; if endIdx > containerCount { endIdx = containerCount }
	v.maxScroll = containerCount - maxItems; if v.maxScroll < 0 { v.maxScroll = 0 }
	for i := startIdx; i < endIdx; i++ {
		c := v.details.Containers[i]
		shortID := c.ContainerID; if len(shortID) > 12 { shortID = shortID[:12] }
		name := c.ContainerName; if name == "" { name = shortID }
		lines = append(lines, "", DetailKeyStyle.Render(fmt.Sprintf("  📦 %s", name)))
		lines = append(lines, "    "+DetailLabelStyle.Render("Container ID:")+" "+DetailValueStyle.Render(shortID))
		if c.IPv4Address != "" { lines = append(lines, "    "+DetailLabelStyle.Render("IPv4:")+" "+DetailValueStyle.Render(c.IPv4Address)) }
		if c.IPv6Address != "" { lines = append(lines, "    "+DetailLabelStyle.Render("IPv6:")+" "+DetailValueStyle.Render(c.IPv6Address)) }
		if c.MacAddress != "" { lines = append(lines, "    "+DetailLabelStyle.Render("MAC:")+" "+DetailValueStyle.Render(c.MacAddress)) }
	}
	if v.maxScroll > 0 {
		scrollInfo := fmt.Sprintf("(%d/%d) ", v.scrollOffset+1, containerCount)
		if v.scrollOffset > 0 { scrollInfo += "↑ " }
		if v.scrollOffset < v.maxScroll { scrollInfo += "↓" }
		lines = append(lines, "", DetailHintStyle.Render(scrollInfo+"  j/k 滚动"))
	}
	boxWidth := v.width - 6; if boxWidth < 60 { boxWidth = 60 }
	return "\n" + v.wrapInBox(fmt.Sprintf("Connected Containers (%d)", containerCount), strings.Join(lines, "\n"), boxWidth)
}

func (v *DetailView) renderLabels() string {
	if v.details == nil || len(v.details.Labels) == 0 { return "\n  " + DetailHintStyle.Render("无标签信息") }
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
		lines = append(lines, "", DetailHintStyle.Render(scrollInfo+"  j/k 滚动"))
	}
	boxWidth := v.width - 6; if boxWidth < 60 { boxWidth = 60 }
	return "\n" + v.wrapInBox(fmt.Sprintf("Labels (%d)", labelCount), strings.Join(lines, "\n"), boxWidth)
}

func (v *DetailView) renderHints() string {
	hints := []string{
		DetailKeyStyle.Render("<Tab/←/→>") + " 切换标签",
		DetailKeyStyle.Render("<1-4>") + " 快速跳转",
		DetailKeyStyle.Render("<j/k>") + " 滚动",
		DetailKeyStyle.Render("<r>") + " 刷新",
		DetailKeyStyle.Render("<Esc>") + " 返回",
	}
	return "  " + DetailHintStyle.Render(strings.Join(hints, "  │  "))
}

func (v *DetailView) formatLine(label, value string) string {
	return DetailLabelStyle.Render(label+":") + " " + DetailValueStyle.Render(value)
}

func (v *DetailView) wrapInBox(title, content string, width int) string {
	boxStyle := DetailBoxStyle.Width(width)
	titleLine := "  " + DetailTitleStyle.Render("─ "+title+" ") + DetailHintStyle.Render(strings.Repeat("─", width-len(title)-6))
	return titleLine + "\n" + boxStyle.Render(content)
}

func (v *DetailView) loadNetworkDetails() tea.Msg {
	if v.network == nil { return NetworkDetailLoadErrorMsg{Err: fmt.Errorf("网络信息为空")} }
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	details, err := v.dockerClient.NetworkDetails(ctx, v.network.ID)
	if err != nil { return NetworkDetailLoadErrorMsg{Err: err} }
	return NetworkDetailLoadedMsg{Details: details}
}

// formatCreatedTime 格式化创建时间
func formatCreatedTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute: return "刚刚"
	case d < time.Hour: return fmt.Sprintf("%d 分钟前", int(d.Minutes()))
	case d < 24*time.Hour: return fmt.Sprintf("%d 小时前", int(d.Hours()))
	case d < 30*24*time.Hour: return fmt.Sprintf("%d 天前", int(d.Hours()/24))
	case d < 365*24*time.Hour: return fmt.Sprintf("%d 个月前", int(d.Hours()/24/30))
	default: return fmt.Sprintf("%d 年前", int(d.Hours()/24/365))
	}
}
