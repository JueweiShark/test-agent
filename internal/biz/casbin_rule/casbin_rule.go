package casbin_rule

import (
    "github.com/casbin/casbin/v2"
    "github.com/go-kratos/kratos/v2/log"

    "wuwen.com/wwuc/internal/biz/common"
)

type CasbinRuleRepo interface {
    //Create(ctx context.Context, e *User) error
    //Update(ctx context.Context, e *User) error
    //FindOne(ctx context.Context, id int64) (*User, error)
    //FindByPage(ctx context.Context, offset, limit int) ([]*User, int64, error)
    //Delete(ctx context.Context, userId, id int64) error
    //FindOneByUserName(ctx context.Context, userName string) (*User, error)
    //FindByIds(ctx context.Context, ids []int64) ([]*User, error)
}
type CasbinRuleUsecase struct {
    log      *log.Helper
    crr      CasbinRuleRepo
    tm       common.Transaction
    enforcer *casbin.SyncedEnforcer
}

func NewCasbinRuleUsecase(logger log.Logger, tm common.Transaction,
    crr CasbinRuleRepo, enforcer *casbin.SyncedEnforcer) *CasbinRuleUsecase {
    return &CasbinRuleUsecase{
        log:      log.NewHelper(logger),
        crr:      crr,
        tm:       tm,
        enforcer: enforcer,
    }
}

func (uc *CasbinRuleUsecase) CheckPermission(sub string, obj string, act string) (bool, error) {
    ok, err := uc.enforcer.Enforce(sub, obj, act)
    if err != nil {
        return false, err
    }
    return ok, nil
}
