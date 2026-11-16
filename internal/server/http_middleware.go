package server

import (
    "context"
    "strings"
    "time"
    
    "github.com/go-kratos/kratos/v2/errors"
    "github.com/go-kratos/kratos/v2/log"
    "github.com/go-kratos/kratos/v2/middleware"
    "github.com/go-kratos/kratos/v2/middleware/auth/jwt"
    "github.com/go-kratos/kratos/v2/middleware/selector"
    "github.com/go-kratos/kratos/v2/transport"
    "github.com/go-kratos/kratos/v2/transport/http"
    "google.golang.org/grpc/metadata"
    
    apiErr "wuwen.com/wwuc/api/errors"
    cPb "wuwen.com/wwuc/api_gen/common"
    
    "wuwen.com/wwuc/internal/biz/common"
    service "wuwen.com/wwuc/internal/service/user"
    "wuwen.com/wwuc/util/wuwen/mapper"
)

var (
    userid           = "x-vcs-userid"
    username         = "x-vcs-username"
    userfullname     = "x-vcs-userfullname"
    groupid = "x-vcs-groupid"
    authorizationKey = "Authorization"
    bearerWord       = "Bearer"
    RequestId        = "Request-Id"
)

// NewTokenWhiteListMatcher 创建白名单
func NewTokenWhiteListMatcher() selector.MatchFunc {
    whiteList := make(map[string]bool)
    whiteList["/api.user.v1.User/Login"] = true
    whiteList["/api.user.v1.User/Logout"] = true
    // v2 登录接口白名单
    whiteList["/api.user.v2.AuthV2/LoginV2"] = true
    // v2 换发组 token (数据隔离改造): 接口自身通过 body 中的基础 token 鉴权, 不依赖 Authorization 头
    whiteList["/api.user.v2.AuthV2/IssueGroupToken"] = true
    return func(ctx context.Context, operation string) bool {
        if pass, ok := whiteList[operation]; ok {
            return !pass
        }
        return true
    }
}
func NewPermissionWhiteListMatcher() selector.MatchFunc {
    whiteList := make(map[string]bool)
    whiteList["/api.user.v1.User/Login"] = true
    whiteList["/api.user.v1.User/Logout"] = true
    whiteList["/api.user.v1.User/UserInfo"] = true
    whiteList["/api.user.v1.User/CreateInnerUser"] = true
    whiteList["/api.user.v1.User/Uid2Token"] = true
    whiteList["/api.user.v1.User/UpdateUser"] = true
    whiteList["/api.user.v1.User/UserGroupInfo"] = true
    whiteList["/api.user.v1.User/UpdateUserRole"] = true
    whiteList["/api.user.v1.User/ListUser"] = true
    whiteList["/api.user.v1.User/ForbiddenUser"] = true
    whiteList["/api.user.v1.User/CreateUser"] = true
    whiteList["/api.user.v1.User/DeleteUser"] = true
    whiteList["/api.user.v1.User/UpdateInfo"] = true
    whiteList["/api.user.v1.User/ListPermissionUser"] = true
    whiteList["/api.user.v1.User/FindUserByCondRequest"] = true
    whiteList["/api.user.v1.Group/ListGroup"] = true
    whiteList["/api.user.v1.Group/ListGroupAll"] = true
    whiteList["/api.user.v1.Group/FindGroupList"] = true
    whiteList["/api.user.v1.Group/UpdateGroup"] = true
    whiteList["/api.user.v1.Group/CreateGroup"] = true
    whiteList["/api.user.v1.Group/GetGroup"] = true
    whiteList["/api.user.v1.Group/MemberList"] = true
    whiteList["/api.user.v1.Group/AddMember"] = true
    whiteList["/api.user.v1.Group/DeleteGroup"] = true
    whiteList["/api.user.v1.Group/RemoveMember"] = true
    whiteList["/api.user.v1.Role/CreateRole"] = true
    whiteList["/api.user.v1.Role/UpdateRole"] = true
    whiteList["/api.user.v1.Role/GetRole"] = true
    whiteList["/api.user.v1.Role/ListRole"] = true
    whiteList["/api.user.v1.Role/DeleteRole"] = true
    whiteList["/api.user.v1.Permission/ListPermission"] = true
    whiteList["/api.user.v1.Permission/UpdatePermission"] = true
    whiteList["/api.user.v1.Permission/CreatePermission"] = true
    whiteList["/api.user.v1.Permission/DeletePermission"] = true

    return func(ctx context.Context, operation string) bool {
        // v2 接口走自己的权限模型, 不走 v1 的 permission 中间件 (对应 design 2.4.4 F3=a)
        if strings.HasPrefix(operation, "/api.user.v2.") {
            return false
        }

        if pass, ok := whiteList[operation]; ok {
            return !pass
        }
        return true
    }
}

