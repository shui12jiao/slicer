package render

import (
	"testing"

	"slicer/model"
	"slicer/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// 统一测试用例数据
var testSlice = model.SliceAndAddress{
	Slice: model.Slice{
		SST:              1,
		SD:               "000001",
		DefaultIndicator: true,
		Sessions:         []model.Session{{Name: "internet"}, {Name: "streaming"}},
	},
	AddressValue: model.AddressValue{
		SessionSubnets: []string{"10.40.0.0/16", "10.41.0.0/16"},
		UPFN3Addr:      "10.10.3.1",
		UPFN4Addr:      "10.10.4.1",
		SMFN3Addr:      "10.10.3.2",
		SMFN4Addr:      "10.10.4.2",
	},
}

// 验证生成的YAML结构
func TestRenderTemplatesWithTestSlice(t *testing.T) {
	config := util.Config{
		TemplatePath: "./template",
		// 假设模板文件已放在测试目录
	}
	r := NewRender(config)

	// 生成配置内容
	contents, err := r.RenderSlice(testSlice)
	require.NoError(t, err, "生成配置失败")
	require.Len(t, contents, 5, "应生成5个配置文件")

	// 定义各文件的验证逻辑
	testCases := []struct {
		name     string
		content  []byte
		validate func(t *testing.T, data []byte)
	}{
		{
			name:    "SMF ConfigMap",
			content: contents[0],
			validate: func(t *testing.T, data []byte) {
				var cm struct {
					Metadata struct {
						Name   string
						Labels map[string]string
					}
					Data struct {
						SMFCfgYAML string `yaml:"smfcfg.yaml"`
					}
				}
				require.NoError(t, yaml.Unmarshal(data, &cm))

				// 验证元数据
				assert.Equal(t, "smf1-000001-configmap", cm.Metadata.Name)
				assert.Equal(t, "smf1-000001", cm.Metadata.Labels["name"])

				// 验证配置内容
				assert.Contains(t, cm.Data.SMFCfgYAML, "- subnet: 10.40.0.0/16", "gateway: 10.40.0.1/16")
				assert.Contains(t, cm.Data.SMFCfgYAML, `
  info:
    - s_nssai:
      - sst: 1
        sd: 000001
        dnn:
         - internet
         - streaming`)
			},
		},
		{
			name:    "SMF Deployment",
			content: contents[1],
			validate: func(t *testing.T, data []byte) {
				var dep struct {
					Spec struct {
						Template struct {
							Metadata struct {
								Labels      map[string]string
								Annotations map[string]string
							}
						}
					}
				}
				require.NoError(t, yaml.Unmarshal(data, &dep))

				assert.Equal(t, "smf1-000001", dep.Spec.Template.Metadata.Labels["name"])
				assert.Contains(t, dep.Spec.Template.Metadata.Annotations["k8s.v1.cni.cncf.io/networks"],
					`"n4", "ips": [ "10.10.4.2" ]`,
					`"n3", "ips": [ "10.10.3.2" ]`)
			},
		},
		{
			name:    "UPF ConfigMap",
			content: contents[3],
			validate: func(t *testing.T, data []byte) {
				var cm struct {
					Data struct {
						UPFCfgYAML string `yaml:"upfcfg.yaml"`
						WrapperSh  string `yaml:"wrapper.sh"`
					}
				}
				require.NoError(t, yaml.Unmarshal(data, &cm))

				assert.Contains(t, cm.Data.UPFCfgYAML, `
  session:
    - subnet: 10.40.0.0/16
      gateway: 10.40.0.1/16
      dnn: internet
    - subnet: 10.41.0.0/16
      gateway: 10.41.0.1/16
      dnn: streaming`)
				assert.Contains(t, cm.Data.WrapperSh, `
ip tuntap add name ogstun0 mode tun;
ip addr add 10.40.0.1/16 dev ogstun0;
ip link set ogstun0 up;
iptables -t nat -A POSTROUTING -s 10.40.0.0/16 ! -o ogstun0 -j MASQUERADE;
ip tuntap add name ogstun1 mode tun;
ip addr add 10.41.0.1/16 dev ogstun1;
ip link set ogstun1 up;
iptables -t nat -A POSTROUTING -s 10.41.0.0/16 ! -o ogstun1 -j MASQUERADE;`)
			},
		},
	}

	// 执行所有子测试
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.validate(t, tc.content)
		})
	}
}
