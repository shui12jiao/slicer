package model

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v2"
)

type Slice struct {
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

func (s *Slice) ID() string {
	return fmt.Sprintf("%d-%s.json", s.SST, s.SD)
}
