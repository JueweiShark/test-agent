package v2

import "github.com/google/wire"

// ProviderSet v2 biz 层 provider 集合
var ProviderSet = wire.NewSet(
	NewAuthUsecase,
	NewGroupUsecase,
	NewRoleUsecase,
	NewMenuUsecase,
	NewUserUsecase,
	NewSystemUsecase,
)
