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

// NetworkDetailTab 网络详情标签页类型
type NetworkDetailTab int

const (
	NetworkTabBasicInfo NetworkDetailTab = iota // 基本信息
	NetworkTabIPAM                              // IPAM 配置
	NetworkTabContainers                        // 连接的容器
	NetworkTabLabels                            // 标签信息
)

// 标签页名称
var networkTabNames = []string{
	"Basic Info",
	"IPAM Config",
	"Containers",
	"Labels",
}

// 网络详情视图样式
var (
	networkDetailTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("220")).
				Bold(true)

	networkDetailLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("81")).
				Width(16)

	networkDetailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	networkDetailBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1)

	networkTabActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("220")).
				Bold(true).
				Underline(true)

	networkTabInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

	networkDetailHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

	networkDetailKeyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("81"))

	networkContainerRunningStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("82"))

	networkContainerStoppedStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("245"))
)

// NetworkDetailView 网络详情视图
type NetworkDetailView struct {
	dockerClient docker.Client

	// UI 尺寸
	width  int
	height int

	// 数据
	network *docker.Network        // 网络基本信息
	details *docker.NetworkDetails // 网络详细信息

	// 标签页状态
	activeTab NetworkDetailTab

	// 滚动状态（用于长内容）
	scrollOffset int
	maxScroll    int

	// 加载状态
	loading  bool
	errorMsg string
}

// NewNetworkDetailView 创建网络详情视图
func NewNetworkDetailView(dockerClient docker.Client, network *docker.Network) *NetworkDetailView {
	return &NetworkDetailView{
		dockerClient: dockerClient,
		network:      network,
		activeTab:    NetworkTabBasicInfo,
		scrollOffset: 0,
	}
}

// Init 初始化视图
func (v *NetworkDetailView) Init() tea.Cmd {
	v.loading = true
	return v.loadNetworkDetails
}

// Update 处理消息
func (v *NetworkDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case networkDetailLoadedMsg:
		v.details = msg.details
		v.loading = false
		v.errorMsg = ""
		return v, nil

	case networkDetailLoadErrorMsg:
		v.loading = false
		v.errorMsg = msg.err.Error()
		return v, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// ESC 返回上一级
			return v, func() tea.Msg { return GoBackMsg{} }
		case "tab", "l", "right":
			v.activeTab = (v.activeTab + 1) % NetworkDetailTab(len(networkTabNames))
			v.scrollOffset = 0
			return v, nil
		case "shift+tab", "h", "left":
			if v.activeTab == 0 {
				v.activeTab = NetworkDetailTab(len(networkTabNames) - 1)
			} else {
				v.activeTab--
			}
			v.scrollOffset = 0
			return v, nil
		case "j", "down":
			if v.scrollOffset < v.maxScroll {
				v.scrollOffset++
			}
			return v, nil
		case "k", "up":
			if v.scrollOffset > 0 {
				v.scrollOffset--
			}
			return v, nil
		case "g":
			v.scrollOffset = 0
			return v, nil
		case "G":
			v.scrollOffset = v.maxScroll
			return v, nil
		case "1":
			v.activeTab = NetworkTabBasicInfo
			v.scrollOffset = 0
			return v, nil
		case "2":
			v.activeTab = NetworkTabIPAM
			v.scrollOffset = 0
			return v, nil
		case "3":
			v.activeTab = NetworkTabContainers
			v.scrollOffset = 0
			return v, nil
		case "4":
			v.activeTab = NetworkTabLabels
			v.scrollOffset = 0
			return v, nil
		case "r":
			v.loading = true
			return v, v.loadNetworkDetails
		}
	}

	return v, nil
}

