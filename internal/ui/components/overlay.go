package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// OverlayCentered 将弹出内容居中叠加到基础内容上
// baseContent: 背景内容
// overlayContent: 要叠加的弹出框内容
// screenWidth: 屏幕宽度（用于水平居中）
// screenHeight: 屏幕高度（用于垂直居中，如果为0则使用baseContent的行数）
func OverlayCentered(baseContent, overlayContent string, screenWidth, screenHeight int) string {
	if overlayContent == "" {
		return baseContent
	}

	// 将基础内容按行分割
	baseLines := strings.Split(baseContent, "\n")
	overlayLines := strings.Split(overlayContent, "\n")

	// 使用屏幕高度或基础内容行数
	totalHeight := screenHeight
	if totalHeight <= 0 {
		totalHeight = len(baseLines)
	}

	// 计算叠加内容的尺寸
	overlayHeight := len(overlayLines)
	overlayWidth := 0
	for _, line := range overlayLines {
		lineWidth := lipgloss.Width(line)
		if lineWidth > overlayWidth {
			overlayWidth = lineWidth
		}
	}

	// 计算垂直居中位置
	insertLine := 0
	if totalHeight > overlayHeight {
		insertLine = (totalHeight - overlayHeight) / 2
	}

	// 计算水平居中的左边距
	leftPadding := 0
	if screenWidth > overlayWidth {
		leftPadding = (screenWidth - overlayWidth) / 2
	}

	// 确保基础内容有足够的行数
	for len(baseLines) < totalHeight {
		baseLines = append(baseLines, "")
	}

	// 构建最终输出
	var result strings.Builder
	paddingStr := strings.Repeat(" ", leftPadding)

	for i := 0; i < len(baseLines); i++ {
		overlayIdx := i - insertLine
		if overlayIdx >= 0 && overlayIdx < len(overlayLines) {
			// 在这个位置显示叠加内容
			result.WriteString(paddingStr)
			result.WriteString(overlayLines[overlayIdx])
		} else {
			// 显示原始内容
			result.WriteString(baseLines[i])
		}
		if i < len(baseLines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// OverlayCenteredBox 将带边框的弹出框居中叠加
func OverlayCenteredBox(baseContent, title, content string, screenWidth, screenHeight int, borderColor string) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(1, 2)

	boxWidth := screenWidth / 2
	if boxWidth < 50 {
		boxWidth = 50
	}
	if boxWidth > 80 {
		boxWidth = 80
	}

	var parts []string
	if title != "" {
		titleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(borderColor)).
			Bold(true)
		parts = append(parts, titleStyle.Render(title))
		parts = append(parts, "")
	}
	parts = append(parts, content)

	boxContent := lipgloss.JoinVertical(lipgloss.Left, parts...)
	box := boxStyle.Width(boxWidth).Render(boxContent)

	return OverlayCentered(baseContent, box, screenWidth, screenHeight)
}
