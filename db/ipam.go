package db

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"slicer/util"
	"strconv"
	"strings"
	"time"

	"github.com/metal-stack/go-ipam"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type IPAM struct {
	config *util.Config
	ipam   ipam.Ipamer
}

func NewIPAM(config *util.Config) (*IPAM, error) {
	// 初始化父前缀
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 创建一个新的MongoDB存储实例
	mongoConfig := ipam.MongoConfig{
		DatabaseName: config.MongoDBName,
		MongoClientOptions: options.Client().
			ApplyURI(config.MongoURI).
			SetServerSelectionTimeout(time.Duration(config.MongoTimeout) * time.Second),
	}

	storage, err := ipam.NewMongo(ctx, mongoConfig)
	if err != nil {
		panic(fmt.Sprintf("创建MongoDB存储实例失败: %v", err))
	}

	ipamer := ipam.NewWithStorage(storage)

	// 检查SessionSubnetLength是否合法
	parentLength, err := strconv.Atoi(strings.Split(config.SessionNetwork, "/")[1])
	if err != nil || int(config.SessionSubnetLength) < parentLength || config.SessionSubnetLength > 32 {
		panic(fmt.Sprintf("会话子网长度不合法: %d", config.SessionSubnetLength))
	}

	// 初始化三个网络的前缀
	for _, p := range []string{config.N3Network, config.N4Network, config.SessionNetwork} {
		_, err := ipamer.NewPrefix(ctx, p)
		if err != nil && !strings.Contains(err.Error(), fmt.Sprintf("overlaps %s", p)) {
			return nil, fmt.Errorf("初始化%s网络失败: %w", p, err)
		}
	}
	// 保留前两个IP地址
	ipam := &IPAM{
		config: config,
		ipam:   ipamer,
	}
	if err := ipam.reserveFirstIPs(ctx, []string{config.N3Network, config.N4Network}); err != nil {
		return nil, fmt.Errorf("保留IP失败: %w", err)
	}
	return ipam, nil
}

func (i *IPAM) reserveFirstIPs(ctx context.Context, prefixes []string) error {
	for _, prefixStr := range prefixes {
		prefix, err := netip.ParsePrefix(prefixStr)
		if err != nil {
			return fmt.Errorf("解析前缀失败 %s: %w", prefixStr, err)
		}

		// 获取 prefix 的起始 IP，然后偏移 +1 和 +2
		networkIP := prefix.Addr()
		reservedIPs := []netip.Addr{}

		nextIP := networkIP.Next() // .1
		if nextIP.IsValid() && prefix.Contains(nextIP) {
			reservedIPs = append(reservedIPs, nextIP)
		}

		nextNextIP := nextIP.Next() // .2
		if nextNextIP.IsValid() && prefix.Contains(nextNextIP) {
			reservedIPs = append(reservedIPs, nextNextIP)
		}

		for _, ip := range reservedIPs {
			_, err := i.ipam.AcquireSpecificIP(ctx, prefixStr, ip.String())
			if err != nil && !strings.Contains(err.Error(), "already allocated") {
				return fmt.Errorf("保留IP失败 %s: %w", ip, err)
			}
		}
	}

	return nil
}

func (i *IPAM) AllocateN3Addr() (string, error) {
	ip, err := i.allocateIP(i.config.N3Network)
	if err != nil {
		return "", fmt.Errorf("分配N3地址失败: %w", err)
	}
	return ip.String(), nil
}
func (i *IPAM) AllocateN4Addr() (string, error) {
	ip, err := i.allocateIP(i.config.N4Network)
	if err != nil {
		return "", fmt.Errorf("分配N4地址失败: %w", err)
	}
	return ip.String(), nil
}

// 将ipam库IP类型转换为netip.Prefix类型
func toNetipPrefix(ip *ipam.IP) (netip.Prefix, error) {
	// 验证 IP 有效性
	if ip == nil {
		return netip.Prefix{}, fmt.Errorf("无效的IP地址")
	}

	// 解析父前缀获取掩码长度
	parentPrefix, err := netip.ParsePrefix(ip.ParentPrefix)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("解析前缀失败: %w", err)
	}

	return netip.PrefixFrom(ip.IP, parentPrefix.Bits()), nil
}

