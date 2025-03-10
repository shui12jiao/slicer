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
	config util.Config
	ipam   ipam.Ipamer
}

func NewIPAM(config util.Config) *IPAM {
	// 初始化父前缀
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
		panic(err)
	}

	ipamer := ipam.NewWithStorage(storage)

	// 检查SessionSubnetLength是否合法
	parentLength, err := strconv.Atoi(strings.Split(config.SessionNetwork, "/")[1])
	if err != nil || int(config.SessionSubnetLength) < parentLength || config.SessionSubnetLength > 32 {
		panic(fmt.Sprintf("invalid session subnet length: %d", config.SessionSubnetLength))
	}

	// 初始化三个网络的前缀
	prefixes := []string{config.N3Network, config.N4Network, config.SessionNetwork}
	for _, p := range prefixes {
		_, err := ipamer.NewPrefix(ctx, p)
		if err != nil {
			panic(fmt.Sprintf("failed to initialize prefix %s: %v", p, err))
		}
	}

	return &IPAM{
		config: config,
		ipam:   ipamer,
	}

}

func (i *IPAM) AllocateN3Addr() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(i.config.IPAMTimeout)*time.Second)
	defer cancel()

	// 调用 ipam.AcquireIP 分配单个 IP 地址，参数为 N3Network 字符串
	alloc, err := i.ipam.AcquireIP(ctx, i.config.N3Network)
	if err != nil {
		return "", err
	}

	return alloc.IP.String(), nil
}
func (i *IPAM) AllocateN4Addr() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	alloc, err := i.ipam.AcquireIP(ctx, i.config.N4Network)
	if err != nil {
		return "", err
	}

	return alloc.IP.String(), nil
}

// AllocateSessionSubnet 从 SessionNetwork 范围内分配一个子网
func (i *IPAM) AllocateSessionSubnet() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(i.config.IPAMTimeout)*time.Second)
	defer cancel()

	// 从SessionNetwork分配子网，子网掩码由SessionSubnetMask指定
	subnet, err := i.ipam.AcquireChildPrefix(ctx, i.config.SessionNetwork, i.config.SessionSubnetLength)
	if err != nil {
		return "", fmt.Errorf("failed to allocate session subnet: %w", err)
	}

	return subnet.Cidr, nil
}

// ReleaseN3Addr 释放一个 N3 地址
func (i *IPAM) ReleaseN3Addr(addr string) error {
	return i.releaseIP(addr, i.config.N3Network)
}

// ReleaseN4Addr 释放一个 N4 地址
func (i *IPAM) ReleaseN4Addr(addr string) error {
	return i.releaseIP(addr, i.config.N4Network)
}

func (i *IPAM) releaseIP(addr, expectedPrefix string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(i.config.IPAMTimeout)*time.Second)
	defer cancel()

	// 验证 IP 有效性
	ip := net.ParseIP(addr)
	if ip == nil {
		return fmt.Errorf("invalid IP address format: %s", addr)
	}

	// 获取 IP 所属前缀
	prefix, err := i.ipam.PrefixFrom(ctx, ip.String())
	if err != nil {
		return fmt.Errorf("failed to locate IP prefix: %w", err)
	}

	// 验证前缀匹配性
	if prefix.Cidr != expectedPrefix {
		return fmt.Errorf("IP %s does not belong to expected prefix %s", addr, expectedPrefix)
	}

	// 执行释放
	if _, err := i.ipam.ReleaseIP(ctx, &ipam.IP{
		IP:           netip.MustParseAddr(ip.String()),
		ParentPrefix: prefix.Cidr,
	}); err != nil {
		return fmt.Errorf("release failed: %w", err)
	}

	return nil
}

// ReleaseSessionSubnet 释放一个子网
func (i *IPAM) ReleaseSessionSubnet(subnet string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(i.config.IPAMTimeout)*time.Second)
	defer cancel()

	// 验证子网格式
	_, _, err := net.ParseCIDR(subnet)
	if err != nil {
		return fmt.Errorf("invalid subnet format: %w", err)
	}

	// 获取父前缀
	prefix, err := i.ipam.PrefixFrom(ctx, subnet)
	if err != nil {
		return fmt.Errorf("subnet not found: %w", err)
	}

	// 检查是否属于 SessionNetwork
	if !strings.HasPrefix(prefix.Cidr, i.config.SessionNetwork) {
		return fmt.Errorf("subnet %s not under parent %s", subnet, i.config.SessionNetwork)
	}

	// 检查子网使用状态
	if usage := prefix.Usage(); usage.AcquiredIPs > 0 {
		return fmt.Errorf("subnet %s still has %d active IPs", subnet, usage.AcquiredIPs)
	}

	// 执行释放
	if err := i.ipam.ReleaseChildPrefix(ctx, prefix); err != nil {
		return fmt.Errorf("subnet release failed: %w", err)
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
