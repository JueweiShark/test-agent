package service

import (
    "github.com/google/wire"

    v2 "wuwen.com/wwuc/internal/service/v2"
    u "wuwen.com/wwuc/internal/service/user"
)

var userSet = wire.NewSet(
    u.NewUserService,
    u.NewRoleService,
    u.NewGroupService,
    u.NewPermissionService,
)

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(
    userSet,
    v2.ProviderSet,
)
