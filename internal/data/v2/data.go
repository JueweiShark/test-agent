package v2

import "github.com/google/wire"

// ProviderSet v2 data 层 provider 集合
var ProviderSet = wire.NewSet(
	NewUserRepo,
	NewGroupRepo,
	NewRefGroupPermissionRepo,
	NewRoleRepo,
	NewRefUserRoleRepo,
	NewRefRolePermissionRepo,
	NewPermissionRepo,
	NewAppRepo,
	NewSystemConfigRepo,
	NewRefUserGroupRepo,
)

