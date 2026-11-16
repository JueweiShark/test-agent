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

type permissionRepo struct {
	data  *common.Data
	log   *log.Helper
	model schema.Permission
}

func (r *permissionRepo) FindAll(ctx context.Context) ([]*bizV2.Permission, error) {
	pos := new([]*schema.Permission)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Order("sort asc, id asc").
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find all permission error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.Permission)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper permission err: %v", err)
		return nil, err
	}
	return *bos, nil
}

func (r *permissionRepo) FindById(ctx context.Context, id int64) (*bizV2.Permission, error) {
	p := new(schema.Permission)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id = ?", id).
		First(p).Error
	if err != nil {
		return nil, err
	}
	return common.ConvertData[schema.Permission, bizV2.Permission](ctx, p), nil
}

func (r *permissionRepo) FindByIds(ctx context.Context, ids []int64) ([]*bizV2.Permission, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	pos := new([]*schema.Permission)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id IN (?)", ids).
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find permission by ids error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.Permission)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper permission err: %v", err)
		return nil, err
	}
	return *bos, nil
}

// FindByRoleIds 按角色过滤的权限列表 (join ref_role_permission, 去重)
func (r *permissionRepo) FindByRoleIds(ctx context.Context, roleIds []int64) ([]*bizV2.Permission, error) {
	if len(roleIds) == 0 {
		return nil, nil
	}
	pos := new([]*schema.Permission)
	err := r.data.DB(ctx).Model(r.model).
		Joins("INNER JOIN ref_role_permission ON ref_role_permission.permission_id = permission.id AND ref_role_permission.deleted_unique_id = 0").
		Where("permission.deleted_unique_id = 0").
		Where("ref_role_permission.role_id IN (?)", roleIds).
		Group("permission.id").
		Order("permission.sort asc, permission.id asc").
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find permission by role ids error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.Permission)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper permission err: %v", err)
		return nil, err
	}
	return *bos, nil
}

// FindEffectiveByUserId 查用户的有效权限 (3表 join: ref_user_role ⨝ ref_role_permission ⨝ ref_group_permission)
// 权限继承: 有效权限 = 角色权限 ∩ 企业 scope
func (r *permissionRepo) FindEffectiveByUserId(ctx context.Context, userId int64) ([]*bizV2.Permission, error) {
	pos := new([]*schema.Permission)
	err := r.data.DB(ctx).Model(r.model).
		Joins("INNER JOIN ref_role_permission rrp ON rrp.permission_id = permission.id AND rrp.deleted_unique_id = 0").
		Joins("INNER JOIN ref_user_role ur ON ur.role_id = rrp.role_id AND ur.deleted_unique_id = 0").
		Joins("INNER JOIN ref_group_permission gp ON gp.permission_id = rrp.permission_id AND gp.group_id = ur.group_id AND gp.deleted_unique_id = 0").
		Where("permission.deleted_unique_id = 0").
		Where("ur.user_id = ?", userId).
		Group("permission.id").
		Order("permission.sort asc, permission.id asc").
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find effective permission by user id error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.Permission)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper permission err: %v", err)
		return nil, err
	}
	return *bos, nil
}

// FindEffectiveByUserIdAndGroupId 查用户在指定企业的有效权限 (3表 join, 固定 group_id)
// 权限继承: 有效权限 = 角色权限 ∩ 企业 scope (限本企业)
func (r *permissionRepo) FindEffectiveByUserIdAndGroupId(ctx context.Context, userId, groupId int64) ([]*bizV2.Permission, error) {
	pos := new([]*schema.Permission)
	err := r.data.DB(ctx).Model(r.model).
		Joins("INNER JOIN ref_role_permission rrp ON rrp.permission_id = permission.id AND rrp.deleted_unique_id = 0").
		Joins("INNER JOIN ref_user_role ur ON ur.role_id = rrp.role_id AND ur.deleted_unique_id = 0").
		Joins("INNER JOIN ref_group_permission gp ON gp.permission_id = rrp.permission_id AND gp.group_id = ur.group_id AND gp.deleted_unique_id = 0").
		Where("permission.deleted_unique_id = 0").
		Where("ur.user_id = ?", userId).
		Where("ur.group_id = ?", groupId).
		Group("permission.id").
		Order("permission.sort asc, permission.id asc").
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find effective permission by user id and group id error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.Permission)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper permission err: %v", err)
		return nil, err
	}
	return *bos, nil
}

func (r *permissionRepo) FindByApp(ctx context.Context, app int64, hidden bool) ([]*bizV2.Permission, error) {
	pos := new([]*schema.Permission)
	db := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Order("sort asc, id asc")
	if app > 0 {
		db = db.Where("app = ?", app)
	}
	if hidden {
		db = db.Where("permission.hidden = ?", 1)
	}
	err := db.Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find permission by app error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.Permission)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper permission err: %v", err)
		return nil, err
	}
	return *bos, nil
}

