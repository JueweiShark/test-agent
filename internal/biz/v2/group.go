package v2

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"

	"wuwen.com/wwuc/internal/biz/common"
)

// Group 租户企业 (复用 group 表, 不读写 managers/app)
type Group struct {
	Id              *int64     `mapper:"id"`
	System          int8       `mapper:"system"`
	Name            string     `mapper:"groupName"`
	GroupCode       string     `mapper:"groupCode"`
	GroupCreatedAt  *time.Time `mapper:"groupCreatedAt"`
	Status          int8       `mapper:"status"`
	Remarks         string     `mapper:"remarks"`
	CreatedBy       int64      `mapper:"createdBy"`
	CreatedAt       *time.Time `mapper:"createdAt"`
	UpdatedBy       int64      `mapper:"updatedBy"`
	UpdatedAt       *time.Time `mapper:"updatedAt"`
	DeletedUniqueId int64      `mapper:"deleteUniqueId"`
	DeletedAt       *time.Time `mapper:"deletedAt"`
}

// GroupCondition 租户分页查询条件
type GroupCondition struct {
	Ids        []int64
	Name      *string
	GroupCode *string
	Status    *int32
}

// GroupRepo 租户仓储
type GroupRepo interface {
	Create(ctx context.Context, t *Group) error
	Update(ctx context.Context, t *Group) error
	Delete(ctx context.Context, userId, groupId int64) error
	FindById(ctx context.Context, id int64) (*Group, error)
	FindByIds(ctx context.Context, ids []int64) ([]*Group, error)
	FindByPage(ctx context.Context, cond GroupCondition, offset, limit int) ([]*Group, int64, error)
	ExistsByName(ctx context.Context, name string) (bool, error)
	ExistsByCode(ctx context.Context, code string) (bool, error)
}

// RefGroupPermission 企业权限范围关系 (父域: 企业最大可见菜单权限集合)
type RefGroupPermission struct {
	Id              *int64     `mapper:"id"`
	GroupId         int64      `mapper:"groupId"`
	PermissionId    int64      `mapper:"permissionId"`
	CreatedBy       int64      `mapper:"createdBy"`
	CreatedAt       *time.Time `mapper:"createdAt"`
	UpdatedBy       int64      `mapper:"updatedBy"`
	UpdatedAt       *time.Time `mapper:"updatedAt"`
	DeletedUniqueId int64      `mapper:"deleteUniqueId"`
	DeletedAt       *time.Time `mapper:"deletedAt"`
}

// RefGroupPermissionRepo 企业权限范围仓储
type RefGroupPermissionRepo interface {
	FindByGroupId(ctx context.Context, groupId int64) ([]*RefGroupPermission, error)
	FindByGroupIds(ctx context.Context, groupIds []int64) ([]*RefGroupPermission, error)
	Create(ctx context.Context, r *RefGroupPermission) error
	DeleteByGroupIdAndPermissionId(ctx context.Context, groupId, permissionId, userId int64) error
	// CascadeDeleteRolePermissions 级联软删越界的 ref_role_permission 记录
	// 当企业 scope 收缩时, 角色-权限关联中超出新 scope 的记录需被软删
	CascadeDeleteRolePermissions(ctx context.Context, groupId int64, permissionIds []int64, userId int64) error
}

// GroupUsecase 租户业务用例
type GroupUsecase struct {
	log  *log.Helper
	tm   common.Transaction
	tr   GroupRepo
	rgpr RefGroupPermissionRepo
	pr   PermissionRepo
	ar   AppRepo
}

// NewGroupUsecase .
func NewGroupUsecase(logger log.Logger, tm common.Transaction, tr GroupRepo, rgpr RefGroupPermissionRepo, pr PermissionRepo, ar AppRepo) *GroupUsecase {
	return &GroupUsecase{
		log:  log.NewHelper(logger),
		tm:   tm,
		tr:   tr,
		rgpr: rgpr,
		pr:   pr,
		ar:   ar,
	}
}

