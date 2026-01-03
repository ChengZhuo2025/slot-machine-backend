// Package scheduler 提供定时任务调度
package scheduler

import (
	"context"
	"log"
	"sync"
	"time"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	tasks  []*Task
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Task 定时任务
type Task struct {
	Name     string
	Interval time.Duration
	Handler  func(ctx context.Context) error
}

// NewScheduler 创建调度器
func NewScheduler() *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		tasks:  make([]*Task, 0),
		ctx:    ctx,
		cancel: cancel,
	}
}

// AddTask 添加任务
func (s *Scheduler) AddTask(name string, interval time.Duration, handler func(ctx context.Context) error) {
	s.tasks = append(s.tasks, &Task{
		Name:     name,
		Interval: interval,
		Handler:  handler,
	})
}

// Start 启动调度器
func (s *Scheduler) Start() {
	log.Printf("[Scheduler] Starting with %d tasks", len(s.tasks))

	for _, task := range s.tasks {
		s.wg.Add(1)
		go s.runTask(task)
	}
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	log.Println("[Scheduler] Stopping...")
	s.cancel()
	s.wg.Wait()
	log.Println("[Scheduler] Stopped")
}

// runTask 运行单个任务
func (s *Scheduler) runTask(task *Task) {
	defer s.wg.Done()

	log.Printf("[Scheduler] Task '%s' started, interval: %v", task.Name, task.Interval)

	ticker := time.NewTicker(task.Interval)
	defer ticker.Stop()

	// 立即执行一次
	s.executeTask(task)

	for {
		select {
		case <-s.ctx.Done():
			log.Printf("[Scheduler] Task '%s' stopped", task.Name)
			return
		case <-ticker.C:
			s.executeTask(task)
		}
	}
}

// executeTask 执行任务
func (s *Scheduler) executeTask(task *Task) {
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()

	start := time.Now()
	if err := task.Handler(ctx); err != nil {
		log.Printf("[Scheduler] Task '%s' failed: %v", task.Name, err)
	} else {
		log.Printf("[Scheduler] Task '%s' completed in %v", task.Name, time.Since(start))
	}
}
