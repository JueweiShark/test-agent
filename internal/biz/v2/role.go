package v2

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"

	"wuwen.com/wwuc/internal/biz/common"
)

// roleAdminId 超管角色 id (硬编码, Role.Id==1 即超管)
const roleAdminId int64 = 1

// Role 角色 (复用 role 表)
type Role struct {
	Id              *int64     `mapper:"id"`
	Name            string     `mapper:"name"`
	Code            string     `mapper:"code"`
	Remarks         string     `mapper:"remarks"`
	GroupId         int64      `mapper:"groupId"`
	App             int64      `mapper:"app"`
	CreateBy        int64      `mapper:"createBy"`
	CreateAt        *time.Time `mapper:"createAt"`
	UpdateBy        int64      `mapper:"updateBy"`
	UpdateAt        *time.Time `mapper:"updateAt"`
	DeletedUniqueId int64      `mapper:"isDelete"`
	DeletedAt       *time.Time `mapper:"deletedAt"`
	System          int64      `mapper:"System"`
}

type QueryRoleCondition struct {
	GroupId int64
	GroupName string
	Name    string
}


// RoleRepo v2 角色仓储
type RoleRepo interface {
	Create(ctx context.Context, r *Role) error
	Update(ctx context.Context, r *Role) error
	Delete(ctx context.Context, userId, roleId int64) error
	FindById(ctx context.Context, id int64) (*Role, error)
	FindByIds(ctx context.Context, ids []int64) ([]*Role, error)
	FindByPage(ctx context.Context, cond QueryRoleCondition, offset, limit int) ([]*Role, int64, error)
	ExistsByName(ctx context.Context, groupId int64, name string) (bool, error)
	// UpdateCode 事务内更新角色 code (code = fmt.Sprintf("role%05d", id))
	UpdateCode(ctx context.Context, roleId int64, code string) error
}

// RefUserRole 用户-角色-企业 绑定 (复用 ref_user_role 表)
type RefUserRole struct {
	Id              *int64     `mapper:"id"`
	UserId          int64      `mapper:"userId"`
	RoleId          int64      `mapper:"roleId"`
	GroupId         int64      `mapper:"groupId"`
	CreateBy        int64      `mapper:"createBy"`
	CreateAt        *time.Time `mapper:"createAt"`
	UpdateBy        int64      `mapper:"updateBy"`
	UpdateAt        *time.Time `mapper:"updateAt"`
	DeletedUniqueId int64      `mapper:"isDelete"`
	DeletedAt       *time.Time `mapper:"deletedAt"`
}

// RefUserRoleRepo v2 用户角色关系仓储
type RefUserRoleRepo interface {
	FindByUserId(ctx context.Context, userId int64) ([]*RefUserRole, error)
	FindByUserIdAndGroupId(ctx context.Context, userId, groupId int64) ([]*RefUserRole, error)
	FindByGroupId(ctx context.Context, groupId int64) ([]*RefUserRole, error)
	FindByUserIdsAndGroupId(ctx context.Context, userIds []int64, groupId int64) ([]*RefUserRole, error)
	Exists(ctx context.Context, userId, roleId, groupId int64) (bool, error)
	CountByRoleId(ctx context.Context, roleId int64) (int64, error)
	Create(ctx context.Context, r *RefUserRole) error
	Delete(ctx context.Context, userId, roleId, groupId int64) error
}

// RefRolePermission 角色-权限 绑定 (复用 ref_role_permission 表)
type RefRolePermission struct {
	Id        *int64     `mapper:"id"`
	RoleId    int64      `mapper:"roleId"`
	PermId    int64      `mapper:"permId"`
	CreateBy  int64      `mapper:"createBy"`
	CreateAt  *time.Time `mapper:"createAt"`
	UpdateBy  int64      `mapper:"updateBy"`
	UpdateAt  *time.Time `mapper:"updateAt"`
	DeletedAt *time.Time `mapper:"deletedAt"`
}

