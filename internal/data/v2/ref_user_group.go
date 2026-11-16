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

type refUserGroupRepo struct {
	data  *common.Data
	log   *log.Helper
	model schema.RefUserGroup
}

// Create 创建用户-企业关系 (一次加入)
func (r *refUserGroupRepo) Create(ctx context.Context, ref *bizV2.RefUserGroup) error {
	sRef := common.ConvertData[bizV2.RefUserGroup, schema.RefUserGroup](ctx, ref)
	err := r.data.DB(ctx).Model(r.model).Create(sRef).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("create ref user group error: %v", err)
		return err
	}
	// GORM 回填自增主键到 sRef.Id (*int64), 同步回 biz 对象
	if sRef.Id != nil {
		ref.Id = sRef.Id
	}
	return nil
}

// FindByUserIdAndGroupId 查询用户在某企业的成员关系 (未软删)
func (r *refUserGroupRepo) FindByUserIdAndGroupId(ctx context.Context, userId, groupId int64) (*bizV2.RefUserGroup, error) {
	p := new(schema.RefUserGroup)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("user_id = ?", userId).
		Where("group_id = ?", groupId).
		First(p).Error
	if err != nil {
		return nil, err
	}
	return common.ConvertData[schema.RefUserGroup, bizV2.RefUserGroup](ctx, p), nil
}

// FindByGroupId 查企业下所有员工关系 (未软删)
func (r *refUserGroupRepo) FindByGroupId(ctx context.Context, groupId int64) ([]*bizV2.RefUserGroup, error) {
	pos := new([]*schema.RefUserGroup)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("group_id = ?", groupId).
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find ref user group by group error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.RefUserGroup)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper ref user group err: %v", err)
		return nil, err
	}
	return *bos, nil
}

// FindByUserIdsAndGroupId 批量查询多个用户在同一企业的关系
func (r *refUserGroupRepo) FindByUserIdsAndGroupId(ctx context.Context, userIds []int64, groupId int64) ([]*bizV2.RefUserGroup, error) {
	if len(userIds) == 0 {
		return nil, nil
	}
	pos := new([]*schema.RefUserGroup)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("user_id IN (?)", userIds).
		Where("group_id = ?", groupId).
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find ref user group by user ids/group error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.RefUserGroup)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper ref user group err: %v", err)
		return nil, err
	}
	return *bos, nil
}

// refUserGroupListRow 列表查询结果结构 (含 user/group 快照字段)
// 通过 SQL select 指定列 + scan 到 struct, 不复用 schema.RefUserGroup (避免与 mapper 标签冲突)
type refUserGroupListRow struct {
	Id          int64      `gorm:"column:id"`
	UserId      int64      `gorm:"column:user_id"`
	GroupId     int64      `gorm:"column:group_id"`
	Status      int64      `gorm:"column:status"`
	Remarks     string     `gorm:"column:remarks"`
	CreatedTime *time.Time `gorm:"column:created_time"`
	UpdatedTime *time.Time `gorm:"column:updated_time"`
	Username    string     `gorm:"column:username"`
	Nickname    string     `gorm:"column:nickname"`
	GroupName   string     `gorm:"column:group_name"`
	GroupCode   string     `gorm:"column:group_code"`
}

