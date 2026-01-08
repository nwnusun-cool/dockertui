package compose

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CommandType compose 命令类型
type CommandType int

const (
	CommandUnknown CommandType = iota
	CommandDockerCompose       // docker compose (Docker CLI v2 插件)
	CommandDockerComposeV1     // docker-compose (独立工具)
)

// composeClient docker-compose 客户端实现
type composeClient struct {
	commandType CommandType
	version     string
	mu          sync.RWMutex
}

// 全局客户端实例
var (
	defaultClient *composeClient
	clientOnce    sync.Once
	initError     error
)

// NewClient 创建新的 compose 客户端
func NewClient() (Client, error) {
	clientOnce.Do(func() {
		defaultClient, initError = detectAndCreateClient()
	})
	
	if initError != nil {
		return nil, initError
	}
	return defaultClient, nil
}

// detectAndCreateClient 检测并创建客户端
func detectAndCreateClient() (*composeClient, error) {
	client := &composeClient{}
	
	// 优先检测 docker compose (v2)
	if version, err := checkDockerComposeV2(); err == nil {
		client.commandType = CommandDockerCompose
		client.version = version
		return client, nil
	}
	
	// 回退到 docker-compose (v1)
	if version, err := checkDockerComposeV1(); err == nil {
		client.commandType = CommandDockerComposeV1
		client.version = version
		return client, nil
	}
	
	return nil, &ComposeError{
		Type:       ErrorNotFound,
		Message:    "docker compose 命令未找到",
		Details:    "请确保已安装 Docker Desktop 或 docker-compose",
		Suggestion: "安装 Docker Desktop: https://www.docker.com/products/docker-desktop",
	}
}

// checkDockerComposeV2 检测 docker compose v2
func checkDockerComposeV2() (string, error) {
	cmd := exec.Command("docker", "compose", "version", "--short")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// checkDockerComposeV1 检测 docker-compose v1
func checkDockerComposeV1() (string, error) {
	cmd := exec.Command("docker-compose", "version", "--short")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// Version 返回版本信息
func (c *composeClient) Version() (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.version, nil
}

// CommandType 返回命令类型字符串
func (c *composeClient) CommandType() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	switch c.commandType {
	case CommandDockerCompose:
		return "docker compose"
	case CommandDockerComposeV1:
		return "docker-compose"
	default:
		return "unknown"
	}
}

// buildCommand 构建命令
func (c *composeClient) buildCommand(project *Project, args ...string) *exec.Cmd {
	var cmdArgs []string
	
	// 添加项目相关参数
	projectArgs := c.buildProjectArgs(project)
	
	switch c.commandType {
	case CommandDockerCompose:
		// docker compose [project-args] [command] [args]
		cmdArgs = append([]string{"compose"}, projectArgs...)
		cmdArgs = append(cmdArgs, args...)
		return exec.Command("docker", cmdArgs...)
		
	case CommandDockerComposeV1:
		// docker-compose [project-args] [command] [args]
		cmdArgs = append(projectArgs, args...)
		return exec.Command("docker-compose", cmdArgs...)
		
	default:
		return nil
	}
}

// buildProjectArgs 构建项目相关参数
func (c *composeClient) buildProjectArgs(project *Project) []string {
	var args []string
	
	if project == nil {
		return args
	}
	
	// 项目名称
	if project.Name != "" {
		args = append(args, "-p", project.Name)
	}
	
	// compose 文件
	for _, f := range project.ComposeFiles {
		filePath := f
		if project.Path != "" && !filepath.IsAbs(f) {
			filePath = filepath.Join(project.Path, f)
		}
		args = append(args, "-f", filePath)
	}
	
	// 环境变量文件
	for _, f := range project.EnvFiles {
		filePath := f
		if project.Path != "" && !filepath.IsAbs(f) {
			filePath = filepath.Join(project.Path, f)
		}
		args = append(args, "--env-file", filePath)
	}
	
	return args
}

// runCommand 执行命令并返回结果
func (c *composeClient) runCommand(project *Project, args ...string) (*OperationResult, error) {
	start := time.Now()
	
	cmd := c.buildCommand(project, args...)
	if cmd == nil {
		return nil, &ComposeError{
			Type:    ErrorUnknown,
			Message: "无法构建命令",
		}
	}
	
	// 设置工作目录
	if project != nil && project.Path != "" {
		cmd.Dir = project.Path
	}
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	
	result := &OperationResult{
		Output:   stdout.String(),
		Error:    stderr.String(),
		Duration: time.Since(start),
	}
	
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		result.Success = false
		result.Message = c.parseErrorMessage(stderr.String())
		return result, c.wrapError(err, stderr.String())
	}
	
	result.Success = true
	result.ExitCode = 0
	result.Message = "操作成功"
	return result, nil
}