// RefRolePermissionRepo v2 角色权限关系仓储
type RefRolePermissionRepo interface {
	FindByRoleIds(ctx context.Context, roleIds []int64) ([]*RefRolePermission, error)
	DeleteByRoleId(ctx context.Context, userId, roleId int64) error
	CreateBatch(ctx context.Context, refs []*RefRolePermission) error
}

// RoleUsecase 角色业务用例
type RoleUsecase struct {
	log  *log.Helper
	tm   common.Transaction
	rr   RoleRepo
	rur  RefUserRoleRepo
	rpr  RefRolePermissionRepo
	pr   PermissionRepo
	ur   UserRepo
	tr   GroupRepo
	rgpr RefGroupPermissionRepo
	rug  RefUserGroupRepo
}

// NewRoleUsecase .
func NewRoleUsecase(logger log.Logger, tm common.Transaction,
	rr RoleRepo, rur RefUserRoleRepo, rpr RefRolePermissionRepo,
	pr PermissionRepo, ur UserRepo, tr GroupRepo, rgpr RefGroupPermissionRepo,
	rug RefUserGroupRepo) *RoleUsecase {
	return &RoleUsecase{
		log:  log.NewHelper(logger),
		tm:   tm,
		rr:   rr,
		rur:  rur,
		rpr:  rpr,
		pr:   pr,
		ur:   ur,
		tr:   tr,
		rgpr: rgpr,
		rug:  rug,
	}
}

