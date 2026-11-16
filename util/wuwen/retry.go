package wuwen

import (
    "errors"
    "time"

    "github.com/go-kratos/kratos/v2/log"
)

const (
    MinRetryInterval = 1 * time.Millisecond
)

type ModeType uint8

const (
    NormalRetry ModeType = iota
    ForeverRetry
)

var modeString = map[ModeType]string{
    NormalRetry:  "normal",
    ForeverRetry: "forever",
}

type RetryOption struct {
    Mode ModeType

    // 重试间隔. 最小为 100ms.
    RetryInterval       time.Duration
    retryIntervalString string

    // 如果为 0, 每次重试出错都会打印. 否则按时间间隔打印错误.
    PrintErrInterval time.Duration
    RetryTimes       uint64 // 重试次数.
    RetryErrs        []error
}

func (o *RetryOption) IsForever() bool {
    return o.Mode == ForeverRetry
}

type RetryInfo struct {
    RunTimes uint64 // 当前是第几次运行, 从 1 开始. 包括第一次运行, 始终为 RetryTimes + 1.

    RetryTimes uint64 // 当前是第几次重试, 从 1 开始. 第一次运行时此项为 0.
    LastErr    error  // 上次错误情况
}

func RetryDo(description string, opt RetryOption, fn func(info RetryInfo) (e error)) (err error) {
    // 检查
    if fn == nil {
        return
    }

    if opt.RetryInterval < MinRetryInterval {
        opt.RetryInterval = MinRetryInterval
    }

    // 执行
    err = fn(RetryInfo{
        RunTimes:   1, // 第一次运行
        RetryTimes: 0, // 此时还没有重试
        LastErr:    nil,
    })

    // 成功
    if err == nil {
        log.Info("retry success", description)
        return
    }
    log.Info("retry begin", description)
    // 重试
    var lastErr = err
    var lastErrTimePoint = time.Now()

    var retriedTimes uint64

    for {
        // 检查是否忽略错误
        var isRetryErr bool
        if len(opt.RetryErrs) == 0 {
            // 如果没有指定重试哪些错误, 所有错误都重试
            isRetryErr = true
        } else {
            // 如果指定了重试哪些错误, 则判断
            isRetryErr = checkErr(err, opt.RetryErrs)
        }

        // 检查是否耗尽重试次数
        var isOutOfRetries bool
        if opt.IsForever() {
            isOutOfRetries = false
        } else if retriedTimes >= opt.RetryTimes {
            isOutOfRetries = true
        }

        // 退出条件
        if !isRetryErr || isOutOfRetries {
            log.Errorf("description = %v, retry out err = %v", description, err)
            return lastErr
        }

        // 重试出现错误, 将要继续重试. 表示现在已是第几次重试
        retriedTimes++

        // 开始一次重试
        err = fn(RetryInfo{
            RunTimes:   retriedTimes + 1,
            RetryTimes: retriedTimes,
            LastErr:    lastErr,
        })
        if err == nil {
            log.Errorf("description = %v, retry run fail err = %v", description, err)
            return
        }

        // 间隔打印错误
        if time.Since(lastErrTimePoint) > opt.PrintErrInterval || !errors.Is(err, lastErr) {
            lastErr = err
            lastErrTimePoint = time.Now()
            log.Errorf("description = %v, retry run fail err = %v", description, lastErr)

        }

        time.Sleep(opt.RetryInterval)
    }
}

// checkErr 错误判断
func checkErr(err error, RetryErrs []error) bool {
    var isRetryErr bool
    for i := range RetryErrs {
        if errors.Is(err, RetryErrs[i]) {
            isRetryErr = true
            return isRetryErr
        }
    }
    return isRetryErr
}
