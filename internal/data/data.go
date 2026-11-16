package data

import (
    "github.com/google/wire"

    "wuwen.com/wwuc/internal/biz/casbin_rule"
    "wuwen.com/wwuc/internal/data/common"
    "wuwen.com/wwuc/internal/data/user"
    v2 "wuwen.com/wwuc/internal/data/v2"
)

var commonSet = wire.NewSet(
    common.NewData,
    common.NewDB,
    common.NewRedis,
    common.NewTransaction,
)

// ProviderSet is data providers.
var userSet = wire.NewSet(
    user.NewTokenRepo,
    user.NewUserRepo,
    user.NewRoleRepo,
    user.NewPermissionRepo,
    user.NewRolePermissionRepo,
    user.NewGroupRepo,
    user.NewRefUserRoleRepo,
    user.NewRefUserGroupRepo,
)

var casbinSet = wire.NewSet(
    casbin_rule.NewCasbinRuleUsecase,
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
    commonSet,
    userSet,
    casbinSet,
    v2.ProviderSet,
)
