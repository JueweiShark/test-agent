package v2

import (
	"context"
	"time"
)

// RefUserGroup 用户-企业 关系 (复用 ref_user_group 表)
// 一个 (user_id, group_id) 在未软删状态下唯一, 代表员工在某企业的成员关系.
// 其下的多角色通过 ref_user_role 表关联 (一个 ref_user_group 可对应多条 ref_user_role).
type RefUserGroup struct {
	Id              *int64     `mapper:"id"`
	UserId          int64      `mapper:"userId"`
	GroupId         int64      `mapper:"groupId"`
	Status          int64      `mapper:"status"`    // 0:禁用 1:启用
	Remarks         string     `mapper:"remarks"`   // 备注信息
	CreatedBy       int64      `mapper:"createdBy"`
	CreatedTime     *time.Time `mapper:"createdTime"`
	UpdatedBy       *int64     `mapper:"updatedBy"`
	UpdatedTime     *time.Time `mapper:"updatedTime"`
	DeletedUniqueId int64      `mapper:"isDeleted"`
	DeletedAt       *time.Time `mapper:"deletedTime"`

	// 列表展示用快照字段 (由 FindPageByCond 通过 JOIN user/group 填充, 其他方法不填)
	Username    string `mapper:"username"`    // user.username
	Nickname    string `mapper:"nickname"`    // user.nickname
	GroupName   string `mapper:"groupName"`   // group.name
	GroupCode   string `mapper:"groupCode"`   // group.group_code
}

// RefUserGroupRepo v2 用户-企业关系仓储
type RefUserGroupRepo interface {
	// Create 创建用户-企业关系 (一次加入)
	Create(ctx context.Context, r *RefUserGroup) error
	// FindByUserIdAndGroupId 查询单条关系 (用户在某企业的成员关系)
	FindByUserIdAndGroupId(ctx context.Context, userId, groupId int64) (*RefUserGroup, error)
	// FindByGroupId 查企业下所有员工关系
	FindByGroupId(ctx context.Context, groupId int64) ([]*RefUserGroup, error)
	// FindByUserIdsAndGroupId 批量查询多个用户在同一企业的关系 (用于列表填充 status/remarks)
	FindByUserIdsAndGroupId(ctx context.Context, userIds []int64, groupId int64) ([]*RefUserGroup, error)
	// FindPageByCond 员工-企业关系分页查询 (一个企业一个员工对应一条记录维度)
	// 支持groupId/groupId=null(跨企业)/用户名模糊/姓名模糊/关联企业名称模糊/status过滤
	// 返回的每条 RefUserGroup 关系同时携带 user/group 维度的字段 (用于列表展示)
	FindPageByCond(ctx context.Context, cond UserCondition, offset, limit int) ([]*RefUserGroup, int64, error)
	// Exists 判断用户是否已加入企业 (未软删)
	Exists(ctx context.Context, userId, groupId int64) (bool, error)
	// Update 更新员工的状态与备注 (status/remarks), 同时刷新 update_by/update_time
	Update(ctx context.Context, r *RefUserGroup) error
	// Delete 软删用户-企业关系 (同时应级联软删对应的 ref_user_role, 由 biz 层编排)
	Delete(ctx context.Context, userId, groupId int64) error
}
