package monitor

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slicer/kube"
	"slicer/util"
	"time"
)

// 与Monarch监控系统沟通客户端
type Monitor struct {
	config        *util.Config
	componentName string
}

func NewMonitor(config *util.Config) *Monitor {
	return &Monitor{config: config, componentName: "kpi-calculator"}
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (m *Monitor) Init(kc *kube.KubeClient) error {
	// 需提前部署好Monarch的相关组件
	// 协程定时运行checkHealth方法检查监控系统的健康状态
	go func() {
		for {
			if err := m.checkHealth(); err != nil {
				slog.Error("监控系统健康检查失败", "error", err)
			} else {
				slog.Info("监控系统健康检查通过")
			}
			// 等待一段时间后再次检查
			time.Sleep(time.Minute * 10) // 每10分钟检查一次
		}
	}()
	// 检查组件是否部署
	return m.checkComponentReady(kc)
}

func (m *Monitor) checkComponentReady(kc *kube.KubeClient) error {
	// 检查组件是否就绪
	deployments, err := kc.GetDeployments(m.config.MonitorNamespace, "app=monarch", fmt.Sprintf("component=%s", m.componentName))
	if err != nil {
		slog.Error("获取部署信息失败", "error", err, "deployment", m.componentName)
		return fmt.Errorf("获取部署信息失败: %v", err)
	}
	if len(deployments) == 0 { // 没有找到相关的部署, 安装
		slog.Info("未找到监控组件部署, 正在安装", "component", m.componentName)

		// 从模板文件渲染配置
		tplFile := fmt.Sprintf("./monitor/%s.yaml.tpl", m.componentName)
		rendered, err := util.RenderTemplateFromFile(tplFile, map[string]string{
			"ThanosURL": m.config.MonarchThanosURI,
		})
		if err != nil {
			slog.Error("渲染模板失败", "error", err, "file", tplFile)
			return fmt.Errorf("渲染模板失败: %v", err)
		}

		// 部署到Kubernetes
		err = kc.Apply(rendered, m.config.MonitorNamespace)
		if err != nil {
			slog.Error("应用模板到Kubernetes失败", "error", err, "file", tplFile)
			return fmt.Errorf("应用模板到Kubernetes失败: %v", err)
		}
	} else {
		slog.Info("监控组件已就绪", "component", m.componentName)
	}

	return nil
}

// checkHealth 检查监控系统的健康状态
// 防止循环依赖,暂时不使用monitor的checkHealth方法
func (m *Monitor) checkHealth() error {
	config := m.config

	// 创建一个HTTP客户端，设置超时时间
	httpClient := http.Client{
		Timeout: config.MonitorTimeout,
	}
	// 测试monarch request translator是否可用
	// 通过config.MONARCH_REQUEST_TRANSLATOR_URI/api/supported-kpis发送一个GET请求给monarch request translator, 返回status_code 200则成功
	req, err := http.NewRequest("GET", config.MonarchRequestTranslatorURI+"/api/supported-kpis", nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求 Monarch request translator 失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("translator 服务异常，状态码: " + fmt.Sprint(resp.StatusCode))
	}

	//测试monarch thanos是否可用
	req, err = http.NewRequest("GET", config.MonarchThanosURI+"/-/ready", nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	resp, err = httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求 Thanos 失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("Thanos 服务异常，状态码: " + fmt.Sprint(resp.StatusCode))
	}

	return nil
}