// FindByAppAndGroupId 按 app 与企业 scope 过滤的权限列表 (join ref_group_permission)
// SELECT p.* FROM permission p JOIN ref_group_permission gp ON gp.permission_id=p.id
// WHERE p.app=? AND gp.group_id=? AND gp.deleted_unique_id=0 AND p.deleted_unique_id=0
func (r *permissionRepo) FindByAppAndGroupId(ctx context.Context, app, groupId int64, hidden bool) ([]*bizV2.Permission, error) {
	pos := new([]*schema.Permission)
	db := r.data.DB(ctx).Model(r.model).
		Joins("INNER JOIN ref_group_permission gp ON gp.permission_id = permission.id AND gp.deleted_unique_id = 0").
		Where("permission.deleted_unique_id = 0").
		Where("gp.group_id = ?", groupId).
		Order("permission.sort asc, permission.id asc")
	if app > 0 {
		db = db.Where("permission.app = ?", app)
	}
	if hidden {
		db = db.Where("permission.hidden = ?", 1)
	}
	err := db.Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find permission by app and group id error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.Permission)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper permission err: %v", err)
		return nil, err
	}
	return *bos, nil
}

func (r *permissionRepo) Create(ctx context.Context, p *bizV2.Permission) error {
	sPerm := common.ConvertData[bizV2.Permission, schema.Permission](ctx, p)
	err := r.data.DB(ctx).Model(r.model).Create(sPerm).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("create permission error: %v", err)
		return err
	}
	p.Id = sPerm.Id
	return nil
}

func (r *permissionRepo) Update(ctx context.Context, p *bizV2.Permission) error {
	sPerm := common.ConvertData[bizV2.Permission, schema.Permission](ctx, p)
	// 用 Select 指定列, 避免零值覆盖: 不传的字段不会被清零
	// 同时保留不可变字段 (app/type/code/domain 由 biz 层已保留 old 值)
	// update_time 用 gorm.Expr 设为 CURRENT_TIMESTAMP, 避免传 NULL 触发 1048
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id = ?", p.Id).
		Select(
			"name",
			"menu_type",
			"route_path",
			"icon",
			"sort",
			"path",
			"act",
			"description",
			"dependence_children",
			"dependence_parents",
			"hidden",
			"app",
			"domain",
			"code",
			"type",
			"update_by",
		).
		Updates(map[string]interface{}{
			"name":               sPerm.Name,
			"menu_type":          sPerm.MenuType,
			"route_path":         sPerm.RoutePath,
			"icon":               sPerm.Icon,
			"sort":               sPerm.Sort,
			"path":               sPerm.Path,
			"act":                sPerm.Act,
			"description":        sPerm.Description,
			"dependence_children": sPerm.DependenceChildren,
			"dependence_parents": sPerm.DependenceParents,
			"hidden":             sPerm.Hidden,
			"app":                sPerm.App,
			"domain":             sPerm.Domain,
			"code":               sPerm.Code,
			"type":               sPerm.Type,
			"update_by":          sPerm.UpdateBy,
			"update_time":        gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("update permission error: %v", err)
		return err
	}
	return nil
}

func (r *permissionRepo) Delete(ctx context.Context, userId int64, id int64) error {
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0 AND id = ?", id).
		Updates(map[string]interface{}{
			"update_by":         userId,
			"update_time":       time.Now(),
			"delete_time":       time.Now(),
			"deleted_unique_id": gorm.Expr("id"),
		}).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("delete permission error: %v", err)
		return err
	}
	return nil
}

// CountChildren 统计某父节点下的未删除子节点数 (用于删除菜单前校验)
func (r *permissionRepo) CountChildren(ctx context.Context, parentId int64) (int64, error) {
	var count int64
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("dependence_parents = ?", parentId).
		Count(&count).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("count children error: %v", err)
		return 0, err
	}
	return count, nil
}

// FindAppIdsByGroupIds 批量查多个企业 scope 内的去重 app_id 列表 (join ref_group_permission)
// SELECT gp.group_id, DISTINCT p.app FROM permission p
// JOIN ref_group_permission gp ON gp.permission_id=p.id AND gp.deleted_unique_id=0
// WHERE p.deleted_unique_id=0 AND p.app>0 AND gp.group_id IN (?) GROUP BY gp.group_id, p.app
// 返回 groupId -> 去重 []app_id 映射, 不在结果集中的 groupId 对应空切片
func (r *permissionRepo) FindAppIdsByGroupIds(ctx context.Context, groupIds []int64) (map[int64][]int64, error) {
	result := make(map[int64][]int64, len(groupIds))
	if len(groupIds) == 0 {
		return result, nil
	}
	type row struct {
		GroupId int64
		App     int64
	}
	var rows []row
	err := r.data.DB(ctx).Table("permission p").
		Select("gp.group_id AS group_id, p.app AS app").
		Joins("INNER JOIN ref_group_permission gp ON gp.permission_id = p.id AND gp.deleted_unique_id = 0").
		Where("p.deleted_unique_id = 0").
		Where("p.app > 0").
		Where("gp.group_id IN (?)", groupIds).
		Group("gp.group_id, p.app").
		Order("gp.group_id ASC, p.app ASC").
		Find(&rows).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find app ids by group ids error: %v", err)
		return nil, err
	}
	for _, x := range rows {
		result[x.GroupId] = append(result[x.GroupId], x.App)
	}
	// 不在结果集中的 groupId 对应空切片, 避免调用方做 nil 判断
	for _, gid := range groupIds {
		if _, ok := result[gid]; !ok {
			result[gid] = []int64{}
		}
	}
	return result, nil
}

// NewPermissionRepo .
func NewPermissionRepo(data *common.Data, logger log.Logger) bizV2.PermissionRepo {
	return &permissionRepo{
		data:  data,
		log:   log.NewHelper(logger),
		model: schema.Permission{},
	}
}
