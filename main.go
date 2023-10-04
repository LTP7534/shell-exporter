package main

import (
        "regexp"
        "bufio"
	"fmt"
        "sort"
	"net/http"
	"os/exec"
	"os"
	"strings"
        "time"
        "path/filepath"
        "strconv"
)

// LISTEN_PORT  监听端口
// SCRIPTS_PATH 脚本路径
// INTERVAL     脚本执行的时间间隔

func main() {
    LISTEN__PORT := os.Getenv("LISTEN_PORT")
    if LISTEN__PORT == "" {
        LISTEN__PORT = "9592"
    }
    // 启动一个 goroutine 执行子进程任务
    go executeScriptsEvery10Seconds()
    fmt.Println("开始监听端口 :"+LISTEN__PORT)
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
        	filename := "/usr/share/result_shell_exporter.txt" // 替换为您的文件路径
        	pattern := `^([a-zA-Z0-9_])+ *\{(([a-zA-Z0-9_])+ *= *\"[[:alnum:][:punct:][:space:]\x{4e00}-\x{9fff}]+\",?)*\} +[0-9]+(\.[0-9]+)?$|^([a-zA-Z0-9_])+ +[0-9]+(\.[0-9]+)?$`         // 正则表达式模式
		output, err := ReadAndMatchLines(filename, pattern)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error running script: %v", err), http.StatusInternalServerError)
			return
		}

		// 将脚本输出转换成 Prometheus 指标格式
		metrics := convertToPrometheusMetrics(output)

		// 将指标写入 HTTP 响应
		for _, metric := range metrics {
			fmt.Fprintf(w, "%s\n", metric)
		}
	})

	// 启动 HTTP 服务器，监听端口
	//http.ListenAndServe(LISTEN__PORT, nil)
	err := http.ListenAndServe(":" + LISTEN__PORT, nil)
	if err != nil {
		fmt.Printf("启动服务器失败：%s\n", err)
	}
}


// 读取metrics
func ReadAndMatchLines(filename, pattern string) (string, error) {
        fileInfo, err := os.Stat(filename)

        if fileInfo.Size() == 0 {
            time.Sleep(2 * time.Second) // 等待2秒
        }
	// 打开文件以供读取
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// 创建一个正则表达式对象
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}

	// 创建一个字符串变量来保存匹配的行
	//var result string
        var lines []string

	// 创建一个扫描器以逐行读取文件内容
	scanner := bufio.NewScanner(file)

	// 逐行扫描文件内容
	for scanner.Scan() {
		line := scanner.Text()
		// 如果行匹配正则表达式，则将其追加到结果变量中
		//if regex.MatchString(line) {
		//	result += line + "\n"
		//} else {
		//	// 如果存在不匹配的行，返回错误
		//	fmt.Printf("Error: does not comply with the prometheus metrics specification - %s\n", line)
		//	return "", err
		//}
                lines = append(lines, line)
	}

	// 检查扫描错误
	if err := scanner.Err(); err != nil {
		return "", err
	}
        // 排序
        sort.Strings(lines)
	result := ""
	for _, line := range lines {
            if regex.MatchString(line) {
 		result += line + "\n"
            } else {
                // 如果存在不匹配的行，返回错误
                fmt.Printf("Error: does not comply with the prometheus metrics specification - %s\n", line)
                return "", err
            }
	}

	return result, nil
}

func convertToPrometheusMetrics(output string) []string {
	// 将脚本输出转换成 Prometheus 指标格式
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var metrics []string
	for _, line := range lines {
		parts := strings.Split(line, " ")
		if len(parts) != 2 {
			continue
		}
		metricName := parts[0]
		metricValue := parts[1]
		//metric := fmt.Sprintf("custom_metric{name=\"%s\"} %s", metricName, metricValue)
		metric := fmt.Sprintf("%s %s",metricName, metricValue)
		metrics = append(metrics, metric)
	}
	return metrics
}

// 判断文件名是否以 "." 开头的隐藏文件
func isHidden(filename string) bool {
	if len(filename) > 0 && filename[0] == '.' {
		return true
	}
	return false
}

// 执行自定义脚本
func executeScriptsEvery10Seconds() {
        SCRIPTS__PATH := os.Getenv("SCRIPTS_PATH")
        if SCRIPTS__PATH == "" {
           SCRIPTS__PATH = "/scripts"
        }
	intervalStr := os.Getenv("INTERVAL")
	// 如果 INTERVAL 环境变量未设置或为空，则使用默认值 300
	Interval := 300
	if intervalStr != "" {
		// 将环境变量的值转换为整数
		interval, err := strconv.Atoi(intervalStr)
		if err == nil {
			Interval = interval
		} else {
			fmt.Printf("无法解析 INTERVAL 的值：%v\n", err)
		}
	}
	// 脚本目录路径
	scriptDir := SCRIPTS__PATH
	fmt.Printf("开始执行%s下脚本,重复间隔%v秒.\n\n", SCRIPTS__PATH, Interval)
       	filePath := "/usr/share/result_shell_exporter.txt"
	// 循环执行任务
	for {
                currentTime := time.Now()
                format := "2006-01-02 15:04:05.999"
                formattedTime := currentTime.Format(format)
		// 创建文件保存输出
		outputFile, err := os.Create(filePath)
		if err != nil {
			fmt.Printf("创建文件失败：%v\n", err)
			return
		}

		// 遍历脚本目录下的 .sh 文件
		err = filepath.Walk(scriptDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
                       	if info.IsDir() && path != scriptDir {
		  	        return filepath.SkipDir
		        }
			if !isHidden(info.Name()){
                            ext := filepath.Ext(path)
                            if ext == ".sh"{
				// 执行 .sh 脚本文件
				fmt.Printf("[%v] 执行脚本: %s\n", formattedTime, path)
                                cmd := exec.Command("sh",path)
				cmd.Stdin = os.Stdin
				// cmd.Stdout = os.Stdout
                                cmd.Stdout = outputFile
				cmd.Stderr = os.Stderr
				err := cmd.Run()
				if err != nil {
					fmt.Printf("脚本执行出错：%v\n", err)
				}
                            } else if ext == ".py"{
                                // 执行 .py 脚本文件
                                fmt.Printf("[%v] 执行脚本: %s\n", formattedTime, path)
                                cmd := exec.Command("python",path)
                                cmd.Stdin = os.Stdin
                                // cmd.Stdout = os.Stdout
                                cmd.Stdout = outputFile
                                cmd.Stderr = os.Stderr
                                err := cmd.Run()
                                if err != nil {
                                        fmt.Printf("脚本执行出错：%v\n", err)
                                }
                            }
			}
			return nil
		})

		if err != nil {
			fmt.Printf("遍历脚本目录出错：%v\n", err)
		}
		// 等待 10 秒后再次执行
		time.Sleep(time.Duration(Interval) * time.Second)
	}
}
