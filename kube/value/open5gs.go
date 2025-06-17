package value

import (
	"log/slog"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

type Common struct {
	AMF   *AMF          `json:"amf,omitempty" yaml:"amf,omitempty" mapstructure:"amf,omitempty"`
	AUSF  *AUSF         `json:"ausf,omitempty" yaml:"ausf,omitempty" mapstructure:"ausf,omitempty"`
	BSF   *BSF          `json:"bsf,omitempty" yaml:"bsf,omitempty" mapstructure:"bsf,omitempty"`
	NRF   *NRF          `json:"nrf,omitempty" yaml:"nrf,omitempty" mapstructure:"nrf,omitempty"`
	NSSF  *NSSF         `json:"nssf,omitempty" yaml:"nssf,omitempty" mapstructure:"nssf,omitempty"`
	PCF   *PCF          `json:"pcf,omitempty" yaml:"pcf,omitempty" mapstructure:"pcf,omitempty"`
	SCP   *SCP          `json:"scp,omitempty" yaml:"scp,omitempty" mapstructure:"scp,omitempty"`
	UDM   *UDM          `json:"udm,omitempty" yaml:"udm,omitempty" mapstructure:"udm,omitempty"`
	UDR   *UDR          `json:"udr,omitempty" yaml:"udr,omitempty" mapstructure:"udr,omitempty"`
	WebUI *Open5GSWebUI `json:"webui,omitempty" yaml:"webui,omitempty" mapstructure:"webui,omitempty"`
}

func (c Common) ToMap() map[string]interface{} {
	return ToMap(c)
}

type Slice struct {
	SMF *SMF `json:"smf,omitempty" yaml:"smf,omitempty" mapstructure:"smf,omitempty"`
	UPF *UPF `json:"upf,omitempty" yaml:"upf,omitempty" mapstructure:"upf,omitempty"`
}

func (s Slice) ToMap() map[string]interface{} {
	return ToMap(s)
}

func ToMap(input any) map[string]interface{} {
	result := make(map[string]interface{})

	config := mapstructure.DecoderConfig{
		Result:           &result,
		WeaklyTypedInput: true, // 允许弱类型输入
		TagName:          "mapstructure",
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
				// 递归解引用所有指针类型
				val := reflect.ValueOf(data)
				for val.Kind() == reflect.Ptr && !val.IsNil() {
					val = val.Elem()
				}
				return val.Interface(), nil
			},
			// 保留其他内置钩子（如时间/切片转换）
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	}

	decoder, err := mapstructure.NewDecoder(&config)
	if err != nil {
		slog.Error("创建mapstructure解码器失败", "error", err)
		return nil
	}

	if err := decoder.Decode(input); err != nil {
		slog.Error("解码失败", "error", err)
		return nil
	}

	return result
}
