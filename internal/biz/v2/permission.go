package v2

import (
	"context"
	"time"
)

// Permission 权限/菜单节点 (v2 复用 permission 表, 新增 menu_type/route_path/icon/sort/app/dependence_children)
type Permission struct {
	Id                 int64      `mapper:"id"`
	Domain             string     `mapper:"domain"`
	Code               int64      `mapper:"permissionCode"`
	Type               uint8      `mapper:"type"`
	MenuType           *uint8     `mapper:"menuType"`
	RoutePath          string     `mapper:"routePath"`
	Icon               string     `mapper:"icon"`
	Sort               int32      `mapper:"sort"`
	Hidden             *uint8     `mapper:"hidden"`     // 1显示 0隐藏, nil=不改
	App                int64      `mapper:"app"`
	Name               string     `mapper:"name"`
	Path               string     `mapper:"path"`
	Act                *string    `mapper:"act"`
	Description        *string    `mapper:"description"`
	DependenceChildren []int64    `mapper:"dependenceChildren"`
	DependenceParents  int64      `mapper:"dependenceParents"`
	CreateBy           int64      `mapper:"createBy"`
	CreateTime         *time.Time `mapper:"createTime"`
	UpdateBy           int64      `mapper:"updateBy"`
	UpdateTime         *time.Time `mapper:"updateTime"`
	DeletedUniqueId    int64      `mapper:"deletedId"`
	DeleteTime         *time.Time `mapper:"deleteTime"`
}

// PermissionRepo v2 权限/菜单仓储
type PermissionRepo interface {
	FindAll(ctx context.Context) ([]*Permission, error)
	FindById(ctx context.Context, id int64) (*Permission, error)
	FindByIds(ctx context.Context, ids []int64) ([]*Permission, error)
	FindByRoleIds(ctx context.Context, roleIds []int64) ([]*Permission, error)
	// FindEffectiveByUserId 查用户的有效权限 (3表 join: ref_user_role ⨝ ref_role_permission ⨝ ref_group_permission)
	// 权限继承: 有效权限 = 角色权限 ∩ 企业 scope
	FindEffectiveByUserId(ctx context.Context, userId int64) ([]*Permission, error)
	// FindEffectiveByUserIdAndGroupId 查用户在指定企业的有效权限 (3表 join, 固定 group_id)
	// 权限继承: 有效权限 = 角色权限 ∩ 企业 scope (限本企业)
	FindEffectiveByUserIdAndGroupId(ctx context.Context, userId, groupId int64) ([]*Permission, error)
	FindByApp(ctx context.Context, app int64, hidden bool) ([]*Permission, error)
	// FindByAppAndGroupId 按 app 与企业 scope 过滤的权限列表 (join ref_group_permission)
	FindByAppAndGroupId(ctx context.Context, app, groupId int64, hidden bool) ([]*Permission, error)
	// FindAppIdsByGroupIds 批量查多个企业 scope 内的去重 app_id, 返回 groupId -> []app_id 映射
	// 用于 GroupInfo.Apps 推导: 企业拥有某 app 的至少一个菜单权限即视为可用该产品
	FindAppIdsByGroupIds(ctx context.Context, groupIds []int64) (map[int64][]int64, error)
	Create(ctx context.Context, p *Permission) error
	Update(ctx context.Context, p *Permission) error
	Delete(ctx context.Context, userId int64, id int64) error
	CountChildren(ctx context.Context, parentId int64) (int64, error)
}
