package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Sparkline ASCII 折线图组件
type Sparkline struct {
	Title     string
	Data      []float64
	Max       float64
	Width     int
	Height    int
	Color     string
	ShowScale bool
	Unit      string
}

// NewSparkline 创建折线图
func NewSparkline(title string, width, height int) *Sparkline {
	return &Sparkline{
		Title:     title,
		Data:      make([]float64, 0),
		Width:     width,
		Height:    height,
		Color:     "82",
		ShowScale: true,
	}
}

// AddPoint 添加数据点
func (s *Sparkline) AddPoint(value float64) {
	s.Data = append(s.Data, value)
	maxPoints := s.Width - 8
	if maxPoints < 10 {
		maxPoints = 10
	}
	if len(s.Data) > maxPoints {
		s.Data = s.Data[len(s.Data)-maxPoints:]
	}
}

// SetData 设置数据
func (s *Sparkline) SetData(data []float64) {
	s.Data = data
}

// Render 渲染折线图
func (s *Sparkline) Render() string {
	if len(s.Data) == 0 {
		return s.renderEmpty()
	}

	maxVal := s.Max
	if maxVal == 0 {
		for _, v := range s.Data {
			if v > maxVal {
				maxVal = v
			}
		}
	}
	if maxVal == 0 {
		maxVal = 100
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	lineStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(s.Color))

	axisStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	scaleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	chars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	var lines []string

	titleLine := titleStyle.Render(s.Title)
	if s.Unit != "" {
		titleLine += " " + scaleStyle.Render("("+s.Unit+")")
	}
	lines = append(lines, titleLine)

	chartWidth := s.Width - 8
	if chartWidth < 10 {
		chartWidth = 10
	}
	chartHeight := s.Height - 2
	if chartHeight < 3 {
		chartHeight = 3
	}

	for row := chartHeight - 1; row >= 0; row-- {
		var line strings.Builder

		if s.ShowScale {
			scaleVal := maxVal * float64(row+1) / float64(chartHeight)
			line.WriteString(scaleStyle.Render(fmt.Sprintf("%5.0f", scaleVal)))
			line.WriteString(axisStyle.Render("┤"))
		}

		for i := 0; i < chartWidth; i++ {
			dataIdx := len(s.Data) - chartWidth + i
			if dataIdx < 0 || dataIdx >= len(s.Data) {
				line.WriteString(" ")
				continue
			}

			val := s.Data[dataIdx]
			normalizedVal := val / maxVal * float64(chartHeight)

			if normalizedVal >= float64(row+1) {
				line.WriteString(lineStyle.Render(string(chars[7])))
			} else if normalizedVal > float64(row) {
				fraction := normalizedVal - float64(row)
				charIdx := int(fraction * 8)
				if charIdx > 7 {
					charIdx = 7
				}
				if charIdx < 0 {
					charIdx = 0
				}
				line.WriteString(lineStyle.Render(string(chars[charIdx])))
			} else {
				line.WriteString(" ")
			}
		}

		lines = append(lines, line.String())
	}

	var xAxis strings.Builder
	if s.ShowScale {
		xAxis.WriteString(scaleStyle.Render("    0"))
		xAxis.WriteString(axisStyle.Render("└"))
	}
	xAxis.WriteString(axisStyle.Render(strings.Repeat("─", chartWidth)))
	lines = append(lines, xAxis.String())

	return strings.Join(lines, "\n")
}

func (s *Sparkline) renderEmpty() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	return titleStyle.Render(s.Title) + "\n" + hintStyle.Render("  等待数据...")
}
