package user

import (
    "context"
)

type RefUserRole struct {
    Id       *int64 `mapper:"id"`
    UserId   int64  `mapper:"userId"`
    RoleId   int64  `mapper:"roleId"`
    GroupId  int64  `mapper:"groupId"`
    CreateBy int64  `mapper:"createBy"`
}

type RefUserRoleRepo interface {
    Create(ctx context.Context, refUserGroups *RefUserRole) error
    BatchCreate(ctx context.Context, refUserGroups []*RefUserRole) error
    FindByUserId(ctx context.Context, userId, groupId int64) ([]*RefUserRole, error)
    FindByUserIds(ctx context.Context, userId []int64, groupId int64) ([]*RefUserRole, error)
    FindByUserIdsAndAppId(ctx context.Context, userId []int64, appId int64) ([]*RefUserRole, error)
    Delete(ctx context.Context, userId int64, refId int64) error
    BatchDelete(ctx context.Context, id, delUserId, groupId int64) error
    BatchDeleteByUserId(ctx context.Context, deletedBy, userId int64) error
    FindByRole(ctx context.Context, roleId int64) ([]*RefUserRole, error)
}