// 分配IP
func (i *IPAM) allocateIP(network string) (netip.Prefix, error) {
	ctx, cancel := context.WithTimeout(context.Background(), i.config.IPAMTimeout)
	defer cancel()

	alloc, err := i.ipam.AcquireIP(ctx, network)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("分配 %s 地址失败: %w", network, err)
	}
	// 将分配的IP转换为netip.Prefix类型
	return toNetipPrefix(alloc)
}

// AllocateSessionSubnet 从 SessionNetwork 范围内分配一个子网
func (i *IPAM) AllocateSessionSubnet() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), i.config.IPAMTimeout)
	defer cancel()

	// 从SessionNetwork分配子网，子网掩码由SessionSubnetMask指定
	subnet, err := i.ipam.AcquireChildPrefix(ctx, i.config.SessionNetwork, i.config.SessionSubnetLength)
	if err != nil {
		return "", fmt.Errorf("分配会话子网失败: %w", err)
	}

	return subnet.Cidr, nil
}

// ReleaseN3Addr 释放一个 N3 地址
func (i *IPAM) ReleaseN3Addr(addr string) error {
	prefix, err := netip.ParsePrefix(addr)
	if err != nil {
		return fmt.Errorf("无效的IP地址格式: %s", addr)
	}
	// 验证prefix属于N3网络
	_, n3Net, _ := net.ParseCIDR(i.config.N3Network)
	if !n3Net.Contains(prefix.Addr().AsSlice()) {
		return fmt.Errorf("IP地址不在N3网络范围内: %s", addr)
	}
	// 执行释放
	return i.releaseIP(prefix)
}

// ReleaseN4Addr 释放一个 N4 地址
func (i *IPAM) ReleaseN4Addr(addr string) error {
	prefix, err := netip.ParsePrefix(addr)
	if err != nil {
		return fmt.Errorf("无效的IP地址格式: %s", addr)
	}
	// 验证prefix属于N4网络
	_, n4Net, _ := net.ParseCIDR(i.config.N4Network)
	if !n4Net.Contains(prefix.Addr().AsSlice()) {
		return fmt.Errorf("IP地址不在N4网络范围内: %s", addr)
	}
	// 执行释放
	return i.releaseIP(prefix)
}

func (i *IPAM) releaseIP(ip netip.Prefix) error {
	ctx, cancel := context.WithTimeout(context.Background(), i.config.IPAMTimeout)
	defer cancel()

	// 执行释放
	if _, err := i.ipam.ReleaseIP(ctx, &ipam.IP{
		IP:           ip.Addr(),
		ParentPrefix: ip.Masked().String(),
	}); err != nil {
		return fmt.Errorf("释放失败: %w", err)
	}

	return nil
}

// ReleaseSessionSubnet 释放一个子网
func (i *IPAM) ReleaseSessionSubnet(subnet string) error {
	ctx, cancel := context.WithTimeout(context.Background(), i.config.IPAMTimeout)
	defer cancel()

	// 验证子网格式
	_, _, err := net.ParseCIDR(subnet)
	if err != nil {
		return fmt.Errorf("子网格式无效: %w", err)
	}

	// 获取子网前缀
	prefix, err := i.ipam.PrefixFrom(ctx, subnet)
	if err != nil {
		return fmt.Errorf("找不到子网: %w", err)
	}

	// 执行释放
	if err := i.ipam.ReleaseChildPrefix(ctx, prefix); err != nil {
		return fmt.Errorf("子网释放失败: %w", err)
	}

	return nil
}

func (i *IPAM) ReleaseSessionSubnets(subnets []string) error {
	for _, subnet := range subnets {
		if err := i.ReleaseSessionSubnet(subnet); err != nil {
			return err
		}
	}
	return nil
}