// parseErrorMessage 解析错误消息
func (c *composeClient) parseErrorMessage(stderr string) string {
	lines := strings.Split(stderr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "time=") {
			return line
		}
	}
	return "操作失败"
}

// wrapError 包装错误
func (c *composeClient) wrapError(err error, stderr string) error {
	composeErr := &ComposeError{
		Type:    ErrorUnknown,
		Message: err.Error(),
		Details: stderr,
	}
	
	stderrLower := strings.ToLower(stderr)
	
	// 检测错误类型
	switch {
	case strings.Contains(stderrLower, "yaml") || strings.Contains(stderrLower, "parse"):
		composeErr.Type = ErrorConfig
		composeErr.Suggestion = "请检查 compose 文件语法"
		
	case strings.Contains(stderrLower, "port") && strings.Contains(stderrLower, "already"):
		composeErr.Type = ErrorNetwork
		composeErr.Suggestion = "端口已被占用，请检查端口冲突"
		
	case strings.Contains(stderrLower, "network"):
		composeErr.Type = ErrorNetwork
		composeErr.Suggestion = "网络配置错误，请检查网络设置"
		
	case strings.Contains(stderrLower, "image") || strings.Contains(stderrLower, "pull"):
		composeErr.Type = ErrorImage
		composeErr.Suggestion = "镜像相关错误，请检查镜像名称或网络连接"
		
	case strings.Contains(stderrLower, "permission") || strings.Contains(stderrLower, "denied"):
		composeErr.Type = ErrorPermission
		composeErr.Suggestion = "权限不足，请检查文件权限或以管理员身份运行"
		
	case strings.Contains(stderrLower, "not found") || strings.Contains(stderrLower, "no such"):
		composeErr.Type = ErrorNotFound
		composeErr.Suggestion = "文件或资源未找到，请检查路径"
	}
	
	return composeErr
}

// Up 启动项目
func (c *composeClient) Up(project *Project, opts UpOptions) (*OperationResult, error) {
	args := []string{"up"}
	
	if opts.Detach {
		args = append(args, "-d")
	}
	if opts.Build {
		args = append(args, "--build")
	}
	if opts.ForceRecreate {
		args = append(args, "--force-recreate")
	}
	if opts.NoDeps {
		args = append(args, "--no-deps")
	}
	if opts.Pull != "" {
		args = append(args, "--pull", opts.Pull)
	}
	if opts.Timeout > 0 {
		args = append(args, "-t", strconv.Itoa(opts.Timeout))
	}
	
	// 添加服务名称
	args = append(args, opts.Services...)
	
	return c.runCommand(project, args...)
}

// Down 停止项目
func (c *composeClient) Down(project *Project, opts DownOptions) (*OperationResult, error) {
	args := []string{"down"}
	
	if opts.RemoveVolumes {
		args = append(args, "-v")
	}
	if opts.RemoveOrphans {
		args = append(args, "--remove-orphans")
	}
	if opts.RemoveImages != "" {
		args = append(args, "--rmi", opts.RemoveImages)
	}
	if opts.Timeout > 0 {
		args = append(args, "-t", strconv.Itoa(opts.Timeout))
	}
	
	return c.runCommand(project, args...)
}

// Start 启动已存在的容器
func (c *composeClient) Start(project *Project, services []string) (*OperationResult, error) {
	args := []string{"start"}
	args = append(args, services...)
	return c.runCommand(project, args...)
}

// Stop 停止容器
func (c *composeClient) Stop(project *Project, services []string, timeout int) (*OperationResult, error) {
	args := []string{"stop"}
	if timeout > 0 {
		args = append(args, "-t", strconv.Itoa(timeout))
	}
	args = append(args, services...)
	return c.runCommand(project, args...)
}

// Restart 重启服务
func (c *composeClient) Restart(project *Project, services []string, timeout int) (*OperationResult, error) {
	args := []string{"restart"}
	if timeout > 0 {
		args = append(args, "-t", strconv.Itoa(timeout))
	}
	args = append(args, services...)
	return c.runCommand(project, args...)
}

// Pause 暂停服务
func (c *composeClient) Pause(project *Project, services []string) (*OperationResult, error) {
	args := []string{"pause"}
	args = append(args, services...)
	return c.runCommand(project, args...)
}

// Unpause 恢复服务
func (c *composeClient) Unpause(project *Project, services []string) (*OperationResult, error) {
	args := []string{"unpause"}
	args = append(args, services...)
	return c.runCommand(project, args...)
}

