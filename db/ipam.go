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

func NewIPAM(config util.Config) (*IPAM, error) {
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

	return &IPAM{
		config: config,
		ipam:   ipamer,
	}, nil

}

func (i *IPAM) AllocateN3Addr() (string, error) {
	return i.allocateIP(i.config.N3Network)

}
func (i *IPAM) AllocateN4Addr() (string, error) {
	return i.allocateIP(i.config.N4Network)
}

// 分配IP
func (i *IPAM) allocateIP(network string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(i.config.IPAMTimeout)*time.Second)
	defer cancel()

	alloc, err := i.ipam.AcquireIP(ctx, network)
	if err != nil {
		return "", fmt.Errorf("分配 %s 地址失败: %w", network, err)
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
		return "", fmt.Errorf("分配会话子网失败: %w", err)
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
		return fmt.Errorf("无效的IP地址格式: %s", addr)
	}

	// 执行释放
	if _, err := i.ipam.ReleaseIP(ctx, &ipam.IP{
		IP:           netip.MustParseAddr(ip.String()),
		ParentPrefix: expectedPrefix,
	}); err != nil {
		return fmt.Errorf("释放失败: %w", err)
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
