package search

import (
	"strings"
)

// Match 表示一个搜索匹配
type Match struct {
	Line   int // 行号（0-based）
	Column int // 列位置（0-based）
	Length int // 匹配长度
}

// TextSearcher 通用文本搜索工具
type TextSearcher struct {
	query        string  // 搜索关键词
	matches      []Match // 所有匹配
	currentIndex int     // 当前匹配索引（-1 表示无匹配）
	caseSensitive bool   // 是否区分大小写
}

// NewTextSearcher 创建文本搜索器
func NewTextSearcher() *TextSearcher {
	return &TextSearcher{
		currentIndex: -1,
	}
}

// Search 在文本行中搜索关键词
func (s *TextSearcher) Search(lines []string, query string) {
	s.query = query
	s.matches = nil
	s.currentIndex = -1

	if query == "" {
		return
	}

	searchQuery := query
	if !s.caseSensitive {
		searchQuery = strings.ToLower(query)
	}

	for lineIdx, line := range lines {
		searchLine := line
		if !s.caseSensitive {
			searchLine = strings.ToLower(line)
		}

		// 查找该行中所有匹配
		offset := 0
		for {
			idx := strings.Index(searchLine[offset:], searchQuery)
			if idx == -1 {
				break
			}
			s.matches = append(s.matches, Match{
				Line:   lineIdx,
				Column: offset + idx,
				Length: len(query),
			})
			offset += idx + 1
		}
	}

	// 如果有匹配，定位到第一个
	if len(s.matches) > 0 {
		s.currentIndex = 0
	}
}

// Clear 清除搜索状态
func (s *TextSearcher) Clear() {
	s.query = ""
	s.matches = nil
	s.currentIndex = -1
}

// Next 跳转到下一个匹配
func (s *TextSearcher) Next() *Match {
	if len(s.matches) == 0 {
		return nil
	}
	s.currentIndex = (s.currentIndex + 1) % len(s.matches)
	return &s.matches[s.currentIndex]
}

// Prev 跳转到上一个匹配
func (s *TextSearcher) Prev() *Match {
	if len(s.matches) == 0 {
		return nil
	}
	s.currentIndex--
	if s.currentIndex < 0 {
		s.currentIndex = len(s.matches) - 1
	}
	return &s.matches[s.currentIndex]
}

// Current 获取当前匹配
func (s *TextSearcher) Current() *Match {
	if s.currentIndex < 0 || s.currentIndex >= len(s.matches) {
		return nil
	}
	return &s.matches[s.currentIndex]
}

// Query 获取当前搜索关键词
func (s *TextSearcher) Query() string {
	return s.query
}

// MatchCount 获取匹配总数
func (s *TextSearcher) MatchCount() int {
	return len(s.matches)
}

// CurrentIndex 获取当前匹配索引（1-based，用于显示）
func (s *TextSearcher) CurrentIndex() int {
	if s.currentIndex < 0 {
		return 0
	}
	return s.currentIndex + 1
}

// HasMatches 是否有匹配结果
func (s *TextSearcher) HasMatches() bool {
	return len(s.matches) > 0
}

// IsLineMatched 检查某行是否有匹配
func (s *TextSearcher) IsLineMatched(lineIdx int) bool {
	for _, m := range s.matches {
		if m.Line == lineIdx {
			return true
		}
	}
	return false
}

// IsCurrentMatchLine 检查某行是否是当前匹配所在行
func (s *TextSearcher) IsCurrentMatchLine(lineIdx int) bool {
	if s.currentIndex < 0 || s.currentIndex >= len(s.matches) {
		return false
	}
	return s.matches[s.currentIndex].Line == lineIdx
}

// GetLineMatches 获取某行的所有匹配
func (s *TextSearcher) GetLineMatches(lineIdx int) []Match {
	var result []Match
	for _, m := range s.matches {
		if m.Line == lineIdx {
			result = append(result, m)
		}
	}
	return result
}

// SetCaseSensitive 设置是否区分大小写
func (s *TextSearcher) SetCaseSensitive(sensitive bool) {
	s.caseSensitive = sensitive
}