// PS 获取服务状态
func (c *composeClient) PS(project *Project) ([]Service, error) {
	// 使用 --format json 获取结构化输出
	args := []string{"ps", "--format", "json", "-a"}
	
	cmd := c.buildCommand(project, args...)
	if cmd == nil {
		return nil, &ComposeError{
			Type:    ErrorUnknown,
			Message: "无法构建命令",
		}
	}
	
	if project != nil && project.Path != "" {
		cmd.Dir = project.Path
	}
	
	output, err := cmd.Output()
	if err != nil {
		// 如果 json 格式失败，尝试普通格式
		return c.psLegacy(project)
	}
	
	return c.parseJSONPS(string(output))
}

// psLegacy 使用传统格式获取服务状态
func (c *composeClient) psLegacy(project *Project) ([]Service, error) {
	args := []string{"ps", "-a"}
	
	cmd := c.buildCommand(project, args...)
	if cmd == nil {
		return nil, &ComposeError{
			Type:    ErrorUnknown,
			Message: "无法构建命令",
		}
	}
	
	if project != nil && project.Path != "" {
		cmd.Dir = project.Path
	}
	
	output, err := cmd.Output()
	if err != nil {
		return nil, c.wrapError(err, "")
	}
	
	return c.parseLegacyPS(string(output))
}

// parseJSONPS 解析 JSON 格式的 ps 输出
func (c *composeClient) parseJSONPS(output string) ([]Service, error) {
	// docker compose ps --format json 输出每行一个 JSON 对象
	var services []Service
	serviceMap := make(map[string]*Service)
	
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "[]" {
			continue
		}
		
		// 简单解析 JSON（避免引入 encoding/json 的复杂性）
		// 格式: {"Name":"xxx","Service":"xxx","State":"xxx",...}
		name := extractJSONField(line, "Service")
		if name == "" {
			name = extractJSONField(line, "Name")
		}
		state := extractJSONField(line, "State")
		image := extractJSONField(line, "Image")
		containerID := extractJSONField(line, "ID")
		
		if name == "" {
			continue
		}
		
		// 从容器名称中提取服务名
		serviceName := extractServiceName(name)
		
		if svc, exists := serviceMap[serviceName]; exists {
			svc.Containers = append(svc.Containers, containerID)
			svc.Replicas++
			if state == "running" {
				svc.Running++
			}
		} else {
			svc := &Service{
				Name:       serviceName,
				Image:      image,
				State:      state,
				Containers: []string{containerID},
				Replicas:   1,
				Running:    0,
			}
			if state == "running" {
				svc.Running = 1
			}
			serviceMap[serviceName] = svc
		}
	}
	
	for _, svc := range serviceMap {
		// 更新状态
		if svc.Running == 0 {
			svc.State = "exited"
		} else if svc.Running < svc.Replicas {
			svc.State = "partial"
		} else {
			svc.State = "running"
		}
		services = append(services, *svc)
	}
	
	return services, nil
}

// parseLegacyPS 解析传统格式的 ps 输出
func (c *composeClient) parseLegacyPS(output string) ([]Service, error) {
	var services []Service
	serviceMap := make(map[string]*Service)
	
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		// 跳过标题行
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		
		// 解析每行
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		
		name := fields[0]
		serviceName := extractServiceName(name)
		
		// 查找状态字段（通常包含 Up 或 Exit）
		state := "unknown"
		for _, f := range fields {
			fLower := strings.ToLower(f)
			if strings.Contains(fLower, "up") || fLower == "running" {
				state = "running"
				break
			} else if strings.Contains(fLower, "exit") || fLower == "exited" {
				state = "exited"
				break
			} else if strings.Contains(fLower, "pause") || fLower == "paused" {
				state = "paused"
				break
			}
		}
		
		if svc, exists := serviceMap[serviceName]; exists {
			svc.Replicas++
			if state == "running" {
				svc.Running++
			}
		} else {
			svc := &Service{
				Name:     serviceName,
				State:    state,
				Replicas: 1,
				Running:  0,
			}
			if state == "running" {
				svc.Running = 1
			}
			serviceMap[serviceName] = svc
		}
	}
	
	for _, svc := range serviceMap {
		if svc.Running == 0 {
			svc.State = "exited"
		} else if svc.Running < svc.Replicas {
			svc.State = "partial"
		} else {
			svc.State = "running"
		}
		services = append(services, *svc)
	}
	
	return services, nil
}

