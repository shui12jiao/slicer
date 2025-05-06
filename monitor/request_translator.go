package monitor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slicer/model"
)

// 获取支持的KPIs
// GET /api/supported-kpis
type getSupportedKpisResponse struct {
	Response
	SupportedKpis []model.SupportedKpi `json:"supported_kpis"`
}

// 联系Monarch request translator获取支持的KPIs
func (m *Monitor) GetSupportedKpis() ([]model.SupportedKpi, error) {
	httpClient := http.Client{
		Timeout: m.config.MonitorTimeout,
	}
	req, err := http.NewRequest("GET", m.config.MonarchRequestTranslatorURI+"/api/supported-kpis", nil)
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

	var supportedKpis getSupportedKpisResponse
	if err := json.NewDecoder(resp.Body).Decode(&supportedKpis); err != nil {
		return nil, fmt.Errorf("解析Monarch response失败: %v", err)
	}
	if supportedKpis.Status != "success" {
		return nil, errors.New("translator 服务异常，状态: " + supportedKpis.Status)
	}

	return supportedKpis.SupportedKpis, nil
}

// 提交监控请求
// POST /api/monitoring-requests

// type submitMonitoringRequest = model.Monitor

type submitMonitoringResponse struct {
	Response
	RequestID string `json:"request_id"`
}

func (m *Monitor) SubmitMonitoring(monitor model.Monitor) (model.Monitor, error) {
	httpClient := http.Client{
		Timeout: m.config.MonitorTimeout,
	}

	// 序列化请求体
	reqBody, err := json.Marshal(monitor)
	if err != nil {
		return monitor, fmt.Errorf("序列化请求体失败: %v", err)
	}
	req, err := http.NewRequest("POST", m.config.MonarchRequestTranslatorURI+"/api/monitoring-requests", bytes.NewBuffer(reqBody))
	if err != nil {
		return monitor, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := httpClient.Do(req)
	if err != nil {
		return monitor, fmt.Errorf("请求 Monarch request translator 失败: %v", err)
	}
	defer resp.Body.Close()

	// 解析响应
	// if resp.StatusCode != http.StatusOK {
	// 	return monitor, errors.New("translator 服务异常，状态码: " + fmt.Sprint(resp.StatusCode))
	// }

	var submitMonitoringResponse submitMonitoringResponse
	if err := json.NewDecoder(resp.Body).Decode(&submitMonitoringResponse); err != nil {
		return monitor, fmt.Errorf("解析Monarch response失败: %v", err)
	}
	if submitMonitoringResponse.Status != "success" || resp.StatusCode != http.StatusOK {
		return monitor, fmt.Errorf("translator 服务异常，消息: %s, 状态码: %d", submitMonitoringResponse.Message, resp.StatusCode)
	}

	monitor.RequestID = submitMonitoringResponse.RequestID
	return monitor, nil
}

// 删除监控请求
// DELETE /api/monitoring-requests/delete/<request_id>

type deleteMonitoringResponse = Response

func (m *Monitor) DeleteMonitoring(requestID string) error {
	httpClient := http.Client{
		Timeout: m.config.MonitorTimeout,
	}
	req, err := http.NewRequest("DELETE", m.config.MonarchRequestTranslatorURI+"/api/monitoring-requests/delete/"+requestID, nil)
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

	var deleteMonitoringResponse deleteMonitoringResponse
	if err := json.NewDecoder(resp.Body).Decode(&deleteMonitoringResponse); err != nil {
		return fmt.Errorf("解析Monarch response失败: %v", err)
	}
	if deleteMonitoringResponse.Status != "success" {
		return errors.New("translator 服务异常，%v" + deleteMonitoringResponse.Message)
	}

	return nil
}
