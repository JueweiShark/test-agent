package v2

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"

	bizV2 "wuwen.com/wwuc/internal/biz/v2"
	"wuwen.com/wwuc/internal/data/common"
	"wuwen.com/wwuc/internal/data/schema"
	"wuwen.com/wwuc/util/wuwen/mapper"
)

// ===== RoleRepo =====

type roleRepo struct {
	data  *common.Data
	log   *log.Helper
	model schema.Role
}

func (r *roleRepo) Create(ctx context.Context, role *bizV2.Role) error {
	sRole := common.ConvertData[bizV2.Role, schema.Role](ctx, role)
	err := r.data.DB(ctx).Model(r.model).Create(sRole).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("create role error: %v", err)
		return err
	}
	role.Id = sRole.Id
	return nil
}

func (r *roleRepo) Update(ctx context.Context, role *bizV2.Role) error {
	sRole := common.ConvertData[bizV2.Role, schema.Role](ctx, role)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id = ?", *role.Id).
		Updates(sRole).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("update role error: %v", err)
		return err
	}
	return nil
}

func (r *roleRepo) Delete(ctx context.Context, userId, roleId int64) error {
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0 AND id = ?", roleId).
		Updates(map[string]interface{}{
			"updated_by":        userId,
			"deleted_at":        time.Now(),
			"deleted_unique_id": gorm.Expr("id"),
		}).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("delete role error: %v", err)
		return err
	}
	return nil
}

func (r *roleRepo) FindById(ctx context.Context, id int64) (*bizV2.Role, error) {
	role := new(schema.Role)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id = ?", id).
		First(role).Error
	if err != nil {
		return nil, err
	}
	return common.ConvertData[schema.Role, bizV2.Role](ctx, role), nil
}

func (r *roleRepo) FindByIds(ctx context.Context, ids []int64) ([]*bizV2.Role, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	pos := new([]*schema.Role)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id IN (?)", ids).
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find roles by ids error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.Role)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper roles err: %v", err)
		return nil, err
	}
	return *bos, nil
}

func (r *roleRepo) FindByPage(ctx context.Context, cond bizV2.QueryRoleCondition, offset, limit int) ([]*bizV2.Role, int64, error) {
	pos := new([]*schema.Role)
	// 列名加 role. 前缀: GroupName 分支会 INNER JOIN `group`, 两表均有 deleted_unique_id/name, 避免歧义 (Error 1052)
	db := r.data.DB(ctx).Model(r.model).
		Where("role.deleted_unique_id = 0").
		Order("role.id desc")
	if cond.Name != "" {
		db = common.QueryWithEscapedLike(db, "role.name", cond.Name)
	}
	if cond.GroupId != 0 {
		db = db.Where("role.group_id = ?", cond.GroupId)
	}
	if cond.GroupName != "" {
		db = db.Joins("INNER JOIN `group` ON `group`.id = role.group_id AND `group`.deleted_unique_id = 0")
		db = common.QueryWithEscapedLike(db, "`group`.name", cond.GroupName)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		r.log.WithContext(ctx).Errorf("count roles error: %v", err)
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(limit).Find(pos).Error; err != nil {
		r.log.WithContext(ctx).Errorf("find roles by group error: %v", err)
		return nil, 0, err
	}
	bos := new([]*bizV2.Role)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper roles error: %v", err)
		return nil, 0, err
	}
	return *bos, total, nil
}

func (r *roleRepo) ExistsByName(ctx context.Context, groupId int64, name string) (bool, error) {
	var count int64
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("group_id = ?", groupId).
		Where("name = ?", name).
		Count(&count).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("exists by name error: %v", err)
		return false, err
	}
	return count > 0, nil
}

// UpdateCode 在事务内更新角色 code (code = fmt.Sprintf("role%05d", id))
func (r *roleRepo) UpdateCode(ctx context.Context, roleId int64, code string) error {
	err := r.data.DB(ctx).Model(r.model).
		Where("id = ?", roleId).
		Update("code", code).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("update role code error: %v", err)
		return err
	}
	return nil
}

// NewRoleRepo .
func NewRoleRepo(data *common.Data, logger log.Logger) bizV2.RoleRepo {
	return &roleRepo{
		data:  data,
		log:   log.NewHelper(logger),
		model: schema.Role{},
	}
}

// ===== RefUserRoleRepo =====

type refUserRoleRepo struct {
	data  *common.Data
	log   *log.Helper
	model schema.RefUserRole
}

func (r *refUserRoleRepo) FindByUserId(ctx context.Context, userId int64) ([]*bizV2.RefUserRole, error) {
	pos := new([]*schema.RefUserRole)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("user_id = ?", userId).
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find ref user role by user id error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.RefUserRole)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper ref user role err: %v", err)
		return nil, err
	}
	return *bos, nil
}

func (r *refUserRoleRepo) FindByUserIdAndGroupId(ctx context.Context, userId, groupId int64) ([]*bizV2.RefUserRole, error) {
	pos := new([]*schema.RefUserRole)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("user_id = ?", userId).
		Where("group_id = ?", groupId).
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find ref user role by user/group error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.RefUserRole)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper ref user role err: %v", err)
		return nil, err
	}
	return *bos, nil
}

