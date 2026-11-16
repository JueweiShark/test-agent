package v2

import "github.com/google/wire"

// ProviderSet v2 service 层 provider 集合
var ProviderSet = wire.NewSet(
	NewAuthV2Service,
	NewGroupV2Service,
	NewRoleV2Service,
	NewMenuV2Service,
	NewUserV2Service,
	NewSystemV2Service,
)
