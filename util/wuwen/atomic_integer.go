package wuwen

import "sync/atomic"

type AtomicReference struct {
    value int64
}

func NewAtomicReference() *AtomicReference {
    return &AtomicReference{}
}

// Store 设置原子值
func (ar *AtomicReference) Store(num int64) {
    atomic.StoreInt64(&(ar.value), num)
}

// Load 获取原子值
func (ar *AtomicReference) Load() int64 {
    return atomic.LoadInt64(&(ar.value))
}

func (ar *AtomicReference) LoadAndIncrement() int64 {
    load := ar.Load()
    ar.Store(load + 1)
    return load
}
