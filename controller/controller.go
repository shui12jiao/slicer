package controller

import (
	"context"
	"log/slog"
	"slicer/db"
	"slicer/kubeclient"
	"slicer/util"
	"sync"
	"time"
)

type Controller interface {
	// 运行
	Start() // 异步运行
	Stop()
	IsRunning() bool // 控制器是否在运行

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
	mu sync.Mutex // 保护running, frequency, strategy, slices
	// 控制器的上下文
	ctx context.Context
	// 控制器的取消函数
	cancel context.CancelFunc

	// config
	config util.Config
	// 存储
	store db.Store
	// 控制器的配置
	kclient *kubeclient.KubeClient

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
func NewBasicController(config util.Config, store db.Store, kclient *kubeclient.KubeClient, strategy ...Strategy) Controller {
	// 创建一个新的上下文和取消函数
	ctx, cancel := context.WithCancel(context.Background())

	c := &BasicController{
		running:   false,
		frequency: 1 * time.Hour,
		ctx:       ctx,
		cancel:    cancel,
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
	go c.run()
}

func (c *BasicController) run() {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.mu.Unlock()

	ticker := time.NewTicker(c.frequency)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done(): // 停止
			c.mu.Lock()
			c.running = false
			c.mu.Unlock()
			return
		case <-ticker.C:
			// 执行控制逻辑
			for _, sliceID := range c.slices {
				err := c.control(sliceID)
				if err != nil {
					slog.Error("控制失败, 跳过", "sliceID", sliceID, "err", err)
					continue
				}
			}
		}
	}
}

func (c *BasicController) control(sliceID string) error {
	// 获取SLA
	sla, err := c.store.GetSLA(sliceID)
	if err != nil {
		slog.Error("获取SLA失败", "sliceID", sliceID, "err", err)
		return err
	}

	// 获取Play
	play, err := c.store.GetPlay(sliceID)
	if err != nil {
		slog.Error("获取Play失败", "sliceID", sliceID, "err", err)
		return err
	}

	// 核心控制逻辑
	// 调用策略执行Reconcile
	// 生成新的Play
	newPlay, err := c.strategy.Reconcile(play, sla)
	if err != nil {
		slog.Error("生成新Play失败", "sliceID", sliceID, "err", err)
		return err
	}

	// 应用新的Play
	err = c.kclient.Play(newPlay, c.config.Namespace)
	if err != nil {
		slog.Error("应用Play失败", "sliceID", sliceID, "err", err)
		return err
	}

	// 更新Play
	_, err = c.store.UpdatePlay(newPlay)
	if err != nil {
		slog.Error("更新Play失败", "sliceID", sliceID, "err", err)
		return err
	}

	// 更新SLA
	_, err = c.store.UpdateSLA(sla)
	if err != nil {
		slog.Error("更新SLA失败", "sliceID", sliceID, "err", err)
		return err
	}

	// 完成
	slog.Info("控制完成", "sliceID", sliceID, "newPlay", newPlay)
	return nil
}

func (c *BasicController) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		c.cancel() // 取消上下文
		c.running = false

		// 等待控制器停止
		time.Sleep(100 * time.Millisecond)

		// 重新创建上下文和取消函数
		c.ctx, c.cancel = context.WithCancel(context.Background())
	}
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
				continue
			}
			c.strategies = append(c.strategies, s)
		}
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
