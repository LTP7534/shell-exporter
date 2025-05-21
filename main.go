package main

import (
	"fmt"
	"regexp"
	"os"
	"os/exec"
	"time"
	"net/http"
	"log"
	"sync"
	"gopkg.in/yaml.v3"
	"strings"
	"flag"
)

const defaultInterval = 300 * time.Second

type EnvVar struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type RawScript struct {
	Name     string   `yaml:"name"`
	Interval string   `yaml:"interval"`
	Env      []EnvVar `yaml:"env"`
}

var allCollectors []Runner
var prometheusMetricRegex = regexp.MustCompile(`^([a-zA-Z0-9_])+ *\{(([a-zA-Z0-9_])+ *= *\"[[:alnum:][:punct:][:space:]\x{4e00}-\x{9fff}]+\",?)*\} +[0-9]+(\.[0-9]+)?$|^([a-zA-Z0-9_])+ +[0-9]+(\.[0-9]+)?$` )

// Collector 通用结构
type Collector struct {
	Name     string
	Interval time.Duration
	Env      []EnvVar
	Type     string 
	Output   string
	scriptPath string
	mu       sync.RWMutex
}
// Runner 接口
type Runner interface {
	Run()
	GetOutput() string
}

// ShellRunner 实现 Runner
type ShellRunner struct {
	// Collector *Collector
	*Collector
}
// PythonRunner 实现 Runner
type PythonRunner struct {
	// Collector *Collector
	*Collector
}
// NewCollector 构建函数
func NewCollector(name string, interval string, env []EnvVar, typ string, scriptPath string) (*Collector, error) {
	var dur time.Duration
	var err error
	if interval == "" {
		dur = defaultInterval
	} else {
		dur, err = time.ParseDuration(interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval for %s: %v", name, err)
		}
	}
	return &Collector{
		Name:     name,
		Interval: dur,
		Env:      env,
		Type:     typ,
		scriptPath: scriptPath,
	}, nil
}
func (c *Collector) SetOutput(output string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Output = output
  }
func (c *Collector) GetOutput() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Output
  }

// 判断文件是否存在
func fileExists(path string) bool {
    _, err := os.Stat(path)
    if err == nil {
        return true // 文件存在
    }
    if os.IsNotExist(err) {
        return false // 文件不存在
    }
    // 其他错误，例如权限错误等
    return false
}

func (s *ShellRunner) Run() {
	fmt.Printf("[ShellRunner] Executing %s every %s\n", s.Name, s.Interval)
	ticker := time.NewTicker(s.Interval)
	if ! fileExists(s.scriptPath + "/" + s.Name){
		fmt.Printf("File %s does not exist\n", s.Name)
		return
	}
	go func() {
		for range ticker.C {
			cmd := exec.Command("sh", s.scriptPath + "/" + s.Name)
			if len(s.Env) > 0 {
				for _, e := range s.Env {
					cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", e.Name, e.Value))
				}
			}
			output, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Sprintf("ERROR: %v\nOUTPUT:\n%s", err, output)
			  } else {
				s.SetOutput(string(output))
			  }
		}
	}()
}


func (p *PythonRunner) Run() {
	fmt.Printf("[PythonRunner] Executing %s every %s\n", p.Name, p.Interval)
	ticker := time.NewTicker(p.Interval)
	if ! fileExists(p.scriptPath + "/" + p.Name) {
		fmt.Printf("File %s does not exist\n", p.Name)
		return
	}
	go func() {
		for range ticker.C {
			cmd := exec.Command("python", p.scriptPath + "/" + p.Name)
			if len(p.Env) > 0 {
				for _, e := range p.Env {
					cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", e.Name, e.Value))
				}
			}
			output, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Sprintf("ERROR: %v\nOUTPUT:\n%s", err, output)
			  } else {
				p.SetOutput(string(output))
			  }
		}
	}()
}

// 加载 config.yaml
func LoadConfig(path string, scriptPath string) ([]Runner, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw map[string][]RawScript
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var runners []Runner
	for typ, scripts := range raw {
		if len(scripts) == 0 {
			continue // skip empty types
		}
		for _, script := range scripts {
			c, err := NewCollector(script.Name, script.Interval, script.Env, typ, scriptPath)
			if err != nil {
				return nil, err
			}
			switch typ {
			case "shell":
				runners = append(runners, &ShellRunner{Collector: c})
			case "python":
				runners = append(runners, &PythonRunner{Collector: c})
			default:
				fmt.Printf("Unsupported type: %s\n", typ)
			}
		}
	}
	return runners, nil
}
func convertToPrometheusMetrics(output string) []string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var metrics []string
	for _, line := range lines {
		if line == "" {
			continue
		}
		if ! prometheusMetricRegex.MatchString(line) {
			log.Printf("invalid Prometheus metric line: %q", line)
			continue
		}
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

func main() {
	config := flag.String("config", "example/config.yaml", "path of the config file")
	webListenAdderss := flag.String("web.listen-adderss", ":8080", "address to listen on for web interface and telemetry")
	scriptPath := flag.String("script-path", "example", "path of the script file")
	flag.Parse()
	runners, err := LoadConfig(*config, *scriptPath)
	if err != nil {
		panic(err)
	}
	for _, runner := range runners {
		runner.Run()
		allCollectors = append(allCollectors, runner)
	}
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		for _, c := range allCollectors {
			// if c.GetOutput() != "" {
			    metrics := convertToPrometheusMetrics(c.GetOutput())
			    for _, metric := range metrics {
			        fmt.Fprintf(w, "%s\n", metric)
			    }
		    //    io.WriteString(w, fmt.Sprintf("%s",  c.GetOutput()))
			// }
		}
	  })
	
	  log.Println("HTTP server running on %s", *webListenAdderss)
	  log.Fatal(http.ListenAndServe(*webListenAdderss, nil))
}