func (r *refUserRoleRepo) FindByGroupId(ctx context.Context, groupId int64) ([]*bizV2.RefUserRole, error) {
	pos := new([]*schema.RefUserRole)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("group_id = ?", groupId).
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find ref user role by group error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.RefUserRole)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper ref user role err: %v", err)
		return nil, err
	}
	return *bos, nil
}

func (r *refUserRoleRepo) FindByUserIdsAndGroupId(ctx context.Context, userIds []int64, groupId int64) ([]*bizV2.RefUserRole, error) {
	if len(userIds) == 0 {
		return nil, nil
	}
	pos := new([]*schema.RefUserRole)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("user_id IN (?)", userIds).
		Where("group_id = ?", groupId).
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find ref user role by user ids/group error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.RefUserRole)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper ref user role err: %v", err)
		return nil, err
	}
	return *bos, nil
}

func (r *refUserRoleRepo) Exists(ctx context.Context, userId, roleId, groupId int64) (bool, error) {
	var count int64
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("user_id = ?", userId).
		Where("role_id = ?", roleId).
		Where("group_id = ?", groupId).
		Count(&count).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("exists ref user role error: %v", err)
		return false, err
	}
	return count > 0, nil
}

// CountByRoleId 统计仍绑定该角色的员工数 (未软删)
func (r *refUserRoleRepo) CountByRoleId(ctx context.Context, roleId int64) (int64, error) {
	var count int64
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("role_id = ?", roleId).
		Count(&count).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("count ref user role by role id error: %v", err)
		return 0, err
	}
	return count, nil
}

func (r *refUserRoleRepo) Create(ctx context.Context, ref *bizV2.RefUserRole) error {
	sRef := common.ConvertData[bizV2.RefUserRole, schema.RefUserRole](ctx, ref)
	err := r.data.DB(ctx).Model(r.model).Create(sRef).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("create ref user role error: %v", err)
		return err
	}
	ref.Id = sRef.Id
	return nil
}

func (r *refUserRoleRepo) Delete(ctx context.Context, userId, roleId, groupId int64) error {
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0 AND user_id = ? AND role_id = ? AND group_id = ?", userId, roleId, groupId).
		Updates(map[string]interface{}{
			"updated_by":        userId,
			"deleted_at":        time.Now(),
			"deleted_unique_id": gorm.Expr("id"),
		}).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("delete ref user role error: %v", err)
		return err
	}
	return nil
}

// NewRefUserRoleRepo .
func NewRefUserRoleRepo(data *common.Data, logger log.Logger) bizV2.RefUserRoleRepo {
	return &refUserRoleRepo{
		data:  data,
		log:   log.NewHelper(logger),
		model: schema.RefUserRole{},
	}
}

// ===== RefRolePermissionRepo =====
// 注意: schema.RefRolePermission 的 isDeleted 字段未导出且 gorm tag 标注为 is_deleted,
// 但实际表列名为 deleted_unique_id (与 v1 实现一致), 因此查询/删除均显式使用 deleted_unique_id.

type refRolePermissionRepo struct {
	data  *common.Data
	log   *log.Helper
	model schema.RefRolePermission
}

func (r *refRolePermissionRepo) FindByRoleIds(ctx context.Context, roleIds []int64) ([]*bizV2.RefRolePermission, error) {
	if len(roleIds) == 0 {
		return nil, nil
	}
	pos := new([]*schema.RefRolePermission)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("role_id IN (?)", roleIds).
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find ref role permission by role ids error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.RefRolePermission)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper ref role permission err: %v", err)
		return nil, err
	}
	return *bos, nil
}

func (r *refRolePermissionRepo) DeleteByRoleId(ctx context.Context, userId, roleId int64) error {
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0 AND role_id = ?", roleId).
		Updates(map[string]interface{}{
			"updated_by":        userId,
			"deleted_at":        time.Now(),
			"deleted_unique_id": gorm.Expr("id"),
		}).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("delete ref role permission error: %v", err)
		return err
	}
	return nil
}

func (r *refRolePermissionRepo) CreateBatch(ctx context.Context, refs []*bizV2.RefRolePermission) error {
	if len(refs) == 0 {
		return nil
	}
	pos := new([]*schema.RefRolePermission)
	if err := mapper.MapperSlice(refs, pos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper ref role permission to schema error: %v", err)
		return err
	}
	err := r.data.DB(ctx).Model(r.model).CreateInBatches(pos, 1000).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("create ref role permission batch error: %v", err)
		return err
	}
	return nil
}

// NewRefRolePermissionRepo .
func NewRefRolePermissionRepo(data *common.Data, logger log.Logger) bizV2.RefRolePermissionRepo {
	return &refRolePermissionRepo{
		data:  data,
		log:   log.NewHelper(logger),
		model: schema.RefRolePermission{},
	}
}
