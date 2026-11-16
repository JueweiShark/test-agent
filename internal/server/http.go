package server

import (
    http2 "net/http"
    "strings"

    "github.com/go-kratos/kratos/v2/middleware/auth/jwt"
    "github.com/go-kratos/kratos/v2/middleware/logging"
    "github.com/go-kratos/kratos/v2/middleware/selector"

    "github.com/go-kratos/kratos/v2/errors"
    "github.com/go-kratos/kratos/v2/log"
    "github.com/go-kratos/kratos/v2/middleware/recovery"
    "github.com/go-kratos/kratos/v2/middleware/tracing"
    "github.com/go-kratos/kratos/v2/transport/http"
    jwtV5 "github.com/golang-jwt/jwt/v5"
    "github.com/gorilla/handlers"

    apiErr "wuwen.com/wwuc/api/errors"
    cPb "wuwen.com/wwuc/api_gen/common"
    uPB "wuwen.com/wwuc/api_gen/user/v1"
    v2PB "wuwen.com/wwuc/api_gen/user/v2"
    "wuwen.com/wwuc/internal/conf"
    "wuwen.com/wwuc/internal/server/customCodec"
    u "wuwen.com/wwuc/internal/service/user"
    v2Svc "wuwen.com/wwuc/internal/service/v2"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Bootstrap,
    user *u.UserService, role *u.RoleService, group *u.GroupService, per *u.PermissionService,
        authV2 *v2Svc.AuthV2Service, groupV2 *v2Svc.GroupV2Service, roleV2 *v2Svc.RoleV2Service,
        menuV2 *v2Svc.MenuV2Service, userV2 *v2Svc.UserV2Service, systemV2 *v2Svc.SystemV2Service,
    logger log.Logger) *http.Server {
    opts := []http.ServerOption{
        http.Filter(
            handlers.CORS(
                handlers.AllowedOrigins([]string{"*"}),
                handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS, PUT", "DELETE"}),
                handlers.AllowedHeaders([]string{"Authorization", "IgnoreCancelToken", "Content-Type"}),
            )),
        http.Middleware(
            recovery.Recovery(),
            tracing.Server(),
            logging.Server(logger),
            LogMiddleware(logger),
            Validator(),
            selector.Server(
                jwt.Server(
                    func(token *jwtV5.Token) (interface{}, error) {
                        return []byte(c.Jwt.Secret), nil
                    },
                    jwt.WithSigningMethod(jwtV5.SigningMethodHS512),
                ), // Server检查token和解析出的信息，并写入到ctx中。注意这个ac.ApiKey要与签发(本例中是登录)的地方一致
                TokenValidationMiddleware(user),
            ).Match(NewTokenWhiteListMatcher()).Build(),
            selector.Server(
                PermissionValidationMiddleware(user),
            ).Match(NewPermissionWhiteListMatcher()).Build(),
            selector.Server(
                ValidateReqCtx(),
            ).Match(NewTokenWhiteListMatcher()).Build(),
        ),
        http.ErrorEncoder(func(w http2.ResponseWriter, r *http2.Request, err error) {
            var oriCode int32
            se := errors.FromError(err)
            if se.GetReason() == cPb.ErrorReason_PARAMS_VALIDATOR_FAIL.String() {
                oriCode = se.Code
                se.Code = apiErr.CodeParamsInvalid
            } else if se != nil {
                oriCode = se.Code
            }
            codec, _ := http.CodecForRequest(r, "Accept")
            body, err := codec.Marshal(se)
            if err != nil {
                w.WriteHeader(http2.StatusInternalServerError)
                return
            }
            w.Header().Set("Content-Type", ContentType(codec.Name()))
            w.WriteHeader(int(oriCode))
            _, _ = w.Write(body)
        },
        ),
        http.RequestDecoder(customCodec.CentralDecodeRequest),
        http.Timeout(c.Server.Http.Timeout.AsDuration()), // 设置整个请求的超时时间
    }
    if c.Server.Http.Network != "" {
        opts = append(opts, http.Network(c.Server.Http.Network))
    }
    if c.Server.Http.Addr != "" {
        opts = append(opts, http.Address(c.Server.Http.Addr))
    }
    if c.Server.Http.Timeout != nil {
        opts = append(opts, http.Timeout(c.Server.Http.Timeout.AsDuration()))
    }
    if len(c.Server.Http.Prefix) > 0 {
        opts = append(opts, http.PathPrefix(c.Server.Http.Prefix))
    }
    srv := http.NewServer(opts...)
    // v1 路由注册
    uPB.RegisterUserHTTPServer(srv, user)
    uPB.RegisterGroupHTTPServer(srv, group)
    uPB.RegisterRoleHTTPServer(srv, role)
    uPB.RegisterPermissionHTTPServer(srv, per)
    // v2 路由注册
    v2PB.RegisterAuthV2HTTPServer(srv, authV2)
    v2PB.RegisterGroupV2HTTPServer(srv, groupV2)
    v2PB.RegisterRoleV2HTTPServer(srv, roleV2)
    v2PB.RegisterMenuV2HTTPServer(srv, menuV2)
    v2PB.RegisterUserV2HTTPServer(srv, userV2)
    v2PB.RegisterSystemV2HTTPServer(srv, systemV2)
    return srv
}

const (
    baseContentType = "application"
)

// ContentType returns the content-type with base prefix.
func ContentType(subtype string) string {
    return strings.Join([]string{baseContentType, subtype}, "/")
}
