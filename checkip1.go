package main

import (
	"os"
	"fmt"
	"net"
	"os/exec"
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

// checkConnectivity 函数用于检查指定 IP 和端口的网络连通性
func checkConnectivity(info ServerInfo, timeout time.Duration, wg *sync.WaitGroup, results chan<- string) {
	defer wg.Done()

	// 解析IP地址
	ip := info.ServerIP
	fmt.Println("ip的结果为:", ip)
	if strings.Contains(info.ServerIP, ".") {
		ip = info.ServerIP
	} else {
		ips, err := net.LookupIP(info.ServerIP)
		if err != nil {
			results <- fmt.Sprintf("Server ID: %d, App Name: %s, IP: %s, Port: %d, get ip failed (Failed to resolve IP)", info.ServerID, info.AppName, info.ServerIP, info.ServerPort)
			return
		}
		ip = ips[0].String()
	}

	// 设置连接超时时间并检查网络连接
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, info.ServerPort), timeout)
	if err != nil {
		results <- fmt.Sprintf("Server ID: %d, App Name: %s, IP: %s, Port: %d, get connected failed ! (Error: %s)", info.ServerID, info.AppName, ip, info.ServerPort, err)
		return
	}
	conn.Close()
	results <- fmt.Sprintf("Server ID: %d, App Name: %s, IP: %s, Port: %d, get connected success !", info.ServerID, info.AppName, ip, info.ServerPort)
}

func main() {
	cmd := exec.Command("sh", "-c", "cat *.conf |grep -E 'appName|serverID|serverIP|serverPort' |grep -o '\"[^\"]*\":[^,}]*' | sed 's/\"//g'")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error running command:", err)
		return
	}

	lines := strings.Split(string(output), "\n")
	var serverInfos []ServerInfo
	var currentServerInfo ServerInfo

	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) != 2 {
			continue
		}
		key := strings.TrimSpace(fields[0])
		value := strings.TrimSpace(fields[1])

		switch key {
		case "appName":
			currentServerInfo.AppName = value
		case "serverIP":
			currentServerInfo.ServerIP = value
		case "serverID":
			serverID, _ := strconv.Atoi(value)
			currentServerInfo.ServerID = serverID
		case "serverPort":
			serverPort, _ := strconv.Atoi(value)
			currentServerInfo.ServerPort = serverPort
			serverInfos = append(serverInfos, currentServerInfo)
		}
	}

	var wg sync.WaitGroup
	results := make(chan string)

	// 设置超时时间为5秒
	timeout := 5 * time.Second

	for _, info := range serverInfos {
		wg.Add(1)
		go checkConnectivity(info, timeout, &wg, results)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	logFile, err := os.Create("connectinfo.log")
	if err != nil {
		fmt.Println("Error creating log file:", err)
		return
	}
	defer logFile.Close()

	for result := range results {
		fmt.Println(result)
		_, err := fmt.Fprintln(logFile, result)
		if err != nil {
			fmt.Println("Error writing to log file:", err)
			return
		}
	}
}

