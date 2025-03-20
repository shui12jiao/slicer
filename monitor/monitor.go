package monitor

import (
	"errors"
	"fmt"
	"net/http"
	"slicer/util"
	"time"
)

// 与Monarch监控系统沟通客户端
type Monitor struct {
	config util.Config
}

func NewMonitor(config util.Config) (*Monitor, error) {
	httpClient := http.Client{
		Timeout: time.Duration(config.MonitorTimeout) * time.Second,
	}
	// 测试monarch request translator是否可用
	// 通过config.MONARCH_REQUEST_TRANSLATOR_URI/api/supported-kpis发送一个GET请求给monarch request translator, 返回status_code 200则成功
	req, err := http.NewRequest("GET", config.MonarchRequestTranslatorURI+"/api/supported-kpis", nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 Monarch request translator 失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("translator 服务异常，状态码: " + fmt.Sprint(resp.StatusCode))
	}

	//测试monarch thanos是否可用
	req, err = http.NewRequest("GET", config.MonarchThanosURI+"/-/ready", nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	resp, err = httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 Thanos 失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Thanos 服务异常，状态码: " + fmt.Sprint(resp.StatusCode))
	}

	return &Monitor{config: config}, nil
}