// extractJSONField 从 JSON 字符串中提取字段值
func extractJSONField(json, field string) string {
	// 简单的正则匹配
	pattern := fmt.Sprintf(`"%s"\s*:\s*"([^"]*)"`, field)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(json)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractServiceName 从容器名称中提取服务名
func extractServiceName(containerName string) string {
	// 容器名称格式通常是: project_service_1 或 project-service-1
	parts := strings.Split(containerName, "_")
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	parts = strings.Split(containerName, "-")
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return containerName
}

// Logs 获取日志流
func (c *composeClient) Logs(project *Project, opts LogOptions) (io.ReadCloser, error) {
	args := []string{"logs"}
	
	if opts.Follow {
		args = append(args, "-f")
	}
	if opts.Tail > 0 {
		args = append(args, "--tail", strconv.Itoa(opts.Tail))
	}
	if opts.Timestamps {
		args = append(args, "-t")
	}
	if opts.Since != "" {
		args = append(args, "--since", opts.Since)
	}
	if opts.Until != "" {
		args = append(args, "--until", opts.Until)
	}
	
	// 添加服务名称
	args = append(args, opts.Services...)
	
	cmd := c.buildCommand(project, args...)
	if cmd == nil {
		return nil, &ComposeError{
			Type:    ErrorUnknown,
			Message: "无法构建命令",
		}
	}
	
	if project != nil && project.Path != "" {
		cmd.Dir = project.Path
	}
	
	// 获取 stdout 管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, c.wrapError(err, "")
	}
	
	// 启动命令
	if err := cmd.Start(); err != nil {
		return nil, c.wrapError(err, "")
	}
	
	// 返回一个包装的 ReadCloser，在关闭时会终止进程
	return &logReader{
		reader: stdout,
		cmd:    cmd,
	}, nil
}

// logReader 日志读取器
type logReader struct {
	reader io.ReadCloser
	cmd    *exec.Cmd
}

func (r *logReader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

func (r *logReader) Close() error {
	r.reader.Close()
	if r.cmd.Process != nil {
		r.cmd.Process.Kill()
	}
	return r.cmd.Wait()
}

// Config 获取合并后的配置
func (c *composeClient) Config(project *Project) (string, error) {
	args := []string{"config"}
	
	cmd := c.buildCommand(project, args...)
	if cmd == nil {
		return "", &ComposeError{
			Type:    ErrorUnknown,
			Message: "无法构建命令",
		}
	}
	
	if project != nil && project.Path != "" {
		cmd.Dir = project.Path
	}
	
	output, err := cmd.Output()
	if err != nil {
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		return "", c.wrapError(err, stderr.String())
	}
	
	return string(output), nil
}

// Build 构建镜像
func (c *composeClient) Build(project *Project, opts BuildOptions) (*OperationResult, error) {
	args := []string{"build"}
	
	if opts.NoCache {
		args = append(args, "--no-cache")
	}
	if opts.Pull {
		args = append(args, "--pull")
	}
	
	// 添加服务名称
	args = append(args, opts.Services...)
	
	return c.runCommand(project, args...)
}

// Pull 拉取镜像
func (c *composeClient) Pull(project *Project, opts PullOptions) (*OperationResult, error) {
	args := []string{"pull"}
	
	if opts.IgnorePullFailures {
		args = append(args, "--ignore-pull-failures")
	}
	
	// 添加服务名称
	args = append(args, opts.Services...)
	
	return c.runCommand(project, args...)
}

// ReadLogs 读取日志（非流式，返回字符串）
func (c *composeClient) ReadLogs(project *Project, opts LogOptions) (string, error) {
	// 确保不是 follow 模式
	opts.Follow = false
	
	args := []string{"logs"}
	
	if opts.Tail > 0 {
		args = append(args, "--tail", strconv.Itoa(opts.Tail))
	}
	if opts.Timestamps {
		args = append(args, "-t")
	}
	if opts.Since != "" {
		args = append(args, "--since", opts.Since)
	}
	if opts.Until != "" {
		args = append(args, "--until", opts.Until)
	}
	
	args = append(args, opts.Services...)
	
	cmd := c.buildCommand(project, args...)
	if cmd == nil {
		return "", &ComposeError{
			Type:    ErrorUnknown,
			Message: "无法构建命令",
		}
	}
	
	if project != nil && project.Path != "" {
		cmd.Dir = project.Path
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), c.wrapError(err, string(output))
	}
	
	return string(output), nil
}

// StreamLogs 流式读取日志并通过 channel 返回
func (c *composeClient) StreamLogs(project *Project, opts LogOptions) (<-chan string, <-chan error, func()) {
	logChan := make(chan string, 100)
	errChan := make(chan error, 1)
	
	opts.Follow = true
	
	reader, err := c.Logs(project, opts)
	if err != nil {
		errChan <- err
		close(logChan)
		close(errChan)
		return logChan, errChan, func() {}
	}
	
	cancel := func() {
		reader.Close()
	}
	
	go func() {
		defer close(logChan)
		defer close(errChan)
		defer reader.Close()
		
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			logChan <- scanner.Text()
		}
		
		if err := scanner.Err(); err != nil {
			errChan <- err
		}
	}()
	
	return logChan, errChan, cancel
}
