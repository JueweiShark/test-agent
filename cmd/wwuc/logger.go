package main

import (
    "fmt"
    "os"
    "path/filepath"
    "runtime"
    "strings"

    "github.com/go-kratos/kratos/v2/log"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
    "gopkg.in/natefinch/lumberjack.v2"

    "wuwen.com/wwuc/internal/conf"
)

func InitLogger(logPath string, c *conf.Bootstrap) log.Logger {
    cwd, err := os.Getwd()
    if err != nil {
        log.Errorf("Error getting current working directory: %v\n", err)
        cwd = ""
    }
    logDir := filepath.Dir(logPath)
    _, err = os.Stat(logDir)
    if os.IsNotExist(err) {
        err := os.Mkdir(logDir, 0755)
        if err != nil {
            log.Fatal("mkdir failed, dir: ", logDir)
            return nil
        }
    }
    encoderConfig := zap.NewProductionEncoderConfig() //指定时间格式
    encoderConfig.EncodeCaller = func(caller zapcore.EntryCaller, encoder zapcore.PrimitiveArrayEncoder) {
        // 获取调用者信息
        pc, file, line, ok := runtime.Caller(8) // 调用层级
        if !ok {
            file = "unknown"
            line = 0
        }
        // 获取函数名称
        funcName := runtime.FuncForPC(pc).Name()
        // 获取文件名
        //fileName := filepath.Base(file)
        var fileName string
        var found bool
        if fileName, found = strings.CutPrefix(file, cwd+string(filepath.Separator)); !found {
            fileName = filepath.Base(file)
        }
        if idx := strings.LastIndexByte(funcName, filepath.Separator); idx > 0 && idx+1 < len(funcName) {
            funcName = funcName[idx+1:]
        }
        // 格式化输出
        encoder.AppendString(fmt.Sprintf("%s:%d %s", fileName, line, funcName))
    }
    encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
    encoderConfig.ConsoleSeparator = " "
    consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
    infoOutput := filepath.Join(logDir, c.Log.InfoFile)
    errorOutput := filepath.Join(logDir, c.Log.WarningFile)

    // 创建 Info 日志输出文件
    infoWriteSyncer := getLogWriter(infoOutput, c.Log.Split)
    // 创建 Error 日志输出文件
    errorWriteSyncer := getLogWriter(errorOutput, c.Log.Split)
    //// 创建文件 Encoder
    core := zapcore.NewCore(
        consoleEncoder,
        zapcore.AddSync(os.Stdout),
        zap.InfoLevel)
    tee := zapcore.NewTee(
        core,
        zapcore.NewCore(consoleEncoder, infoWriteSyncer, zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
            return lev >= zap.InfoLevel
        })),
        zapcore.NewCore(consoleEncoder, errorWriteSyncer, zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
            return lev >= zap.WarnLevel
        })),
    )

    zapLogger := zap.New(tee,
        zap.AddCaller(),
        zap.AddCallerSkip(3),
    )
    kZapLogger := NewLogger(zapLogger)
    logger := log.With(kZapLogger, "requestId", kratosLogValuer, "traceId", traceLogValuer)
    log.SetLogger(logger)
    _ = logger.Log(log.LevelInfo, "Logger init success", infoOutput, errorOutput)
    return logger
}

func getLogWriter(path string, c *conf.LogFileSplit) zapcore.WriteSyncer {
    lumberJackLogger := &lumberjack.Logger{
        Filename:   path,              // 文件位置
        MaxSize:    int(c.MaxSize),    // 进行切割之前,日志文件的最大大小(MB为单位)
        MaxBackups: int(c.MaxBackups), // 保留旧文件的最大天数
        MaxAge:     int(c.MaxAge),     // 保留旧文件的最大个数
        Compress:   false,             // 是否压缩/归档旧文件
    }
    return zapcore.AddSync(lumberJackLogger)
}
