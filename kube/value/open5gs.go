package value

import (
	"log/slog"

	"github.com/mitchellh/mapstructure"
)

type Open5gs struct {
	AMF   *AMF          `json:"amf,omitempty" yaml:"amf,omitempty" mapstructure:"amf,omitempty"`
	AUSF  *AUSF         `json:"ausf,omitempty" yaml:"ausf,omitempty" mapstructure:"ausf,omitempty"`
	BSF   *BSF          `json:"bsf,omitempty" yaml:"bsf,omitempty" mapstructure:"bsf,omitempty"`
	NRF   *NRF          `json:"nrf,omitempty" yaml:"nrf,omitempty" mapstructure:"nrf,omitempty"`
	NSSF  *NSSF         `json:"nssf,omitempty" yaml:"nssf,omitempty" mapstructure:"nssf,omitempty"`
	PCF   *PCF          `json:"pcf,omitempty" yaml:"pcf,omitempty" mapstructure:"pcf,omitempty"`
	SCP   *SCP          `json:"scp,omitempty" yaml:"scp,omitempty" mapstructure:"scp,omitempty"`
	SMF   *SMF          `json:"smf,omitempty" yaml:"smf,omitempty" mapstructure:"smf,omitempty"`
	UDM   *UDM          `json:"udm,omitempty" yaml:"udm,omitempty" mapstructure:"udm,omitempty"`
	UDR   *UDR          `json:"udr,omitempty" yaml:"udr,omitempty" mapstructure:"udr,omitempty"`
	UPF   *UPF          `json:"upf,omitempty" yaml:"upf,omitempty" mapstructure:"upf,omitempty"`
	WebUI *Open5GSWebUI `json:"webui,omitempty" yaml:"webui,omitempty" mapstructure:"webui,omitempty"`
}

func (value Open5gs) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	config := mapstructure.DecoderConfig{
		Result:           &result,
		WeaklyTypedInput: true, // 允许弱类型输入
		TagName:          "mapstructure",
	}

	decoder, err := mapstructure.NewDecoder(&config)
	if err != nil {
		slog.Error("创建mapstructure解码器失败", "error", err)
		return nil
	}

	if err := decoder.Decode(value); err != nil {
		slog.Error("解码HelmChartValues失败", "error", err)
		return nil
	}

	return result
}
