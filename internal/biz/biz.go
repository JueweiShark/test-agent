package biz

import (
    "github.com/google/wire"

    "wuwen.com/wwuc/internal/biz/user"
    v2 "wuwen.com/wwuc/internal/biz/v2"
    "wuwen.com/wwuc/internal/conf"
)

var UserBizProviderSet = wire.NewSet(
    user.NewTokenUsecase,
    user.NewUserUsecase,
    user.NewRoleUsecase,
    user.NewGroupUsecase,
    user.NewRefRolePermissionUsecase,
    user.NewPermissionUsecase,
)

var ProviderSet = wire.NewSet(
    UserBizProviderSet,
    CasbinRuleProviderSet,
    v2.ProviderSet,
    // 从 bootstrap 提取密码策略配置, 供 v2.NewUserUsecase 注入
    NewPasswordConfig,
)

var CasbinRuleProviderSet = wire.NewSet(
    //casbin_rule.InitCasbin,
)

// NewPasswordConfig 从 Bootstrap 提取密码策略配置指针
func NewPasswordConfig(b *conf.Bootstrap) *conf.Password {
    if b == nil {
        return nil
    }
    return b.Password
}