// CreateGroup 超管创建企业
func (uc *GroupUsecase) CreateGroup(ctx context.Context, t *Group) *errors.Error {
	if ok, _ := uc.tr.ExistsByName(ctx, t.Name); ok {
		return errGroupNameExists
	}
	if t.GroupCode != "" {
		if ok, _ := uc.tr.ExistsByCode(ctx, t.GroupCode); ok {
			return errGroupCodeExists
		}
	}
	t.System = 1
	t.Status = 1
	now := time.Now()
	t.GroupCreatedAt = &now
	if err := uc.tr.Create(ctx, t); err != nil {
		uc.log.WithContext(ctx).Errorf("create group failed: %v", err)
		if err == gorm.ErrDuplicatedKey {
			return errGroupNameExists
		}
		return errInternalError
	}
	return nil
}

// UpdateGroup 编辑企业
func (uc *GroupUsecase) UpdateGroup(ctx context.Context, t *Group) *errors.Error {
	old, err := uc.tr.FindById(ctx, *t.Id)
	if err != nil {
		return errGroupNotFound
	}
	if t.Name != "" && t.Name != old.Name {
		if ok, _ := uc.tr.ExistsByName(ctx, t.Name); ok {
			return errGroupNameExists
		}
	}
	if t.GroupCode != "" && t.GroupCode != old.GroupCode {
		if ok, _ := uc.tr.ExistsByCode(ctx, t.GroupCode); ok {
			return errGroupCodeExists
		}
	}
	// status=0 视为未变更, 避免 proto3 默认值误停用企业
	if t.Status == 0 {
		t.Status = old.Status
	}
	if err := uc.tr.Update(ctx, t); err != nil {
		uc.log.WithContext(ctx).Errorf("update group failed: %v", err)
		return errInternalError
	}
	return nil
}

// DeleteGroup 软删企业
func (uc *GroupUsecase) DeleteGroup(ctx context.Context, userId, groupId int64) *errors.Error {
	if _, err := uc.tr.FindById(ctx, groupId); err != nil {
		return errGroupNotFound
	}
	if err := uc.tr.Delete(ctx, userId, groupId); err != nil {
		uc.log.WithContext(ctx).Errorf("delete group failed: %v", err)
		return errInternalError
	}
	return nil
}

// GetGroup 企业详情
func (uc *GroupUsecase) GetGroup(ctx context.Context, groupId int64) (*Group, *errors.Error) {
	t, err := uc.tr.FindById(ctx, groupId)
	if err != nil {
		return nil, errGroupNotFound
	}
	return t, nil
}

// ListGroups 企业分页列表
func (uc *GroupUsecase) ListGroups(ctx context.Context, cond GroupCondition, offset, limit int) ([]*Group, int64, *errors.Error) {
	groups, total, err := uc.tr.FindByPage(ctx, cond, offset, limit)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("list groups failed: %v", err)
		return nil, 0, errInternalError
	}
	return groups, total, nil
}

// SetGroupPermissions 设置企业权限范围 (全量覆盖语义, 父域)
// 1. diff: 计算 removedIds = 旧scope - 新scope
// 2. 事务内: 软删 removed 的 ref_group_permission + 级联软删越界的 ref_role_permission
// 3. 新增新加入的 ref_group_permission
func (uc *GroupUsecase) SetGroupPermissions(ctx context.Context, userId, groupId int64, permissionIds []int64) *errors.Error {
	if _, err := uc.tr.FindById(ctx, groupId); err != nil {
		return errGroupNotFound
	}
	existing, err := uc.rgpr.FindByGroupId(ctx, groupId)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("find group permissions failed: %v", err)
		return errInternalError
	}
	existingSet := make(map[int64]bool, len(existing))
	for _, r := range existing {
		existingSet[r.PermissionId] = true
	}
	targetSet := make(map[int64]bool, len(permissionIds))
	for _, p := range permissionIds {
		targetSet[p] = true
	}
	// removed: 旧scope 中不在新scope 的权限 → 需级联软删角色权限
	removedIds := make([]int64, 0)
	for _, r := range existing {
		if !targetSet[r.PermissionId] {
			removedIds = append(removedIds, r.PermissionId)
		}
	}
	err = uc.tm.InTx(ctx, func(ctx context.Context) error {
		// 1. 软删移除的 ref_group_permission + 级联软删 ref_role_permission
		for _, r := range existing {
			if !targetSet[r.PermissionId] {
				if e := uc.rgpr.DeleteByGroupIdAndPermissionId(ctx, groupId, r.PermissionId, userId); e != nil {
					return e
				}
			}
		}
		if len(removedIds) > 0 {
			if e := uc.rgpr.CascadeDeleteRolePermissions(ctx, groupId, removedIds, userId); e != nil {
				return e
			}
		}
		// 2. 新增新加入的 ref_group_permission
		for _, p := range permissionIds {
			if !existingSet[p] {
				if e := uc.rgpr.Create(ctx, &RefGroupPermission{
					GroupId: groupId, PermissionId: p, CreatedBy: userId,
				}); e != nil {
					return e
				}
			}
		}
		return nil
	})
	if err != nil {
		uc.log.WithContext(ctx).Errorf("set group permissions failed: %v", err)
		return errInternalError
	}
	return nil
}

