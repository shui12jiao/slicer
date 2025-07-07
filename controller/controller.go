package controller

import (
	"context"
	"errors"
	"log/slog"
	"slicer/db"
	"slicer/kube"
	"slicer/util"
	"sync"
	"time"
)

type Controller interface {
	// 运行
	Start() // 异步运行
	Stop()
	IsRunning() bool // 控制器是否在运行

	// 立刻执行控制
	Trigger(slices []string) error // 触发控制, 如果不传入切片ID, 则控制器会使用当前所有切片

	// 频率
	SetFrequency(duration time.Duration) // 设置控制频率
	GetFrequency() time.Duration         // 获取控制频率

	// 切片
	AddSlice(sliceID string)
	RemoveSlice(sliceID string)
	ListSlices() []string

	// 策略
	SetStrategy(strategy Strategy)
	GetStrategy() Strategy
	RegisterStrategy(strategy ...Strategy)
	UnregisterStrategy(strategy ...Strategy)
	ListStrategy() []Strategy
	GetStrategyByName(name string) Strategy
}

type BasicController struct {
	// 互斥锁
	mu sync.Mutex // 保护running, frequency以及切片和策略相关资源
	// 控制器的上下文
	ctx context.Context
	// 控制器的取消函数
	cancel context.CancelFunc
	// 用于等待 run Goroutine 退出
	wg sync.WaitGroup
	// 立刻触发指定slice控制
	trigger chan []string

	// config
	config *util.Config
	// 存储
	store db.Store
	// 控制器的配置
	kclient *kube.KubeClient

	// 运行状态
	running bool
	// 控制频率
	frequency time.Duration

	// 切片列表
	slices []string
	// 策略列表
	strategies []Strategy
	// 策略
	strategy Strategy
}

// NewBasicController 创建一个新的控制器
// 注册传入的所有strategy, 并将第一个strategy设置为默认策略, 若不传入则strategy为nil
func NewBasicController(config *util.Config, store db.Store, kclient *kube.KubeClient, strategy ...Strategy) Controller {
	// 初始创建一个已取消的上下文，作为占位符
	initialCtx, initialCancel := context.WithCancel(context.Background())
	initialCancel() // 确保这个占位符上下文是已取消状态

	c := &BasicController{
		running:   false,
		frequency: config.Frequency,
		ctx:       initialCtx, // 使用初始已取消的上下文
		cancel:    initialCancel,
		trigger:   make(chan []string, 1),
		slices:    []string{},
		config:    config,
		store:     store,
		kclient:   kclient,
		strategy: func() Strategy {
			if len(strategy) > 0 {
				return strategy[0]
			}
			return nil
		}(),
	}
	c.RegisterStrategy(strategy...) // 注册策略
	return c
}

// 运行相关
func (c *BasicController) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		slog.Warn("控制器已在运行中, 无需重复启动")
		return
	}

	// 每次启动都创建新的上下文
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.running = true
	c.wg.Add(1) // 增加一个等待的 Goroutine

	go c.run() // 启动控制器的运行逻辑
	slog.Info("控制器启动中...")
}

func (c *BasicController) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		slog.Info("控制器未运行，无需停止")
		return
	}

	slog.Info("控制器停止中...")
	c.cancel()    // 发送取消信号给 run() Goroutine
	c.mu.Unlock() // cancel本身是线程安全的，这里防止并发时候未知状态

	// 等待 run() Goroutine 完全退出
	c.wg.Wait()

	// Goroutine 退出后，安全地更新状态
	c.mu.Lock() // 重新获取锁保护 running 状态
	c.running = false
	c.mu.Unlock()

	slog.Info("控制器已停止")
}

