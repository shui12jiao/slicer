package value

import "encoding/json"

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
	data, err := json.Marshal(input)
	if err != nil {
		return nil
	}
	var result map[string]interface{}
	if err = json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}
