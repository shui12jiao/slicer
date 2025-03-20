package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

type Slice struct {
	ID               string    `json:"id" yaml:"id" bson:"_id"`
	SST              int       `json:"sst" yaml:"sst"`
	SD               string    `json:"sd" yaml:"sd"`
	DefaultIndicator bool      `json:"default_indicator" yaml:"default_indicator"`
	Sessions         []Session `json:"session" yaml:"session"`
}

type Session struct {
	Name    string    `json:"name" yaml:"name"`
	Type    int       `json:"type" yaml:"type"`
	PCCRule []PCCRule `json:"pcc_rule" yaml:"pcc_rule"`
	AMBR    AMBR      `json:"ambr" yaml:"ambr"`
	QoS     QoS       `json:"qos" yaml:"qos"`
}

type PCCRule struct {
	Flows []Flow `json:"flow" yaml:"flow"`
	QoS   QoS    `json:"qos" yaml:"qos"`
}

type Flow struct {
	Direction   int    `json:"direction" yaml:"direction"`
	Description string `json:"description" yaml:"description"`
}

type AMBR struct {
	Uplink   BitRate `json:"uplink" yaml:"uplink"`
	Downlink BitRate `json:"downlink" yaml:"downlink"`
}

type BitRate struct {
	Value int `json:"value" yaml:"value"`
	Unit  int `json:"unit" yaml:"unit"`
}

type QoS struct {
	Index int  `json:"index" yaml:"index"`
	ARP   ARP  `json:"arp" yaml:"arp"`
	MBR   *MBR `json:"mbr,omitempty" yaml:"mbr,omitempty"`
	GBR   *GBR `json:"gbr,omitempty" yaml:"gbr,omitempty"`
}

type ARP struct {
	PriorityLevel           int `json:"priority_level" yaml:"priority_level"`
	PreEmptionCapability    int `json:"pre_emption_capability" yaml:"pre_emption_capability"`
	PreEmptionVulnerability int `json:"pre_emption_vulnerability" yaml:"pre_emption_vulnerability"`
}

type GBR = AMBR
type MBR = AMBR

func (s *Slice) ToYAML() ([]byte, error) {
	return yaml.Marshal(s)
}

func (s *Slice) FromYAML(data []byte) error {
	return yaml.Unmarshal(data, s)
}

func (s *Slice) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

func (s *Slice) FromJSON(data []byte) error {
	return json.Unmarshal(data, s)
}

func (s *Slice) SliceID() string {
	return fmt.Sprintf("%d-%s", s.SST, s.SD)
}

// Validate 校验 Slice 结构
func (s *Slice) Validate() error {
	var errs []error

	// 校验 SST 格式
	if s.SST < 1 || s.SST > 4 {
		errs = append(errs, errors.New("SST格式错误"))
	}

	// 校验 SD 格式
	if !regexp.MustCompile(`^[0-9A-Fa-f]{1,6}$`).MatchString(s.SD) {
		errs = append(errs, errors.New("SD格式错误"))
	}

	// 至少包含一个会话
	if len(s.Sessions) == 0 {
		errs = append(errs, errors.New("必须包含至少一个会话"))
	}

	// 递归校验所有会话及其子结构
	for i := range s.Sessions {
		if err := validateSession(&s.Sessions[i], i); err != nil {
			errs = append(errs, fmt.Errorf("会话[%d]校验失败：%w", i, err))
		}
	}

	return errors.Join(errs...)
}

// validateSession 校验单个会话结构
func validateSession(s *Session, idx int) error {
	var errs []error

	// 会话名称不能为空
	if strings.TrimSpace(s.Name) == "" {
		errs = append(errs, fmt.Errorf("会话名称不能为空"))
	}

	// 会话类型校验(1:IPv4, 2:IPv6, 3:IPv4v6)
	if s.Type < 1 || s.Type > 3 { //
		errs = append(errs, fmt.Errorf("会话类型取值错误：%d 不是有效的会话类型", s.Type))
	}

	// 校验 AMBR 单位参数（0:bps, 1:Kbps, 2:Mbps, 3:Gbps, 4:Tbps)
	if s.AMBR.Uplink.Unit < 0 || s.AMBR.Uplink.Unit > 4 {
		errs = append(errs, fmt.Errorf("AMBR单位参数无效"))
	}

	// 校验 QoS 参数
	if s.QoS.Index < 1 || s.QoS.Index > 86 {
		errs = append(errs, fmt.Errorf("QoS索引值超出范围"))
	}

	// 校验所有 PCC 规则
	for i := range s.PCCRule {
		if err := validatePCCRule(&s.PCCRule[i], idx, i); err != nil {
			errs = append(errs, fmt.Errorf("PCC规则[%d]校验失败：%w", i, err))
		}
	}

	return errors.Join(errs...)
}

// validatePCCRule 校验单个 PCC 规则
func validatePCCRule(p *PCCRule, sessionIdx, ruleIdx int) error {
	var errs []error

	// 校验每个流的方向参数（0:上行, 1:下行)
	for i, flow := range p.Flows {
		if flow.Direction < 0 || flow.Direction > 1 {
			errs = append(errs, fmt.Errorf("会话[%d] PCC规则[%d] - 流量[%d]方向参数错误（允许值：0-2）", sessionIdx, ruleIdx, i))
		}
	}

	// 校验 ARP 优先级（1-15）
	if p.QoS.ARP.PriorityLevel < 1 || p.QoS.ARP.PriorityLevel > 15 {
		errs = append(errs, fmt.Errorf("会话[%d] PCC规则[%d] - ARP优先级超出范围（允许值：0-15）", sessionIdx, ruleIdx))
	}

	// 当 QoS 索引为 5-9 时必须包含 GBR 参数? 出处?
	// if p.QoS.Index >= 5 && p.QoS.Index <= 9 && p.QoS.GBR == nil {
	// 	errs = append(errs, fmt.Errorf("会话[%d] PCC规则[%d] - GBR参数缺失（QoS索引5-9需要保障带宽）", sessionIdx, ruleIdx))
	// }

	return errors.Join(errs...)
}
