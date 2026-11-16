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

type refGroupPermissionRepo struct {
	data  *common.Data
	log   *log.Helper
	model schema.RefGroupPermission
}

func (r *refGroupPermissionRepo) FindByGroupId(ctx context.Context, groupId int64) ([]*bizV2.RefGroupPermission, error) {
	pos := new([]*schema.RefGroupPermission)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("group_id = ?", groupId).
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find ref group permission by group id error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.RefGroupPermission)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper ref group permission err: %v", err)
		return nil, err
	}
	return *bos, nil
}

func (r *refGroupPermissionRepo) FindByGroupIds(ctx context.Context, groupIds []int64) ([]*bizV2.RefGroupPermission, error) {
	if len(groupIds) == 0 {
		return nil, nil
	}
	pos := new([]*schema.RefGroupPermission)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("group_id IN (?)", groupIds).
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find ref group permission by group ids error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.RefGroupPermission)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper ref group permission err: %v", err)
		return nil, err
	}
	return *bos, nil
}

func (r *refGroupPermissionRepo) Create(ctx context.Context, ref *bizV2.RefGroupPermission) error {
	sRef := common.ConvertData[bizV2.RefGroupPermission, schema.RefGroupPermission](ctx, ref)
	err := r.data.DB(ctx).Model(r.model).Create(sRef).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("create ref group permission error: %v", err)
		return err
	}
	ref.Id = sRef.Id
	return nil
}

func (r *refGroupPermissionRepo) DeleteByGroupIdAndPermissionId(ctx context.Context, groupId, permissionId, userId int64) error {
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0 AND group_id = ? AND permission_id = ?", groupId, permissionId).
		Updates(map[string]interface{}{
			"updated_by":        userId,
			"deleted_at":        time.Now(),
			"deleted_unique_id": gorm.Expr("id"),
		}).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("delete ref group permission error: %v", err)
		return err
	}
	return nil
}

// CascadeDeleteRolePermissions 级联软删越界的 ref_role_permission 记录.
// 当企业 scope 收缩 (removed permissionIds) 时, 角色-权限关联中引用了被移除权限的记录需软删.
// 过滤条件: 角色属于该企业 (role.group_id=groupId) AND permission_id IN (removed) AND 未软删.
//
// 实现说明: 用 role_id IN (子查询查 role) 替代 JOIN.
// GORM 的 Updates(map) 不会把 .Joins() 正确带进 UPDATE 语句, 导致别名 r.group_id 未知 (Error 1054).
// ref_role_permission 表本身无 group_id 列, 必须经由 role 表定位企业归属.
func (r *refGroupPermissionRepo) CascadeDeleteRolePermissions(ctx context.Context, groupId int64, permissionIds []int64, userId int64) error {
	if len(permissionIds) == 0 {
		return nil
	}
	roleIdsInGroup := r.data.DB(ctx).Table("role").
		Select("id").
		Where("deleted_unique_id = 0").
		Where("group_id = ?", groupId)
	err := r.data.DB(ctx).Table("ref_role_permission").
		Where("deleted_unique_id = 0").
		Where("permission_id IN (?)", permissionIds).
		Where("role_id IN (?)", roleIdsInGroup).
		Updates(map[string]interface{}{
			"updated_by":        userId,
			"deleted_at":        time.Now(),
			"deleted_unique_id": gorm.Expr("id"),
		}).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("cascade delete role permission error: %v", err)
		return err
	}
	return nil
}

// NewRefGroupPermissionRepo .
func NewRefGroupPermissionRepo(data *common.Data, logger log.Logger) bizV2.RefGroupPermissionRepo {
	return &refGroupPermissionRepo{
		data:  data,
		log:   log.NewHelper(logger),
		model: schema.RefGroupPermission{},
	}
}
