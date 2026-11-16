//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
    "wuwen.com/wwuc/internal/biz"
    "wuwen.com/wwuc/internal/conf"
    "wuwen.com/wwuc/internal/data"
    "wuwen.com/wwuc/internal/pkg/oss"
    "wuwen.com/wwuc/internal/server"
    "wuwen.com/wwuc/internal/service"

    "github.com/go-kratos/kratos/v2"
    "github.com/go-kratos/kratos/v2/log"
    "github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Bootstrap, log.Logger) (*kratos.App, func(), error) {
    panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, oss.ProviderSet, service.ProviderSet, newApp))
}