// CreateRole 为指定企业建角色并绑定初始权限 (事务内: insert → 取 id → 生成 code → 更新 code → 绑权限)
func (uc *RoleUsecase) CreateRole(ctx context.Context, r *Role, permIds []int64) *errors.Error {
	if ok, _ := uc.rr.ExistsByName(ctx, r.GroupId, r.Name); ok {
		return errRoleNameExists
	}
	dedup := dedupPermissionIds(permIds)
	// scope 校验放在事务外, 避免 403 被遮蔽为 500
	if err := uc.checkPermissionScope(ctx, r.GroupId, dedup); err != nil {
		return err
	}
	err := uc.tm.InTx(ctx, func(ctx context.Context) error {
		if err := uc.rr.Create(ctx, r); err != nil {
			if err == gorm.ErrDuplicatedKey {
				return errRoleNameExists
			}
			return err
		}
		if r.Id == nil {
			return errInternalError
		}
		// 生成 code = role + YYMMdd + %05d
		code := fmt.Sprintf("role%s%05d", time.Now().Format("060102"), *r.Id)
		if err := uc.rr.UpdateCode(ctx, *r.Id, code); err != nil {
			return err
		}
		r.Code = code
		// 绑定初始权限 (新角色无旧绑定, 直接批量插)
		if len(dedup) > 0 {
			if err := uc.overwriteRolePermissions(ctx, r.CreateBy, *r.Id, dedup); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		uc.log.WithContext(ctx).Errorf("create role failed: %v", err)
		if err == errRoleNameExists {
			return errRoleNameExists
		}
		return errInternalError
	}
	return nil
}

// UpdateRole 编辑角色并全量覆盖权限 (System=1 不可改名且权限固定)
// permission_ids 语义: 非系统角色全量覆盖, 空=清空; 系统角色权限固定, 跳过且禁止显式重绑
func (uc *RoleUsecase) UpdateRole(ctx context.Context, r *Role, permIds []int64) *errors.Error {
	old, err := uc.rr.FindById(ctx, *r.Id)
	if err != nil {
		return errRoleNotFound
	}
	// 系统角色: 名称不可改, 权限固定 (禁止显式重绑)
	if old.System == 1 {
		if r.Name != "" && r.Name != old.Name {
			return errRoleSystemCannotEdit
		}
		if len(permIds) > 0 {
			return errRoleSystemCannotEdit
		}
	}
	if r.Name != "" && r.Name != old.Name {
		if ok, _ := uc.rr.ExistsByName(ctx, old.GroupId, r.Name); ok {
			return errRoleNameExists
		}
	}
	// 保留 system/group_id/code 等不可变字段
	r.System = old.System
	r.GroupId = old.GroupId
	r.Code = old.Code

	// 系统角色: 仅更新字段, 不触碰权限
	if old.System == 1 {
		if err := uc.rr.Update(ctx, r); err != nil {
			uc.log.WithContext(ctx).Errorf("update role failed: %v", err)
			return errInternalError
		}
		return nil
	}
	// 非系统角色: scope 校验 + 事务内更新字段并全量覆盖权限
	dedup := dedupPermissionIds(permIds)
	if err := uc.checkPermissionScope(ctx, r.GroupId, dedup); err != nil {
		return err
	}
	err = uc.tm.InTx(ctx, func(ctx context.Context) error {
		if err := uc.rr.Update(ctx, r); err != nil {
			return err
		}
		return uc.overwriteRolePermissions(ctx, r.UpdateBy, *r.Id, dedup)
	})
	if err != nil {
		uc.log.WithContext(ctx).Errorf("update role failed: %v", err)
		return errInternalError
	}
	return nil
}

// DeleteRole 删除角色 (System=1 不可删; 仍有员工绑定则 409 RoleInUse).
// 事务内: 软删角色 + 软删该角色的 ref_role_permission 绑定 (避免悬空引用).
func (uc *RoleUsecase) DeleteRole(ctx context.Context, userId, roleId int64) *errors.Error {
	r, err := uc.rr.FindById(ctx, roleId)
	if err != nil {
		return errRoleNotFound
	}
	if r.System == 1 {
		return errRoleSystemCannotEdit
	}
	// 若尚有员工绑定该角色, 则不允许删除 (与 v1 行为一致)
	if cnt, err := uc.rur.CountByRoleId(ctx, roleId); err != nil {
		uc.log.WithContext(ctx).Errorf("count role users failed, roleId=%d err=%v", roleId, err)
		return errInternalError
	} else if cnt > 0 {
		return errRoleInUse
	}
	if err := uc.tm.InTx(ctx, func(ctx context.Context) error {
		if err := uc.rr.Delete(ctx, userId, roleId); err != nil {
			return err
		}
		// 级联软删该角色的权限绑定 (ref_role_permission), 避免悬空引用
		if err := uc.rpr.DeleteByRoleId(ctx, userId, roleId); err != nil {
			return err
		}
		return nil
	}); err != nil {
		uc.log.WithContext(ctx).Errorf("delete role failed, roleId=%d err=%v", roleId, err)
		return errInternalError
	}
	return nil
}

// ListRolesByGroup 企业下角色列表
func (uc *RoleUsecase) ListRolesByGroup(ctx context.Context, cond QueryRoleCondition, offset, limit int) ([]*Role, int64, *errors.Error) {
	roles, total, err := uc.rr.FindByPage(ctx, cond, offset, limit)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("list roles failed: %v", err)
		return nil, 0, errInternalError
	}
	return roles, total, nil
}

// FindGroupsByIds 批量查企业 (供角色列表填充 groupName 用)
func (uc *RoleUsecase) FindGroupsByIds(ctx context.Context, groupIds []int64) ([]*Group, *errors.Error) {
	if len(groupIds) == 0 {
		return nil, nil
	}
	groups, err := uc.tr.FindByIds(ctx, groupIds)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("find groups by ids failed: %v", err)
		return nil, errInternalError
	}
	return groups, nil
}