// ListGroupPermissions 查企业权限范围 (返回 PermissionId 列表)
func (uc *GroupUsecase) ListGroupPermissions(ctx context.Context, groupId int64) ([]*RefGroupPermission, *errors.Error) {
	refs, err := uc.rgpr.FindByGroupId(ctx, groupId)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("list group permissions failed: %v", err)
		return nil, errInternalError
	}
	return refs, nil
}

// ListGroupApps 查单个企业可用的 app 列表 (从企业 scope 内 permission.app 去重推导)
// 语义: 企业拥有某 app 的至少一个菜单权限即视为可用该产品 (design 4.2.4)
func (uc *GroupUsecase) ListGroupApps(ctx context.Context, groupId int64) ([]*App, *errors.Error) {
	appIdMap, err := uc.pr.FindAppIdsByGroupIds(ctx, []int64{groupId})
	if err != nil {
		uc.log.WithContext(ctx).Errorf("list group apps find app ids failed: %v", err)
		return nil, errInternalError
	}
	appIds := appIdMap[groupId]
	if len(appIds) == 0 {
		return []*App{}, nil
	}
	apps, err := uc.ar.FindByIds(ctx, appIds)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("list group apps find apps failed: %v", err)
		return nil, errInternalError
	}
	return apps, nil
}

// ListGroupAppsByGroupIds 批量查多个企业可用的 app 列表 (返回 groupId -> []*App 映射)
// 1. 一次 join 查出 groupId -> []app_id 映射 (去重)
// 2. 收集所有 appId 去重后批量查 app 表
// 3. 按 groupId 装配对应 app 列表 (保持 app_id 顺序)
func (uc *GroupUsecase) ListGroupAppsByGroupIds(ctx context.Context, groupIds []int64) (map[int64][]*App, *errors.Error) {
	out := make(map[int64][]*App, len(groupIds))
	if len(groupIds) == 0 {
		return out, nil
	}
	groupAppIds, err := uc.pr.FindAppIdsByGroupIds(ctx, groupIds)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("list group apps by group ids find app ids failed: %v", err)
		return nil, errInternalError
	}
	// 收集所有 appId 并去重
	appIdSet := make(map[int64]bool)
	allAppIds := make([]int64, 0)
	for _, appIds := range groupAppIds {
		for _, aid := range appIds {
			if !appIdSet[aid] {
				appIdSet[aid] = true
				allAppIds = append(allAppIds, aid)
			}
		}
	}
	appMap := make(map[int64]*App, len(allAppIds))
	if len(allAppIds) > 0 {
		apps, aErr := uc.ar.FindByIds(ctx, allAppIds)
		if aErr != nil {
			uc.log.WithContext(ctx).Errorf("list group apps by group ids find apps failed: %v", aErr)
			return nil, errInternalError
		}
		for _, a := range apps {
			if a != nil {
				appMap[*a.Id] = a
			}
		}
	}
	// 按 groupId 装配, 保持 app_id 在 SQL 中的排序 (group_id, app)
	for _, gid := range groupIds {
		appIds := groupAppIds[gid]
		apps := make([]*App, 0, len(appIds))
		for _, aid := range appIds {
			if a, ok := appMap[aid]; ok {
				apps = append(apps, a)
			}
		}
		out[gid] = apps
	}
	return out, nil
}
