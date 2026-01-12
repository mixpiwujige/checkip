package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ServerInfo 结构体用于存储服务器信息
type ServerInfo struct {
	AppName    string
	ServerIP   string
	ServerID   int
	ServerPort int
}

// CheckResult 存储检查结果
type CheckResult struct {
	ServerInfo ServerInfo
	IsSuccess  bool
	Error      string
	CheckTime  time.Time
	Duration   time.Duration
}

// Config 存储程序配置
type Config struct {
	Timeout         time.Duration
	ConcurrentLimit int
	RetryCount      int
	RetryDelay      time.Duration
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Timeout:         5 * time.Second,
		ConcurrentLimit: 10,
		RetryCount:      3,
		RetryDelay:      time.Second,
	}
}

// parseServerInfo 解析单个配置文件
func parseServerInfo(filePath string) ([]ServerInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开配置文件 %s: %w", filePath, err)
	}
	defer file.Close()

	var serverInfos []ServerInfo
	var currentInfo ServerInfo
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")

		switch key {
		case "appName":
			currentInfo.AppName = value
		case "serverIP":
			currentInfo.ServerIP = value
		case "serverID":
			id, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("解析 serverID 失败 %s: %w", value, err)
			}
			currentInfo.ServerID = id
		case "serverPort":
			port, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("解析 serverPort 失败 %s: %w", value, err)
			}
			currentInfo.ServerPort = port
			// 当端口解析完成时，说明一个完整的服务器信息已收集完毕
			serverInfos = append(serverInfos, currentInfo)
			currentInfo = ServerInfo{} // 重置当前信息
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取配置文件出错 %s: %w", filePath, err)
	}

	return serverInfos, nil
}

// parseAllConfigFiles 解析目录下所有配置文件
func parseAllConfigFiles(folderPath string) ([]ServerInfo, error) {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败 %s: %w", folderPath, err)
	}

	var allServerInfos []ServerInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".conf") {
			continue
		}

		filePath := filepath.Join(folderPath, entry.Name())
		infos, err := parseServerInfo(filePath)
		if err != nil {
			fmt.Printf("警告: 解析文件 %s 失败: %v\n", filePath, err)
			continue // 继续处理其他文件
		}
		allServerInfos = append(allServerInfos, infos...)
	}

	if len(allServerInfos) == 0 {
		return nil, fmt.Errorf("未在目录 %s 中找到有效的配置", folderPath)
	}

	return allServerInfos, nil
}

// checkConnectivity 检查服务器连通性
func checkConnectivity(ctx context.Context, info ServerInfo, config Config) CheckResult {
	result := CheckResult{
		ServerInfo: info,
		CheckTime:  time.Now(),
	}

	// 解析IP地址
	ip := info.ServerIP
	if !strings.Contains(info.ServerIP, ".") {
		ips, err := net.LookupIP(info.ServerIP)
		if err != nil {
			result.Error = fmt.Sprintf("DNS解析失败: %v", err)
			return result
		}
		ip = ips[0].String()
	}

	var lastErr error
	for i := 0; i < config.RetryCount; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				result.Error = "操作被取消"
				return result
			case <-time.After(config.RetryDelay):
			}
		}

		start := time.Now()
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, info.ServerPort), config.Timeout)
		result.Duration = time.Since(start)

		if err == nil {
			conn.Close()
			result.IsSuccess = true
			return result
		}
		lastErr = err
	}

	result.Error = lastErr.Error()
	return result
}

// formatResult 格式化检查结果
func formatResult(result CheckResult) string {
	status := "成功"
	if !result.IsSuccess {
		status = fmt.Sprintf("失败 (%s)", result.Error)
	}
	return fmt.Sprintf("[%s] 服务器ID: %d, 应用: %s, IP: %s, 端口: %d, 耗时: %v, 状态: %s",
		result.CheckTime.Format("2006-01-02 15:04:05"),
		result.ServerInfo.ServerID,
		result.ServerInfo.AppName,
		result.ServerInfo.ServerIP,
		result.ServerInfo.ServerPort,
		result.Duration,
		status)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: ./program <配置文件夹路径>")
		return
	}

	// 初始化配置
	config := DefaultConfig()
	configFolderPath := os.Args[1]

	// 解析服务器信息
	serverInfos, err := parseAllConfigFiles(configFolderPath)
	if err != nil {
		fmt.Printf("解析配置文件失败: %v\n", err)
		return
	}

	// 创建日志文件
	logFileName := fmt.Sprintf("connectinfo_%s.log", time.Now().Format("2006-01-02_150405"))
	logFile, err := os.Create(logFileName)
	if err != nil {
		fmt.Printf("创建日志文件失败: %v\n", err)
		return
	}
	defer logFile.Close()

	// 初始化上下文和等待组
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	results := make(chan CheckResult, len(serverInfos))
	semaphore := make(chan struct{}, config.ConcurrentLimit)

	// 启动检查任务
	startTime := time.Now()
	fmt.Printf("开始检查 %d 个服务器的连通性...\n", len(serverInfos))

	for _, info := range serverInfos {
		wg.Add(1)
		go func(info ServerInfo) {
			defer wg.Done()
			semaphore <- struct{}{} // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			result := checkConnectivity(ctx, info, config)
			results <- result
		}(info)
	}

	// 等待所有检查完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 统计结果
	var successCount, failCount int
	for result := range results {
		if result.IsSuccess {
			successCount++
		} else {
			failCount++
		}

		resultStr := formatResult(result)
		fmt.Println(resultStr)
		fmt.Fprintln(logFile, resultStr)
	}

	// 输出总结
	duration := time.Since(startTime)
	summary := fmt.Sprintf("\n检查完成！\n总计: %d\n成功: %d\n失败: %d\n总耗时: %v\n结果已保存至: %s",
		len(serverInfos),
		successCount,
		failCount,
		duration,
		logFileName)

	fmt.Println(summary)
	fmt.Fprintln(logFile, summary)
} 