// FindPermissionsByRoleIds 批量查角色绑定的菜单权限 (供角色列表填充 permissions 用)
// 内部按 role_id 聚合, 返回 roleId -> []Permission 映射
func (uc *RoleUsecase) FindPermissionsByRoleIds(ctx context.Context, roleIds []int64) (map[int64][]*Permission, *errors.Error) {
	if len(roleIds) == 0 {
		return nil, nil
	}
	refs, err := uc.rpr.FindByRoleIds(ctx, roleIds)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("find ref_role_permission by role ids failed: %v", err)
		return nil, errInternalError
	}
	if len(refs) == 0 {
		return make(map[int64][]*Permission), nil
	}
	permIdSet := make(map[int64]bool, len(refs))
	for _, ref := range refs {
		if ref.PermId > 0 && !permIdSet[ref.PermId] {
			permIdSet[ref.PermId] = true
		}
	}
	permIds := make([]int64, 0, len(permIdSet))
	for pid := range permIdSet {
		permIds = append(permIds, pid)
	}
	permMap := make(map[int64]*Permission, len(permIds))
	if len(permIds) > 0 {
		perms, perr := uc.pr.FindByIds(ctx, permIds)
		if perr != nil {
			uc.log.WithContext(ctx).Errorf("find permissions by ids failed: %v", perr)
			return nil, errInternalError
		}
		for _, p := range perms {
			if p != nil {
				permMap[p.Id] = p
			}
		}
	}
	result := make(map[int64][]*Permission, len(roleIds))
	for _, ref := range refs {
		if p, ok := permMap[ref.PermId]; ok && p != nil {
			result[ref.RoleId] = append(result[ref.RoleId], p)
		}
	}
	return result, nil
}