// FindPageByCond 员工-企业关系分页查询 (一条 ref_user_group 记录 = 一个员工-企业关系).
// 通过 JOIN user/group 一次查出展示所需字段, 避免后续 N+1 查询.
// cond.GroupId=nil 表示不限企业 (跨企业搜索); 否则限定单企业.
// cond.Username/Nickname/GroupName 支持模糊匹配; cond.Status 过滤 ref_user_group.status.
func (r *refUserGroupRepo) FindPageByCond(ctx context.Context, cond bizV2.UserCondition, offset, limit int) ([]*bizV2.RefUserGroup, int64, error) {
	db := r.data.DB(ctx).Table("ref_user_group rug").
		Select("rug.id, rug.user_id, rug.group_id, rug.status, rug.remarks, rug.created_time, rug.updated_time, " +
			"u.username, u.nickname, g.name AS group_name, g.group_code").
		Joins("INNER JOIN `user` u ON u.id = rug.user_id AND u.deleted_unique_id = 0").
		Joins("INNER JOIN `group` g ON g.id = rug.group_id AND g.deleted_unique_id = 0").
		Where("rug.deleted_unique_id = 0").
		Order("rug.id desc")

	if cond.GroupId != nil {
		db = db.Where("rug.group_id = ?", *cond.GroupId)
	}
	if cond.Username != nil && *cond.Username != "" {
		db = common.QueryWithEscapedLike(db, "u.username", *cond.Username)
	}
	if cond.Nickname != nil && *cond.Nickname != "" {
		db = common.QueryWithEscapedLike(db, "u.nickname", *cond.Nickname)
	}
	if cond.GroupName != nil && *cond.GroupName != "" {
		db = common.QueryWithEscapedLike(db, "g.name", *cond.GroupName)
	}
	if cond.Status != nil {
		db = db.Where("rug.status = ?", *cond.Status)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		r.log.WithContext(ctx).Errorf("count ref_user_group by cond error: %v", err)
		return nil, 0, err
	}

	var rows []refUserGroupListRow
	if err := db.Offset(offset).Limit(limit).Scan(&rows).Error; err != nil {
		r.log.WithContext(ctx).Errorf("find ref_user_group by cond error: %v", err)
		return nil, 0, err
	}

	out := make([]*bizV2.RefUserGroup, 0, len(rows))
	for i := range rows {
		row := &rows[i]
		bo := &bizV2.RefUserGroup{
			Id:          &row.Id,
			UserId:      row.UserId,
			GroupId:     row.GroupId,
			Status:      row.Status,
			Remarks:     row.Remarks,
			CreatedTime: row.CreatedTime,
			UpdatedTime: row.UpdatedTime,
			Username:    row.Username,
			Nickname:    row.Nickname,
			GroupName:   row.GroupName,
			GroupCode:   row.GroupCode,
		}
		out = append(out, bo)
	}
	return out, total, nil
}

// Exists 判断用户是否已加入企业 (未软删)
func (r *refUserGroupRepo) Exists(ctx context.Context, userId, groupId int64) (bool, error) {
	var count int64
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("user_id = ?", userId).
		Where("group_id = ?", groupId).
		Count(&count).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("exists ref user group error: %v", err)
		return false, err
	}
	return count > 0, nil
}

// Update 更新员工的状态与备注 (status/remarks)
func (r *refUserGroupRepo) Update(ctx context.Context, ref *bizV2.RefUserGroup) error {
	updates := map[string]interface{}{
		"status":       ref.Status,
		"remarks":      ref.Remarks,
		"updated_time": gorm.Expr("CURRENT_TIMESTAMP"),
	}
	if ref.UpdatedBy != nil {
		updates["updated_by"] = *ref.UpdatedBy
	}
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("user_id = ?", ref.UserId).
		Where("group_id = ?", ref.GroupId).
		Updates(updates).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("update ref user group error: %v", err)
		return err
	}
	return nil
}

// Delete 软删用户-企业关系 (未软删 → 软删, 同时刷新 update_by/update_time)
// 注意: 级联软删对应的 ref_user_role 由 biz 层在事务内编排, 此处只负责 ref_user_group 本身
func (r *refUserGroupRepo) Delete(ctx context.Context, userId, groupId int64) error {
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0 AND user_id = ? AND group_id = ?", userId, groupId).
		Updates(map[string]interface{}{
			"updated_by":        userId,
			"updated_time":      time.Now(),
			"deleted_at":        time.Now(),
			"deleted_unique_id": gorm.Expr("id"),
		}).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("delete ref user group error: %v", err)
		return err
	}
	return nil
}

// NewRefUserGroupRepo .
func NewRefUserGroupRepo(data *common.Data, logger log.Logger) bizV2.RefUserGroupRepo {
	return &refUserGroupRepo{
		data:  data,
		log:   log.NewHelper(logger),
		model: schema.RefUserGroup{},
	}
}
