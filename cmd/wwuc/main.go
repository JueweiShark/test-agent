package main

import (
    "context"
    "flag"
    "os"

    "wuwen.com/wwuc/internal/conf"
    "wuwen.com/wwuc/internal/pkg/trace"
    "wuwen.com/wwuc/pkg/bootstrap"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
    // Name is the name of the compiled software.
    Name string
    // Version is the version of the compiled software.
    Version string
    // flagConf is the config flag.

    runtimeConf string

    flagLog string

    id, _ = os.Hostname()

    flagInitDB string

    flagParams string
)

func init() {
    flag.StringVar(&runtimeConf, "conf", "./conf/wsi.yaml", "config path, eg: -conf wsi.yaml")
    flag.StringVar(&flagLog, "log", "./log/wsi.log", "log path, eg: -log log/wsi.log")

    flag.StringVar(&flagInitDB, "method", "", "db update method, eg: -method createAdmin")
    flag.StringVar(&flagParams, "params", "", "params for createAdmin: username=xxx&password=xxx&nickname=xxx&appId=3&roleId=1")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
    return kratos.New(
        kratos.ID(id),
        kratos.Name("wsi-user"),
        kratos.Version("v0.0.1"),
        kratos.Metadata(map[string]string{}),
        kratos.Logger(logger),
        kratos.Server(
            gs,
            hs,
        ),
        //kratos.Registrar(r),
    )
}

func main() {
    flag.Parse()

    c := bootstrap.NewConfigProvider(nil, runtimeConf)
    defer c.Close()
    var bc conf.Bootstrap
    if err := c.Load(); err != nil {
        panic(err)
    }
    if err := c.Scan(&bc); err != nil {
        panic(err)
    }
    logger := InitLogger(flagLog, &bc)
    if len(flagInitDB) > 0 {
        updateDB(&bc, logger, flagInitDB)
        return
    }
    log.Infof("server conf %+v", &bc)
    //err := discovery.Init(&bc, false)
    //if err != nil {
    //    log.Errorf("err: %+v", err)
    //    panic(err)
    //}
    // 链路追踪：初始化全局 TracerProvider（未开启时为 noop），退出时上报剩余 span
    shutdownTrace, err := trace.Init(context.Background(), bc.Trace, "wwuc")
    if err != nil {
        log.Warnf("trace init failed: %+v", err)
    } else {
        defer func() { _ = shutdownTrace(context.Background()) }()
    }
    app, cleanup, err := wireApp(&bc, logger)
    if err != nil {
        panic(err)
    }

    defer cleanup()
    // start and wait for stop signal
    if err := app.Run(); err != nil {
        panic(err)
    }
}