// AssignRoleToUser 超管给企业加员工 (一次加入, 多角色绑定).
// - 写 ref_user_group (status/remarks, 重复加入返回 409 UserAlreadyInGroup)
// - 写多条 ref_user_role (roleIds 须全部 ⊆ 该企业, 否则 403 RoleNotInGroup)
// 事务内完成; status<=0 时默认 1 启用.
func (uc *RoleUsecase) AssignRoleToUser(ctx context.Context, userId, groupId int64, roleIds []int64, status int32, remarks string, operatorId int64) *errors.Error {
	// 1. 重复加入校验 (ref_user_group 未软删)
	if ok, _ := uc.rug.Exists(ctx, userId, groupId); ok {
		return errUserAlreadyInGroup
	}
	// 2. 角色合法性校验: 须全部属于该企业
	dedup := dedupPermissionIds(roleIds) // 复用: 去重并过滤 <=0
	if err := uc.checkRolesBelongToGroup(ctx, groupId, dedup); err != nil {
		return err
	}
	// 3. status 默认值
	st := int64(status)
	if st == 0 {
		st = 1
	}
	err := uc.tm.InTx(ctx, func(ctx context.Context) error {
		// 写 ref_user_group
		if err := uc.rug.Create(ctx, &RefUserGroup{
			UserId:    userId,
			GroupId:   groupId,
			Status:    st,
			Remarks:   remarks,
			CreatedBy: operatorId,
		}); err != nil {
			return err
		}
		// 写多条 ref_user_role
		for _, rid := range dedup {
			if err := uc.rur.Create(ctx, &RefUserRole{
				UserId:   userId,
				RoleId:   rid,
				GroupId:  groupId,
				CreateBy: operatorId,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		uc.log.WithContext(ctx).Errorf("assign role to user failed: %v", err)
		return errInternalError
	}
	return nil
}

// UpdateGroupEmployee 修改企业员工的角色/状态/备注.
// - roleIds 全量覆盖该用户在该企业的角色绑定 (空=清空角色, 但不删 ref_user_group)
// - status/remarks 更新到 ref_user_group
// - 前置: 员工必须已加入企业 (否则 404 UserNotInGroup)
func (uc *RoleUsecase) UpdateGroupEmployee(ctx context.Context, userId, groupId int64, roleIds []int64, status int32, remarks string, operatorId int64) *errors.Error {
	// 1. 员工必须已加入企业
	exist, err := uc.rug.Exists(ctx, userId, groupId)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("check user in group failed: %v", err)
		return errInternalError
	}
	if !exist {
		return errUserNotInGroup
	}
	// 2. 角色合法性校验
	dedup := dedupPermissionIds(roleIds)
	if err := uc.checkRolesBelongToGroup(ctx, groupId, dedup); err != nil {
		return err
	}
	// 3. 事务内更新 ref_user_group + 全量覆盖 ref_user_role
	err = uc.tm.InTx(ctx, func(ctx context.Context) error {
		// 更新 ref_user_group: status + remarks
		updatedBy := operatorId
		if err := uc.rug.Update(ctx, &RefUserGroup{
			UserId:    userId,
			GroupId:   groupId,
			Status:    int64(status),
			Remarks:   remarks,
			UpdatedBy: &updatedBy,
		}); err != nil {
			return err
		}
		// 全量覆盖该用户在该企业的 ref_user_role:
		// 先软删旧绑定, 再批量插入新的
		oldRefs, fErr := uc.rur.FindByUserIdAndGroupId(ctx, userId, groupId)
		if fErr != nil {
			return fErr
		}
		for _, ref := range oldRefs {
			if err := uc.rur.Delete(ctx, userId, ref.RoleId, groupId); err != nil {
				return err
			}
		}
		for _, rid := range dedup {
			if err := uc.rur.Create(ctx, &RefUserRole{
				UserId:   userId,
				RoleId:   rid,
				GroupId:  groupId,
				CreateBy: operatorId,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		uc.log.WithContext(ctx).Errorf("update group employee failed: %v", err)
		return errInternalError
	}
	return nil
}

// checkRolesBelongToGroup 校验 roleIds 全部属于该企业 (role.group_id == groupId 且未软删)
func (uc *RoleUsecase) checkRolesBelongToGroup(ctx context.Context, groupId int64, roleIds []int64) *errors.Error {
	if len(roleIds) == 0 {
		return nil
	}
	roles, err := uc.rr.FindByIds(ctx, roleIds)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("find roles by ids failed: %v", err)
		return errInternalError
	}
	if len(roles) != len(roleIds) {
		return errRoleNotFound
	}
	for _, r := range roles {
		if r.GroupId != groupId {
			return errRoleNotInGroup
		}
	}
	return nil
}

// RemoveRoleFromUser 移除用户角色绑定
func (uc *RoleUsecase) RemoveRoleFromUser(ctx context.Context, userId, roleId, groupId int64) *errors.Error {
	if err := uc.rur.Delete(ctx, userId, roleId, groupId); err != nil {
		uc.log.WithContext(ctx).Errorf("remove role failed: %v", err)
		return errInternalError
	}
	return nil
}

// ListUsersByGroup 企业下员工列表 (含该企业下的角色 + 员工关系 status/remarks).
// 返回分页后的用户 + 这些用户在该企业的角色绑定 + 这些用户的 ref_user_group 关系 + 总数.
func (uc *RoleUsecase) ListUsersByGroup(ctx context.Context, groupId int64, cond UserCondition, offset, limit int) ([]*User, []*RefUserRole, []*RefUserGroup, int64, *errors.Error) {
	users, total, err := uc.ur.FindByGroupId(ctx, groupId, cond, offset, limit)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("list users by group failed: %v", err)
		return nil, nil, nil, 0, errInternalError
	}
	if len(users) == 0 {
		return users, nil, nil, total, nil
	}
	userIds := make([]int64, 0, len(users))
	for _, u := range users {
		if u.Id != nil {
			userIds = append(userIds, *u.Id)
		}
	}
	refs, err := uc.rur.FindByUserIdsAndGroupId(ctx, userIds, groupId)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("find user roles failed: %v", err)
		return nil, nil, nil, 0, errInternalError
	}
	ugs, err := uc.rug.FindByUserIdsAndGroupId(ctx, userIds, groupId)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("find user group relations failed: %v", err)
		return nil, nil, nil, 0, errInternalError
	}
	return users, refs, ugs, total, nil
}

// ListGroupEmployees 员工-企业关系分页查询 (一条记录 = 一个员工在某企业的成员关系).
// 支持跨企业搜索 (cond.GroupId=nil) 与单企业搜索; 关联企业名称/用户名/姓名模糊; 状态过滤.
// 返回的 RefUserGroup 携带 user/group 快照字段 (Username/Nickname/GroupName/GroupCode) 用于列表展示.
func (uc *RoleUsecase) ListGroupEmployees(ctx context.Context, cond UserCondition, offset, limit int) ([]*RefUserGroup, int64, *errors.Error) {
	ugs, total, err := uc.rug.FindPageByCond(ctx, cond, offset, limit)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("list group employees failed: %v", err)
		return nil, 0, errInternalError
	}
	return ugs, total, nil
}

// ListUserRolesByGroup 查某用户在某企业的角色
func (uc *RoleUsecase) ListUserRolesByGroup(ctx context.Context, userId, groupId int64) ([]*RefUserRole, *errors.Error) {
	refs, err := uc.rur.FindByUserIdAndGroupId(ctx, userId, groupId)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("list user roles failed: %v", err)
		return nil, errInternalError
	}
	return refs, nil
}

// FindUserRolesByGroupId 查某企业下所有用户的角色绑定 (用于列表批量填充角色)
func (uc *RoleUsecase) FindUserRolesByGroupId(ctx context.Context, groupId int64) ([]*RefUserRole, *errors.Error) {
	refs, err := uc.rur.FindByGroupId(ctx, groupId)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("find user roles by group failed: %v", err)
		return nil, errInternalError
	}
	return refs, nil
}