func ValidateReqCtx() middleware.Middleware {

    return func(handler middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req interface{}) (interface{}, error) {
            if ok := common.HasReqCtx(ctx); !ok {
                return nil, errors.BadRequest("cant find ReqCtx", "cant find ReqCtx")
            }
            return handler(ctx, req)
        }
    }
}

func TokenValidationMiddleware(user *service.UserService) middleware.Middleware {
    return func(handler middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req interface{}) (interface{}, error) {
            var reqId string
            token, ok := jwt.FromContext(ctx)
            if !ok {
                log.Errorf("token is miss ")
                return nil, errors.Unauthorized("token is missed", "token is missed")
            }
            if tr, ok := transport.FromServerContext(ctx); ok {

                auths := strings.SplitN(tr.RequestHeader().Get(authorizationKey), " ", 2)
                if len(auths) != 2 || !strings.EqualFold(auths[0], bearerWord) {
                    return nil, errors.Unauthorized("token is missed", "token not exist")
                }
                jwtToken := auths[1]
                if !user.CheckTokenExist(ctx, jwtToken) {
                    log.Errorf("token is miss ")
                    return nil, errors.Unauthorized("token is missed", "token is missed")
                }
            }
            if serverContext, ok := http.RequestFromServerContext(ctx); ok {
                reqId = serverContext.Header.Get("Request-Id")
            }

            expirationTime, err := token.GetExpirationTime()
            if err != nil {
                return nil, errors.Unauthorized("token invalid", "token invalid")
            }
            if expirationTime.Time.Before(time.Now()) {
                return nil, errors.Unauthorized("token expired", "token expired")
            }
            subject, err := token.GetSubject()
            if err != nil {
                log.Error("get subject fail")
            }
            userId, err := mapper.Convert(subject).Int64()
            if err != nil {
                log.Errorf("parse user id fail %s", subject)
                return nil, err
            }
            ctx = common.NewReqCtx(ctx, userId, reqId)

            // 继续处理请求
            return handler(ctx, req)
        }
    }
}

func PermissionValidationMiddleware(user *service.UserService) middleware.Middleware {
    return func(handler middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req interface{}) (interface{}, error) {
            var path, method string
            err := user.CheckPermission(ctx, path, method)
            if err != nil {
                return nil, errors.Unauthorized("permission not match", "permission miss")
            }
            // 继续处理请求
            return handler(ctx, req)
        }
    }
}

// LogMiddleware is a middleware for logging request and response content
func LogMiddleware(logger log.Logger) middleware.Middleware {
    return func(handler middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req interface{}) (interface{}, error) {
            startTime := time.Now()
            // Log the request content

            // Handle the request
            reply, err := handler(ctx, req)
            // Log the response content
            if tr, ok := transport.FromServerContext(ctx); ok {
                // 绑定请求 ctx，使 logger 的 traceId/requestId valuer 能从 ctx 取值
                log.NewHelper(log.WithContext(ctx, logger)).Infof("Operation :{%s};Duration: {%+v}<===>Response:{ %+v}", tr.Operation(), time.Since(startTime), reply)
            }
            return reply, err
        }
    }
}

type validator interface {
    Validate() error
}

// Validator is a validator middleware.
func Validator() middleware.Middleware {
    return func(handler middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
            if v, ok := req.(validator); ok {
                if err := v.Validate(); err != nil {
                    return &cPb.ParamsInvalidReplay{
                        Code:    apiErr.CodeParamsInvalid,
                        Message: err.Error(),
                    }, cPb.ErrorParamsValidatorFail("VALIDATOR", err.Error())
                }
            }
            return handler(ctx, req)
        }
    }
}

// 获取请求协议
func getProtocol(r *http.Request) string {
    if r.TLS != nil {
        return "https"
    }
    return "http"
}
func ReqCtxGrpcMiddleware() middleware.Middleware {
    return func(handler middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req interface{}) (interface{}, error) {
            md, b := metadata.FromIncomingContext(ctx)
            var userId, rid string
            if b {
                vals := md[userid]
                if len(vals) > 0 {
                    userId = vals[0]
                }
                vals = md[RequestId]
                if len(vals) > 0 {
                    rid = vals[0]
                }
            }
            uid, _ := mapper.Convert(userId).Int64()
            ctx = common.NewReqCtx(ctx, uid, rid)
            return handler(ctx, req)
        }
    }
}
