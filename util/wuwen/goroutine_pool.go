package wuwen

import (
    "sync"
)

// GoroutinePool 表示 Goroutine 池
type GoroutinePool struct {
    taskChannel chan Task
    wg          sync.WaitGroup
}

type Task func()

// NewGoroutinePool 创建一个新的 Goroutine 池
func NewGoroutinePool(workerCount int) *GoroutinePool {
    pool := &GoroutinePool{
        taskChannel: make(chan Task),
    }

    pool.startWorkers(workerCount)

    return pool
}

// startWorkers 启动 Goroutine 池中的工作线程
func (p *GoroutinePool) startWorkers(workerCount int) {
    for i := 0; i < workerCount; i++ {
        go func() {
            for task := range p.taskChannel {
                p.processTask(task)
                p.wg.Done()
            }
        }()
    }
}

// processTask 处理任务
func (p *GoroutinePool) processTask(task Task) {
    // 模拟任务执行时间
    task()
}

// AddTask 添加任务到 Goroutine 池
func (p *GoroutinePool) AddTask(task Task) {
    p.wg.Add(1)
    p.taskChannel <- task
}

// Destroy 销毁 Goroutine 池
func (p *GoroutinePool) Destroy() {
    close(p.taskChannel)
    p.wg.Wait()
}