// dedupPermissionIds 去重并过滤掉 <=0 的权限 id
func dedupPermissionIds(permIds []int64) []int64 {
	seen := make(map[int64]bool, len(permIds))
	dedup := make([]int64, 0, len(permIds))
	for _, id := range permIds {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		dedup = append(dedup, id)
	}
	return dedup
}

// checkPermissionScope 校验 dedup 必须 ⊆ group_id 的企业 scope (ref_group_permission)
// 权限继承: ref_role_permission (子集) 必须 ⊆ ref_group_permission (父域 = 企业 scope)
func (uc *RoleUsecase) checkPermissionScope(ctx context.Context, groupId int64, dedup []int64) *errors.Error {
	if len(dedup) == 0 {
		return nil
	}
	groupPerms, gErr := uc.rgpr.FindByGroupId(ctx, groupId)
	if gErr != nil {
		uc.log.WithContext(ctx).Errorf("find group permissions for scope check failed: %v", gErr)
		return errInternalError
	}
	scopeSet := make(map[int64]bool, len(groupPerms))
	for _, gp := range groupPerms {
		scopeSet[gp.PermissionId] = true
	}
	for _, pid := range dedup {
		if !scopeSet[pid] {
			return errPermissionOutOfScope
		}
	}
	return nil
}

// overwriteRolePermissions 全量覆盖角色权限 (软删旧 + 批量插新, 空 dedup=仅删=清空)
// 不含 scope 校验, 调用方须在事务外先校验. 复用 ctx 中的 tx (由调用方 InTx 包裹).
func (uc *RoleUsecase) overwriteRolePermissions(ctx context.Context, userId, roleId int64, dedup []int64) error {
	if err := uc.rpr.DeleteByRoleId(ctx, userId, roleId); err != nil {
		return err
	}
	if len(dedup) == 0 {
		return nil
	}
	refs := make([]*RefRolePermission, 0, len(dedup))
	for _, pid := range dedup {
		refs = append(refs, &RefRolePermission{RoleId: roleId, PermId: pid, CreateBy: userId})
	}
	return uc.rpr.CreateBatch(ctx, refs)
}

// FindRolesByIds 批量查角色 (供 auth 聚合用)
func (uc *RoleUsecase) FindRolesByIds(ctx context.Context, roleIds []int64) ([]*Role, *errors.Error) {
	if len(roleIds) == 0 {
		return nil, nil
	}
	roles, err := uc.rr.FindByIds(ctx, roleIds)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("find roles by ids failed: %v", err)
		return nil, errInternalError
	}
	return roles, nil
}

// FindUserRoles 查用户所有角色绑定 (供 auth 聚合用)
func (uc *RoleUsecase) FindUserRoles(ctx context.Context, userId int64) ([]*RefUserRole, *errors.Error) {
	refs, err := uc.rur.FindByUserId(ctx, userId)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("find user roles failed: %v", err)
		return nil, errInternalError
	}
	return refs, nil
}

// IsSuperAdmin 判断角色 id 列表是否含超管
func IsSuperAdmin(roleIds []int64) bool {
	for _, id := range roleIds {
		if id == roleAdminId {
			return true
		}
	}
	return false
}
