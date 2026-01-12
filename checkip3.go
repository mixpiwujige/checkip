package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
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

// parseServerInfo 函数用于解析配置文件中的服务器信息
func parseAllConfigFiles(folderPath string) ([]ServerInfo, error) {
    entries, err := os.ReadDir(folderPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read directory: %w", err)
    }

    var serverInfos []ServerInfo

    for _, entry := range entries {
        if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".conf") {
            continue // 只处理 .conf 文件
        }

        filePath := fmt.Sprintf("%s/%s", folderPath, entry.Name())
        infos, err := parseServerInfo(filePath)
        if err != nil {
            return nil, fmt.Errorf("error parsing file %s: %w", filePath, err)
        }

        serverInfos = append(serverInfos, infos...)
    }

    return serverInfos, nil
}

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: ./program <config-folder-path>")
        return
    }
    configFolderPath := os.Args[1]

    serverInfos, err := parseAllConfigFiles(configFolderPath)
    if err != nil {
        fmt.Println("Error parsing server info:", err)
        return
    }

    var wg sync.WaitGroup
    results := make(chan string)

    // 设置超时时间为5秒
    timeout := 5 * time.Second
    concurrentLimit := 10
    semaphore := make(chan struct{}, concurrentLimit)

    logFileName := fmt.Sprintf("connectinfo_%s.log", time.Now().Format("2006-01-02"))
    logFile, err := os.Create(logFileName)
    if err != nil {
        fmt.Println("Error creating log file:", err)
        return
    }
    defer logFile.Close()

    for _, info := range serverInfos {
        wg.Add(1)
        semaphore <- struct{}{} // 控制并发
        go func(info ServerInfo) {
            defer wg.Done()
            defer func() { <-semaphore }()
            checkConnectivity(info, timeout, results)
        }(info)
    }

    go func() {
        wg.Wait()
        close(results)
    }()

    for result := range results {
        fmt.Println(result)
        if _, err := fmt.Fprintln(logFile, result); err != nil {
            fmt.Println("Error writing to log file:", err)
            return
        }
    }
}