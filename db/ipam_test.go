package db

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"slicer/util"

	"github.com/metal-stack/go-ipam"
	"github.com/stretchr/testify/require"
)

// 测试配置模板
var testConfig = &util.Config{
	IPAMConfig: util.IPAMConfig{
		IPAMTimeout:         255,
		N3Network:           "10.10.3.0/24",
		N4Network:           "10.10.4.0/24",
		SessionNetwork:      "10.32.0.0/11",
		SessionSubnetLength: 16,
	},
}

// 测试初始化
func newTestIPAM(t *testing.T, config *util.Config) *IPAM {
	// 初始化父前缀
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	storage := ipam.NewMemory(ctx)
	ipamer := ipam.NewWithStorage(storage)

	// 检查SessionSubnetLength是否合法
	parentLength, err := strconv.Atoi(strings.Split(config.SessionNetwork, "/")[1])
	require.NoError(t, err, "会话子网长度不合法")
	// 检查N3和N4网络是否合法
	require.Condition(t, func() bool {
		return int(config.SessionSubnetLength) >= parentLength && config.SessionSubnetLength <= 32
	}, "会话子网长度不合法")

	// 初始化三个网络的前缀
	for _, p := range []string{config.N3Network, config.N4Network, config.SessionNetwork} {
		_, err := ipamer.NewPrefix(ctx, p)
		require.NoError(t, err, fmt.Sprintf("初始化%s网络失败: %v", p, err))
	}

	return &IPAM{
		config: config,
		ipam:   ipamer,
	}
}

// N3 和 N4 地址分配测试相同
func TestAllocateAndReleaseAddr(t *testing.T) {
	i := newTestIPAM(t, testConfig)

	// 分配 N4 地址
	ip, err := i.AllocateN4Addr() // 结果ex: 10.10.4.1/24
	require.NoError(t, err)
	require.NotEmpty(t, ip)
	// 验证 IP 地址格式
	_, ipnet, err := net.ParseCIDR(ip)
	require.NoError(t, err)
	// 验证 IP 地址属于 N4 网络
	_, n4Net, err := net.ParseCIDR(i.config.N4Network)
	require.NoError(t, err)
	require.True(t, n4Net.Contains(ipnet.IP), "IP 地址不在 N4 网络范围内")
	require.Equal(t, ipnet.String(), n4Net.String(), "IP 地址不在 N4 网络范围内")

	// 释放 N4 地址
	err = i.ReleaseN4Addr(ip)
	require.NoError(t, err)
}

func TestAllocateAndReleaseSessionSubnet(t *testing.T) {
	i := newTestIPAM(t, testConfig)

	// 分配 Session 子网
	subnet, err := i.AllocateSessionSubnet()
	require.NoError(t, err)
	require.NotEmpty(t, subnet)

	// 验证 CIDR 格式
	_, ipnet, err := net.ParseCIDR(subnet)
	require.NoError(t, err)
	require.Equal(t, ipnet.String(), subnet)

	// 释放 Session 子网
	err = i.ReleaseSessionSubnet(subnet)
	require.NoError(t, err)
}

func TestReleaseSessionSubnets(t *testing.T) {
	i := newTestIPAM(t, testConfig)

	// 批量分配和释放子网
	s1, err1 := i.AllocateSessionSubnet()
	require.NoError(t, err1)
	s2, err2 := i.AllocateSessionSubnet()
	require.NoError(t, err2)

	err := i.ReleaseSessionSubnets([]string{s1, s2})
	require.NoError(t, err)
}
