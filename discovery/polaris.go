package discovery

import (
    "context"
    "fmt"
    "strings"
    "sync"
    "time"

    "wuwen.com/wwuc/internal/conf"

    "github.com/go-kratos/kratos/contrib/registry/polaris/v2"
    "github.com/go-kratos/kratos/v2/log"
    "github.com/go-kratos/kratos/v2/selector"
    "github.com/go-kratos/kratos/v2/selector/filter"
    "github.com/go-kratos/kratos/v2/selector/wrr"
    transGrpc "github.com/go-kratos/kratos/v2/transport/grpc"
    "github.com/go-kratos/kratos/v2/transport/http"
    "github.com/polarismesh/polaris-go/api"
    pconfig "github.com/polarismesh/polaris-go/pkg/config"
    "google.golang.org/grpc"
)

var Discovery *WSIDiscovery

type WSIDiscovery struct {
    conf              *conf.Bootstrap
    httpConnectionMap sync.Map
    grpcConnectionMap sync.Map
}

func Init(config *conf.Bootstrap, block bool) error {
    if Discovery == nil {
        Discovery = &WSIDiscovery{
            config, sync.Map{}, sync.Map{},
        }
    }
    var err error

    switch config.ServiceConf.UseServiceConf {
    case conf.ServiceConfig_local:
        log.Info("load local discovery")
        err = Discovery.loadLocalHttpClient(config.ServiceConf, block)
        break
    case conf.ServiceConfig_polaris:
        log.Info("load Polaris discovery")
        // 创建 Polaris 客户端配置
        err = api.SetLoggersDir(config.ServiceConf.PolarisConfig.LogPath)
        if err != nil {
            log.Errorf("Init WDI Discovery set logger path  Error: %v", err)
            return err
        }
        err = api.SetLoggersLevel(int(config.ServiceConf.PolarisConfig.LogLevel))
        if err != nil {
            log.Errorf("Init WDI Discovery set logger level Error: %v", err)
            return err
        }
        log.Info("Init WSI Discovery,", config.ServiceConf.PolarisConfig.LogPath)
        err = Discovery.loadPolarisHttpClient(config.ServiceConf, block)
        break
    default:
        err = fmt.Errorf("invalid discovery service name %s", config.ServiceConf.UseServiceConf)
    }

    if err != nil {
        log.Errorf("Init  Discovery:load HttpClient,Error: %v", err)
        return err
    }
    return nil
}
func (PD *WSIDiscovery) loadLocalHttpClient(config *conf.ServiceConfig, block bool) error {
    err := PD.loadHttpClient(nil, block, config.LocalConfig.Services)
    if err != nil {
        log.Errorf("load http client err:%v", err)
        return err
    }
    return nil
}

func (PD *WSIDiscovery) loadPolarisHttpClient(config *conf.ServiceConfig, block bool) error {
    serviceConf := config.PolarisConfig
    configuration := pconfig.NewDefaultConfiguration(serviceConf.Global.ServerConnector.Address)
    selector.SetGlobalSelector(wrr.NewBuilder())
    polarisClient := polaris.NewRegistryWithConfig(configuration, polaris.WithNamespace(serviceConf.NameSpace))

    err := PD.loadHttpClient(polarisClient, block, serviceConf.Services)
    if err != nil {
        log.Errorf("load http client err:%v", err)
        return err
    }
    return nil
}

func (PD *WSIDiscovery) loadHttpClient(client *polaris.Registry, block bool, services []*conf.ServiceConfig_Service) error {
    var errMsg strings.Builder
    for _, service := range services {
        log.Infof("service %v", service)
        var httpOpts []http.ClientOption
        var grpcOpts []transGrpc.ClientOption
        endPoint := service.EndPoint
        if client != nil {
            endPoint = fmt.Sprintf("discovery:///%s", service.Name)
            httpOpts = append(httpOpts, http.WithDiscovery(client))
            grpcOpts = append(grpcOpts, transGrpc.WithDiscovery(client))
        }
        if len(service.Version) != 0 {
            httpOpts = append(httpOpts, http.WithNodeFilter(filter.Version(service.Version)))
            grpcOpts = append(grpcOpts, transGrpc.WithNodeFilter(filter.Version(service.Version)))
        }

        httpOpts = append(httpOpts, http.WithEndpoint(endPoint))
        grpcOpts = append(grpcOpts, transGrpc.WithEndpoint(endPoint))

        if block {
            httpOpts = append(httpOpts, http.WithBlock())
        }
        if service.Type == "http" {
            httpClient, err := http.NewClient(
                context.Background(),
                httpOpts...,
            )
            if err != nil {
                log.Errorf("service %v load http client fail", service)
                errMsg.WriteString("service:http:" + service.Name)
            }
            PD.httpConnectionMap.Store(service.Name, httpClient)
        } else {
            insecure, err := transGrpc.DialInsecure(context.Background(), grpcOpts...)
            if err != nil {
                log.Errorf("service %v load grpc client fail", insecure)
                errMsg.WriteString("service:grpc:" + service.Name)
            }
            PD.grpcConnectionMap.Store(service.Name, insecure)
        }
    }
    msg := errMsg.String()
    if len(msg) != 0 {
        return fmt.Errorf("%s,load http client fail", msg)
    }
    return nil
}

func (PD *WSIDiscovery) GetHttpClient(serviceName string) (*http.Client, error) {
    value, ok := PD.httpConnectionMap.Load(serviceName)
    if !ok {
        err := longRunningOperation(2*time.Second, func() error {
            err := Init(PD.conf, true)
            if err != nil {
                log.Errorf("service %s load http client fail", serviceName)
                return err
            }
            return nil
        })
        if err != nil {
            log.Errorf("service %s load http client fail %v", serviceName, err)
            return nil, err
        }
        value, _ = PD.httpConnectionMap.Load(serviceName)
    }
    client, ok := value.(*http.Client)
    if !ok {
        return nil, fmt.Errorf("get http client fail")
    }
    return client, nil
}
func (PD *WSIDiscovery) GetGrpcClient(serviceName string) (*grpc.ClientConn, error) {
    value, ok := PD.grpcConnectionMap.Load(serviceName)
    if !ok {
        err := longRunningOperation(20*time.Second, func() error {
            err := Init(PD.conf, false)
            if err != nil {
                log.Errorf("service %s load grpc client fail", serviceName)
                return err
            }
            return nil
        })
        if err != nil {
            log.Errorf("service %s load grpc client fail %v", serviceName, err)
            return nil, err
        }
        value, _ = PD.grpcConnectionMap.Load(serviceName)
    }
    client, ok := value.(*grpc.ClientConn)
    if !ok {
        return nil, fmt.Errorf("get grpc client fail")
    }
    return client, nil
}

type task func() error

func longRunningOperation(timeout time.Duration, t task) error {
    // 创建一个 channel 用于接收结果
    resultCh := make(chan error)
    go func() {
        // 模拟耗时操作
        err := t()
        if err != nil {
            resultCh <- err
        }
        resultCh <- nil
    }()

    // 使用 select 语句实现超时控制
    select {
    case err := <-resultCh:
        return err
    case <-time.After(timeout):
        return fmt.Errorf("operation timed out after %v", timeout)
    }
}