func (c *BasicController) run() {
	defer c.wg.Done() // Goroutine 退出时通知 WaitGroup 完成

	ticker := time.NewTicker(c.frequency)
	defer ticker.Stop()

	slog.Info("控制器已启动", "频率", c.frequency)

	for {
		var slices []string

		select {
		// 停止
		case <-c.ctx.Done():
			slog.Info("控制器核心逻辑 Goroutine 收到停止信号，正在退出...")
			return // 退出循环和 Goroutine
		// 立刻触发控制
		case slices = <-c.trigger:
		// 定时触发控制，触发所有切片
		case <-ticker.C:
			slices = c.ListSlices()
		}

		// 执行控制逻辑
		for _, sliceID := range slices {
			err := c.control(sliceID)
			if err != nil {
				slog.Error("控制失败, 跳过", "sliceID", sliceID, "err", err)
				continue
			}
		}
	}
}

func (c *BasicController) control(sliceID string) error {
	// 获取切片信息
	slice, err := c.store.GetSliceBySliceID(sliceID)
	if err != nil {
		slog.Error("获取切片信息失败", "sliceID", sliceID, "err", err)
		return err
	}

	// 核心控制逻辑
	// 调用策略执行Reconcile, 生成新的Deploy
	newDeploy, err := c.strategy.Reconcile(sliceID, slice.Deploy, slice.SLA)
	if err != nil {
		slog.Error("生成新Deploy失败", "sliceID", sliceID, "err", err)
		return err
	}

	// 应用新的Deploy
	err = c.kclient.Deploy(newDeploy,
		sliceID,
		c.config.HelmReleasePrefix+sliceID+"-upf",
		c.config.Namespace)
	if err != nil {
		slog.Error("应用Deploy失败", "sliceID", sliceID, "err", err)
		return err
	}

	// 更新Slice
	slice.Deploy = newDeploy // 更新Slice的Deploy
	// 更新SliceProfile存储
	err = c.store.UpdateSlice(slice)
	if err != nil {
		slog.Error("更新Slice失败", "sliceID", sliceID, "err", err)
		return err
	}

	// 完成
	slog.Info("控制完成", "sliceID", sliceID, "newDeploy", newDeploy)
	return nil
}

func (c *BasicController) Trigger(slices []string) error {
	if !c.IsRunning() {
		return errors.New("控制器未运行, 无法触发控制")
	}

	if len(slices) == 0 {
		slices = c.ListSlices() // 测试方便，这里如果没有传入切片ID，则使用当前所有切片
	}
	slog.Info("触发控制", "切片ID", slices)
	c.trigger <- slices // 传入指定的切片ID
	return nil
}

func (c *BasicController) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

// 切片相关
func (c *BasicController) AddSlice(sliceID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range c.slices {
		if id == sliceID {
			return
		}
	}
	c.slices = append(c.slices, sliceID)
}
func (c *BasicController) RemoveSlice(sliceID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, id := range c.slices {
		if id == sliceID {
			c.slices = append(c.slices[:i], c.slices[i+1:]...)
			return
		}
	}
}

func (c *BasicController) ListSlices() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.slices
}

// 频率相关
func (c *BasicController) SetFrequency(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.frequency = duration

	// 重启
	if c.running {
		c.Stop()
		// 重新启动控制器
		c.Start()
	}
}

func (c *BasicController) GetFrequency() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.frequency
}

// 策略相关
func (c *BasicController) SetStrategy(strategy Strategy) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.strategy = strategy
}

func (c *BasicController) GetStrategy() Strategy {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.strategy
}

func (c *BasicController) RegisterStrategy(strategy ...Strategy) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// range定义nil为len=0,不需要进行判断
	for _, s := range strategy {
		for _, st := range c.strategies {
			if st.Name() == s.Name() {
				slog.Warn("策略已存在, 跳过注册", "策略名称", s.Name())
				goto next
			}
		}
		slog.Info("注册新策略", "策略名称", s.Name())
		c.strategies = append(c.strategies, s)
	next:
	}
}

func (c *BasicController) UnregisterStrategy(strategy ...Strategy) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, s := range strategy {
		for i, st := range c.strategies {
			if st.Name() == s.Name() {
				c.strategies = append(c.strategies[:i], c.strategies[i+1:]...)
				break
			}
		}
	}
}

func (c *BasicController) ListStrategy() []Strategy {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.strategies
}

func (c *BasicController) GetStrategyByName(name string) Strategy {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, s := range c.strategies {
		if s.Name() == name {
			return s
		}
	}
	return nil
}