// View 渲染视图
func (v *NetworkDetailView) View() string {
	var s strings.Builder

	// 标题
	title := "🌐 Network Details"
	if v.network != nil {
		networkName := v.network.Name
		if len(networkName) > 40 {
			networkName = networkName[:37] + "..."
		}
		title = "🌐 " + networkName
	}
	s.WriteString("\n  " + networkDetailTitleStyle.Render(title) + "\n\n")

	// 标签页导航
	s.WriteString(v.renderTabs())
	s.WriteString("\n")

	// 加载中状态
	if v.loading {
		s.WriteString("\n  " + networkDetailHintStyle.Render("⏳ 正在加载网络详情...") + "\n")
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
func (v *NetworkDetailView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// renderTabs 渲染标签页导航
func (v *NetworkDetailView) renderTabs() string {
	var tabs []string

	for i, name := range networkTabNames {
		tabNum := fmt.Sprintf("[%d]", i+1)
		if NetworkDetailTab(i) == v.activeTab {
			tabs = append(tabs, networkTabActiveStyle.Render(tabNum+" "+name))
		} else {
			tabs = append(tabs, networkTabInactiveStyle.Render(tabNum+" "+name))
		}
	}

	return "  " + strings.Join(tabs, "  │  ")
}

// renderCurrentTab 渲染当前标签页内容
func (v *NetworkDetailView) renderCurrentTab() string {
	switch v.activeTab {
	case NetworkTabBasicInfo:
		return v.renderBasicInfo()
	case NetworkTabIPAM:
		return v.renderIPAMConfig()
	case NetworkTabContainers:
		return v.renderContainers()
	case NetworkTabLabels:
		return v.renderLabels()
	default:
		return ""
	}
}

// renderBasicInfo 渲染基本信息
func (v *NetworkDetailView) renderBasicInfo() string {
	if v.details == nil {
		return "\n  " + networkDetailHintStyle.Render("无网络信息")
	}

	var lines []string

	lines = append(lines, v.formatLine("NETWORK ID", v.details.ID))
	lines = append(lines, v.formatLine("NAME", v.details.Name))
	lines = append(lines, v.formatLine("DRIVER", v.details.Driver))
	lines = append(lines, v.formatLine("SCOPE", v.details.Scope))
	lines = append(lines, v.formatLine("CREATED", v.details.Created.Format("2006-01-02 15:04:05")+" ("+formatCreatedTime(v.details.Created)+")"))

	// 布尔属性
	internalStr := "No"
	if v.details.Internal {
		internalStr = "Yes (不能访问外部网络)"
	}
	lines = append(lines, v.formatLine("INTERNAL", internalStr))

	ipv6Str := "No"
	if v.details.IPv6 {
		ipv6Str = "Yes"
	}
	lines = append(lines, v.formatLine("IPv6", ipv6Str))

	attachableStr := "No"
	if v.details.Attachable {
		attachableStr = "Yes (可手动连接容器)"
	}
	lines = append(lines, v.formatLine("ATTACHABLE", attachableStr))

	ingressStr := "No"
	if v.details.Ingress {
		ingressStr = "Yes"
	}
	lines = append(lines, v.formatLine("INGRESS", ingressStr))

	// 驱动选项
	if len(v.details.Options) > 0 {
		lines = append(lines, "")
		lines = append(lines, networkDetailLabelStyle.Render("DRIVER OPTIONS:"))
		for k, val := range v.details.Options {
			lines = append(lines, "  "+networkDetailKeyStyle.Render(k)+" = "+networkDetailValueStyle.Render(val))
		}
	}

	content := strings.Join(lines, "\n")
	boxWidth := v.width - 6
	if boxWidth < 60 {
		boxWidth = 60
	}

	return "\n" + v.wrapInBox("Basic Information", content, boxWidth)
}

// renderIPAMConfig 渲染 IPAM 配置
func (v *NetworkDetailView) renderIPAMConfig() string {
	if v.details == nil {
		return "\n  " + networkDetailHintStyle.Render("无 IPAM 配置信息")
	}

	var lines []string

	// IPAM 驱动
	driver := v.details.IPAM.Driver
	if driver == "" {
		driver = "default"
	}
	lines = append(lines, v.formatLine("IPAM DRIVER", driver))

	// IPAM 选项
	if len(v.details.IPAM.Options) > 0 {
		lines = append(lines, "")
		lines = append(lines, networkDetailLabelStyle.Render("IPAM OPTIONS:"))
		for k, val := range v.details.IPAM.Options {
			lines = append(lines, "  "+networkDetailKeyStyle.Render(k)+" = "+networkDetailValueStyle.Render(val))
		}
	}

	// IP 池配置
	if len(v.details.IPAM.Configs) > 0 {
		lines = append(lines, "")
		lines = append(lines, networkDetailLabelStyle.Render("IP POOLS:")+" ("+fmt.Sprintf("%d", len(v.details.IPAM.Configs))+")")

		for i, cfg := range v.details.IPAM.Configs {
			lines = append(lines, "")
			lines = append(lines, networkDetailKeyStyle.Render(fmt.Sprintf("  Pool #%d:", i+1)))

			if cfg.Subnet != "" {
				lines = append(lines, "    "+networkDetailLabelStyle.Render("Subnet:")+" "+networkDetailValueStyle.Render(cfg.Subnet))
			}
			if cfg.Gateway != "" {
				lines = append(lines, "    "+networkDetailLabelStyle.Render("Gateway:")+" "+networkDetailValueStyle.Render(cfg.Gateway))
			}
			if cfg.IPRange != "" {
				lines = append(lines, "    "+networkDetailLabelStyle.Render("IP Range:")+" "+networkDetailValueStyle.Render(cfg.IPRange))
			}
		}
	} else {
		lines = append(lines, "")
		lines = append(lines, networkDetailHintStyle.Render("无 IP 池配置（使用默认配置）"))
	}

	content := strings.Join(lines, "\n")
	boxWidth := v.width - 6
	if boxWidth < 60 {
		boxWidth = 60
	}

	return "\n" + v.wrapInBox("IPAM Configuration", content, boxWidth)
}

// renderContainers 渲染连接的容器
func (v *NetworkDetailView) renderContainers() string {
	if v.details == nil || len(v.details.Containers) == 0 {
		return "\n  " + networkDetailHintStyle.Render("没有容器连接到此网络")
	}

	var lines []string
	containerCount := len(v.details.Containers)

	// 计算可显示的行数（每个容器占 4-5 行）
	maxItems := (v.height - 15) / 5
	if maxItems < 2 {
		maxItems = 2
	}

	// 应用滚动
	startIdx := v.scrollOffset
	endIdx := startIdx + maxItems
	if endIdx > containerCount {
		endIdx = containerCount
	}
	v.maxScroll = containerCount - maxItems
	if v.maxScroll < 0 {
		v.maxScroll = 0
	}

	for i := startIdx; i < endIdx; i++ {
		c := v.details.Containers[i]

		// 容器 ID（短）
		shortID := c.ContainerID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}

		// 容器名称
		name := c.ContainerName
		if name == "" {
			name = shortID
		}

		lines = append(lines, "")
		lines = append(lines, networkDetailKeyStyle.Render(fmt.Sprintf("  📦 %s", name)))
		lines = append(lines, "    "+networkDetailLabelStyle.Render("Container ID:")+" "+networkDetailValueStyle.Render(shortID))

		if c.IPv4Address != "" {
			lines = append(lines, "    "+networkDetailLabelStyle.Render("IPv4:")+" "+networkDetailValueStyle.Render(c.IPv4Address))
		}
		if c.IPv6Address != "" {
			lines = append(lines, "    "+networkDetailLabelStyle.Render("IPv6:")+" "+networkDetailValueStyle.Render(c.IPv6Address))
		}
		if c.MacAddress != "" {
			lines = append(lines, "    "+networkDetailLabelStyle.Render("MAC:")+" "+networkDetailValueStyle.Render(c.MacAddress))
		}
	}

	// 滚动提示
	if v.maxScroll > 0 {
		scrollInfo := fmt.Sprintf("(%d/%d) ", v.scrollOffset+1, containerCount)
		if v.scrollOffset > 0 {
			scrollInfo += "↑ "
		}
		if v.scrollOffset < v.maxScroll {
			scrollInfo += "↓"
		}
		lines = append(lines, "")
		lines = append(lines, networkDetailHintStyle.Render(scrollInfo+"  j/k 滚动"))
	}

	content := strings.Join(lines, "\n")
	boxWidth := v.width - 6
	if boxWidth < 60 {
		boxWidth = 60
	}

	title := fmt.Sprintf("Connected Containers (%d)", containerCount)
	return "\n" + v.wrapInBox(title, content, boxWidth)
}

// renderLabels 渲染标签信息
func (v *NetworkDetailView) renderLabels() string {
	if v.details == nil || len(v.details.Labels) == 0 {
		return "\n  " + networkDetailHintStyle.Render("无标签信息")
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
		lines = append(lines, networkDetailHintStyle.Render(scrollInfo+"  j/k 滚动"))
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
func (v *NetworkDetailView) renderHints() string {
	hints := []string{
		networkDetailKeyStyle.Render("<Tab/←/→>") + " 切换标签",
		networkDetailKeyStyle.Render("<1-4>") + " 快速跳转",
		networkDetailKeyStyle.Render("<j/k>") + " 滚动",
		networkDetailKeyStyle.Render("<r>") + " 刷新",
		networkDetailKeyStyle.Render("<Esc>") + " 返回",
	}

	return "  " + networkDetailHintStyle.Render(strings.Join(hints, "  │  "))
}

// formatLine 格式化一行信息
func (v *NetworkDetailView) formatLine(label, value string) string {
	return networkDetailLabelStyle.Render(label+":") + " " + networkDetailValueStyle.Render(value)
}

// wrapInBox 将内容包装在边框中
func (v *NetworkDetailView) wrapInBox(title, content string, width int) string {
	boxStyle := networkDetailBoxStyle.Width(width)
	titleLine := "  " + networkDetailTitleStyle.Render("─ "+title+" ") + networkDetailHintStyle.Render(strings.Repeat("─", width-len(title)-6))
	return titleLine + "\n" + boxStyle.Render(content)
}

// loadNetworkDetails 加载网络详情
func (v *NetworkDetailView) loadNetworkDetails() tea.Msg {
	if v.network == nil {
		return networkDetailLoadErrorMsg{err: fmt.Errorf("网络信息为空")}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	details, err := v.dockerClient.NetworkDetails(ctx, v.network.ID)
	if err != nil {
		return networkDetailLoadErrorMsg{err: err}
	}

	return networkDetailLoadedMsg{details: details}
}

// 消息类型
type networkDetailLoadedMsg struct {
	details *docker.NetworkDetails
}

type networkDetailLoadErrorMsg struct {
	err error
}
