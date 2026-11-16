package wuwen

import (
    "testing"

    "github.com/go-kratos/kratos/v2/log"
)

var (
    parseExecutorService *GoroutinePool = NewGoroutinePool(5)
)

func TestAtomic(t *testing.T) {
    reference := NewAtomicReference()
    reference.Store(0)
    for i := 0; i < 100; i++ {
        parseExecutorService.AddTask(func() {
            perfermAutoAdd(reference)
        })
    }

}

func perfermAutoAdd(atomic *AtomicReference) {
    atomic.Store(atomic.Load() + 5)
    log.Info(atomic.Load())
}
