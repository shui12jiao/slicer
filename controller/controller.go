package controller

import (
	"context"
	"log/slog"
	"slicer/db"
	"slicer/kubeclient"
	"slicer/model"
	"slicer/util"
	"sync"
	"time"
)

type Controller interface {
	Run()

	Stop()

	// 设置控制频率
	SetFrequency(duration time.Duration)

	AddSlice(sliceID string)
	RemoveSlice(sliceID string)
	ListSlices() []string

	Strategy
	SetStrategy(strategy Strategy)
}

type BasicController struct {
	running bool
	mu      sync.Mutex
	// 控制频率
	frequency time.Duration
	// 控制器的上下文
	ctx context.Context
	// 控制器的取消函数
	cancel context.CancelFunc

	// 切片列表
	slices []string
	// config
	config util.Config
	// 存储
	store db.Store
	// 控制器的配置
	kclient *kubeclient.KubeClient
	// 策略
	strategy Strategy
}

func NewBasicController(config util.Config, store db.Store, kclient *kubeclient.KubeClient) Controller {
	// 创建一个新的上下文和取消函数
	ctx, cancel := context.WithCancel(context.Background())

	return &BasicController{
		running:   false,
		frequency: 1 * time.Hour,
		ctx:       ctx,
		cancel:    cancel,
		slices:    []string{},
		config:    config,
		store:     store,
		kclient:   kclient,
	}
}

func (c *BasicController) Run() {
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

// 核心逻辑, 根据 SLA 与当前指标，生成新 Play 策略
func (c *BasicController) Reconcile(current model.Play, sla model.SLA) (new model.Play, err error) {
	return c.strategy.Reconcile(current, sla)
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

	// 生成新的Play
	newPlay, err := c.Reconcile(play, sla)
	if err != nil {
		slog.Error("生成新Play失败", "sliceID", sliceID, "err", err)
		return err
	}

	// 应用新的Play
	err = c.kclient.ApplyPlay(newPlay, c.config.Namespace)
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

func (c *BasicController) SetFrequency(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.frequency = duration

	// 重启
	if c.running {
		c.Stop()
		// 重新启动控制器
		c.Run()
	}
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

func (c *BasicController) SetStrategy(strategy Strategy) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.strategy = strategy
}
