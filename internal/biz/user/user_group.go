package user

import (
    "context"
    "time"
)

type RefUserGroup struct {
    Id          *int64     `mapper:"id"`
    UserId      int64      `mapper:"userId"`
    GroupId     int64      `mapper:"groupId"`
    CreatedBy   int64      `mapper:"createdBy"`
    CreatedTime *time.Time `mapper:"createdTime"`
}

type RefUserGroupRepo interface {
    BatchCreate(ctx context.Context, refUserGroups []*RefUserGroup) error
    BatchDelete(ctx context.Context, id int64, groups []*RefUserGroup) error
    BatchDeleteByIds(ctx context.Context, id, delUserId int64) error
    RemoveUser(ctx context.Context, updater, groupId int64, memberId []int64) error
    FindMembersByGroup(ctx context.Context, c UserCondition, id int64, offset int64, limit int64) ([]*RefUserGroup, int64, error)
    FindByUser(ctx context.Context, id int64) ([]*RefUserGroup, error)
    FindByUserIds(ctx context.Context, id []int64) ([]*RefUserGroup, error)
    FindByGroupId(ctx context.Context, id int64) ([]*RefUserGroup, error)
}
