package wuwen

import (
    "fmt"
    "net/http"
    _ "net/http/pprof"
    "os"
    "runtime"
    "runtime/pprof"
    "sync"
    "testing"

    "github.com/go-kratos/kratos/v2/log"
)

func TestWalkDir(t *testing.T) {
    file, relationMap, err := WalkDir("./mapper")
    fmt.Println(file)
    fmt.Println(relationMap)
    fmt.Println(len(relationMap))
    fmt.Println(err)

}
func TestRoutine(t *testing.T) {
    pool := NewGoroutinePool(10)
    startPprofServer()
    //// 开始 CPU 配置文件
    //f, err := os.Create("cpu_profile.pprof")
    //if err != nil {
    //    t.Fatalf("could not create CPU profile: %v", err)
    //}
    //defer f.Close()
    //if err := pprof.StartCPUProfile(f); err != nil {
    //    t.Fatalf("could not start CPU profile: %v", err)
    //}
    //defer pprof.StopCPUProfile()

    // 内存使用上限 (例如 50MB)
    maxAlloc := uint64(5000 * 1024 * 1024)

    var url string = "http://192.168.2.49:9527/testdata/jili_dyrh_25g.zip"
    for i := 0; i < 100; i++ {
        idx := i
        pool.AddTask(func() {
            err := DownloadFile(url, fmt.Sprintf("saveFile_%d", idx), func(step, progress, totalsize int64) {
                if step%5000 == 0 {
                    log.Infof("%s  total %d ;progress:%d ", url, totalsize, progress)
                }
                // 调用待测函数后检查内存
                CheckMemoryUsage(t, maxAlloc)
            })
            if err != nil {
                t.Errorf("url download fail %v", err)
                return
            }
        })
    }

    for {
        select {}
    }
}
func TestDownloadFileFromUrl(t *testing.T) {
    startPprofServer()
    //// 开始 CPU 配置文件
    //f, err := os.Create("cpu_profile.pprof")
    //if err != nil {
    //    t.Fatalf("could not create CPU profile: %v", err)
    //}
    //defer f.Close()
    //if err := pprof.StartCPUProfile(f); err != nil {
    //    t.Fatalf("could not start CPU profile: %v", err)
    //}
    //defer pprof.StopCPUProfile()

    // 内存使用上限 (例如 50MB)
    maxAlloc := uint64(5000 * 1024 * 1024)

    var url string = "http://192.168.2.49:9527/testdata/jili_dyrh_25g.zip"
    group := sync.WaitGroup{}
    r := 20
    group.Add(r)

    for i := 0; i < r; i++ {
        index := i
        saveFile, err := CreateFile(fmt.Sprintf("%s_%d", "saveFile", index))
        go func() {
            defer group.Done()
            if err != nil {
                log.Errorf("create file fail %v", err)
                t.Error("create file fail")
            }
            err = DownloadFileFromUrl(url, saveFile, func(step, progress, totalsize int64) {
                if step%5000 == 0 {
                    log.Infof("%s  total %d ;progress:%d ", url, totalsize, progress)
                }
                // 调用待测函数后检查内存
                CheckMemoryUsage(t, maxAlloc)
            })
            if err != nil {
                t.Errorf("url download fail %v", err)
                return
            }
        }()

    }
    group.Wait()
    // 强制 GC 并记录内存配置文件
    runtime.GC()
    fmem, err := os.Create("mem_profile.pprof")
    if err != nil {
        t.Fatalf("could not create memory profile: %v", err)
    }
    defer fmem.Close()
    if err := pprof.WriteHeapProfile(fmem); err != nil {
        t.Fatalf("could not write memory profile: %v", err)
    }
}

// CheckMemoryUsage 检查当前内存使用情况
func CheckMemoryUsage(t *testing.T, maxAlloc uint64) {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    if m.Alloc > maxAlloc {
        t.Fatalf("Memory usage exceeded: %d bytes used, limit is %d bytes", m.Alloc, maxAlloc)
    }
    t.Logf("Memory usage used: %d bytes", m.Alloc)
}

func startPprofServer() {
    go func() {
        log.Info(http.ListenAndServe(":6060", nil))
    }()
}

func TestMain(m *testing.M) {
    startPprofServer()
    fmt.Println("start pprof server")
    os.Exit(m.Run())
}
