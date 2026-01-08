package compose

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Scanner compose 项目扫描器接口
type Scanner interface {
	// Scan 扫描指定路径下的 compose 项目
	Scan(ctx context.Context, paths []string) ([]Project, error)
	
	// RefreshProject 刷新单个项目状态
	RefreshProject(ctx context.Context, project *Project) error
}

// ScanConfig 扫描配置
type ScanConfig struct {
	MaxDepth       int      // 最大扫描深度（默认 5）
	IgnorePatterns []string // 忽略模式
}

// DefaultScanConfig 默认扫描配置
func DefaultScanConfig() ScanConfig {
	return ScanConfig{
		MaxDepth: 5,
		IgnorePatterns: []string{
			"node_modules",
			"vendor",
			".git",
			"__pycache__",
			".venv",
			"venv",
			".idea",
			".vscode",
			"dist",
			"build",
			"target",
		},
	}
}

// scanner 扫描器实现
type scanner struct {
	client Client
	config ScanConfig
	mu     sync.RWMutex
}

// NewScanner 创建扫描器
func NewScanner(client Client, config ScanConfig) Scanner {
	if config.MaxDepth <= 0 {
		config.MaxDepth = 5
	}
	return &scanner{
		client: client,
		config: config,
	}
}

// composeFilePatterns compose 文件名模式
var composeFilePatterns = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yml",
	"compose.yaml",
}

// Scan 扫描 compose 项目
func (s *scanner) Scan(ctx context.Context, paths []string) ([]Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	projectMap := make(map[string]*Project)
	
	for _, basePath := range paths {
		// 获取绝对路径
		absPath, err := filepath.Abs(basePath)
		if err != nil {
			continue
		}
		
		// 检查路径是否存在
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			continue
		}
		
		// 递归扫描
		err = s.scanDirectory(ctx, absPath, 0, projectMap)
		if err != nil {
			// 继续扫描其他路径
			continue
		}
	}
	
	// 转换为切片
	projects := make([]Project, 0, len(projectMap))
	for _, p := range projectMap {
		projects = append(projects, *p)
	}
	
	return projects, nil
}

// scanDirectory 递归扫描目录
func (s *scanner) scanDirectory(ctx context.Context, dir string, depth int, projectMap map[string]*Project) error {
	// 检查上下文是否取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	// 检查深度限制
	if depth > s.config.MaxDepth {
		return nil
	}
	
	// 检查是否应该忽略
	if s.shouldIgnore(dir) {
		return nil
	}
	
	// 读取目录内容
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	
	// 查找 compose 文件
	var composeFiles []string
	var overrideFiles []string
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		name := entry.Name()
		nameLower := strings.ToLower(name)
		
		// 检查是否是主 compose 文件
		for _, pattern := range composeFilePatterns {
			if nameLower == pattern {
				composeFiles = append(composeFiles, name)
				break
			}
		}
		
		// 检查是否是 override 文件
		if strings.HasPrefix(nameLower, "docker-compose.") && 
		   (strings.HasSuffix(nameLower, ".yml") || strings.HasSuffix(nameLower, ".yaml")) &&
		   !isMainComposeFile(nameLower) {
			overrideFiles = append(overrideFiles, name)
		}
	}
	
	// 如果找到 compose 文件，创建项目
	if len(composeFiles) > 0 {
		// 使用目录名作为项目名
		projectName := filepath.Base(dir)
		
		// 合并所有 compose 文件
		allFiles := append(composeFiles, overrideFiles...)
		
		project := &Project{
			Name:         projectName,
			Path:         dir,
			ComposeFiles: allFiles,
			Status:       StatusUnknown,
			LastUpdated:  time.Now(),
		}
		
		// 查找 .env 文件
		envFile := filepath.Join(dir, ".env")
		if _, err := os.Stat(envFile); err == nil {
			project.EnvFiles = []string{".env"}
		}
		
		projectMap[dir] = project
	}
	
	// 递归扫描子目录
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		// 跳过隐藏目录
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		
		subDir := filepath.Join(dir, entry.Name())
		s.scanDirectory(ctx, subDir, depth+1, projectMap)
	}
	
	return nil
}

// shouldIgnore 检查是否应该忽略该路径
func (s *scanner) shouldIgnore(path string) bool {
	base := filepath.Base(path)
	baseLower := strings.ToLower(base)
	
	for _, pattern := range s.config.IgnorePatterns {
		if strings.ToLower(pattern) == baseLower {
			return true
		}
	}
	
	return false
}

// isMainComposeFile 检查是否是主 compose 文件
func isMainComposeFile(name string) bool {
	for _, pattern := range composeFilePatterns {
		if name == pattern {
			return true
		}
	}
	return false
}

// RefreshProject 刷新项目状态
func (s *scanner) RefreshProject(ctx context.Context, project *Project) error {
	if s.client == nil {
		return nil
	}
	
	// 获取服务状态
	services, err := s.client.PS(project)
	if err != nil {
		project.Status = StatusError
		return err
	}
	
	project.Services = services
	project.LastUpdated = time.Now()
	
	// 计算项目状态
	project.Status = calculateProjectStatus(services)
	
	return nil
}

// calculateProjectStatus 计算项目状态
func calculateProjectStatus(services []Service) ProjectStatus {
	if len(services) == 0 {
		return StatusStopped
	}
	
	runningCount := 0
	totalCount := len(services)
	
	for _, svc := range services {
		if svc.State == "running" || svc.Running > 0 {
			runningCount++
		}
	}
	
	if runningCount == 0 {
		return StatusStopped
	}
	if runningCount == totalCount {
		return StatusRunning
	}
	return StatusPartial
}
