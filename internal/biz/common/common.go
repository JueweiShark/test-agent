package common

import (
    "context"
)

type Transaction interface {
    InTx(context.Context, func(ctx context.Context) error) error
}

type RequestInfo struct {
    userId    int64
    endPoint  string
    protocol  string
    requestId string
}

func (r *RequestInfo) GetUserId() int64 {
    if r == nil {
        return 0
    }
    return r.userId
}

func (r *RequestInfo) GetRequestId() interface{} {
    if r == nil {
        return "[nil]"
    }
    return r.requestId
}

type reqCtxKey struct{}

type userKey struct{}

var rqCtxKey reqCtxKey
var uKey userKey

func NewReqCtx(ctx context.Context, userId int64, reId string) context.Context {
    return context.WithValue(ctx, rqCtxKey, &RequestInfo{userId: userId, requestId: reId})
}

// ReqCtxFromContext extract auth info from context
func ReqCtxFromContext(ctx context.Context) (reqCtx *RequestInfo) {
    reqCtx, _ = ctx.Value(rqCtxKey).(*RequestInfo)
    return
}
func HasReqCtx(ctx context.Context) (ok bool) {
    _, ok = ctx.Value(rqCtxKey).(*RequestInfo)
    return
}

func GetPageOffset(pageNum, pageSize int64) (int64, int64) {
    if pageNum <= 0 {
        pageNum = 1
    }
    if pageSize <= 0 {
        pageSize = 20
    }
    return (pageNum - 1) * pageSize, pageSize
}
