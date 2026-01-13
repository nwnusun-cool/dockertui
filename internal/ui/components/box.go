package components

import (
	"docktui/internal/ui/styles"
)

// WrapInBox 用边框包裹内容，统一的边框渲染函数
func WrapInBox(title, content string, width int) string {
	boxStyle := styles.BoxStyle.Width(width)
	titleLine := "  " + styles.TitleStyle.Render("─ "+title)
	return titleLine + "\n" + boxStyle.Render(content)
}
