package server

import (
    "github.com/go-kratos/kratos/v2/log"
    "github.com/go-kratos/kratos/v2/middleware/logging"
    "github.com/go-kratos/kratos/v2/middleware/recovery"
    "github.com/go-kratos/kratos/v2/middleware/tracing"
    "github.com/go-kratos/kratos/v2/transport/grpc"

    uPB "wuwen.com/wwuc/api_gen/user/v1"
    v2PB "wuwen.com/wwuc/api_gen/user/v2"
    "wuwen.com/wwuc/internal/conf"
    u "wuwen.com/wwuc/internal/service/user"
    v2Svc "wuwen.com/wwuc/internal/service/v2"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Bootstrap, logger log.Logger, user *u.UserService, group *u.GroupService, authV2 *v2Svc.AuthV2Service, groupV2 *v2Svc.GroupV2Service) *grpc.Server {
    var opts = []grpc.ServerOption{
        grpc.Middleware(
            recovery.Recovery(),
            tracing.Server(),
            logging.Server(logger),
            LogMiddleware(logger),
            ReqCtxGrpcMiddleware(),
        ),
    }
    if c.Server.Grpc.Network != "" {
        opts = append(opts, grpc.Network(c.Server.Grpc.Network))
    }
    if c.Server.Grpc.Addr != "" {
        opts = append(opts, grpc.Address(c.Server.Grpc.Addr))
    }
    if c.Server.Grpc.Timeout != nil {
        opts = append(opts, grpc.Timeout(c.Server.Grpc.Timeout.AsDuration()))
    }
    srv := grpc.NewServer(opts...)
    // v1 gRPC 服务注册
    uPB.RegisterUserServer(srv, user)
    uPB.RegisterGroupServer(srv, group)
    // v2 gRPC 服务注册 (AuthV2 的 CheckUserGroupAccess 供下游/网关调用; GroupV2 供下游查询企业)
    v2PB.RegisterAuthV2Server(srv, authV2)
    v2PB.RegisterGroupV2Server(srv, groupV2)
    return srv
}
