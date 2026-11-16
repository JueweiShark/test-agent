package user

import (
    "context"

    "github.com/go-kratos/kratos/v2/log"

    "wuwen.com/wwuc/internal/biz/user"
    "wuwen.com/wwuc/internal/data/common"
    "wuwen.com/wwuc/internal/data/schema"
)

type tokenRepo struct {
    data  *common.Data
    log   *log.Helper
    model schema.UserToken
}

func (r tokenRepo) First(ctx context.Context, t string) (*user.Token, error) {
    var st schema.UserToken
    err := r.data.DB(ctx).Model(r.model).First(&st, "token = ?", t).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("find token err :%v", err)
        return nil, err
    }
    bo := common.ConvertData[schema.UserToken, user.Token](ctx, &st)
    return bo, nil
}

func (r tokenRepo) Create(ctx context.Context, t *user.Token) error {
    sToken := common.ConvertData[user.Token, schema.UserToken](ctx, t)
    err := r.data.DB(ctx).Model(r.model).Create(sToken).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("create user error:%v", err)
        return err
    }
    return nil
}

func (r tokenRepo) DeleteByUserId(ctx context.Context, userId int64) error {
    err := r.data.DB(ctx).Model(r.model).
        Where("created_by = ?", userId).
        Delete(&schema.UserToken{}).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("delete token by userId err:%v", err)
    }
    return err
}

// NewTokenRepo .
func NewTokenRepo(data *common.Data, logger log.Logger) user.TokenRepo {
    return &tokenRepo{
        data:  data,
        log:   log.NewHelper(logger),
        model: schema.UserToken{},
    }
}
