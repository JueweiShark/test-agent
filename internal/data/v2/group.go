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

// ===== GroupRepo =====

type groupRepo struct {
	data  *common.Data
	log   *log.Helper
	model schema.Group
}

func (r *groupRepo) Create(ctx context.Context, t *bizV2.Group) error {
	sGroup := common.ConvertData[bizV2.Group, schema.Group](ctx, t)
	err := r.data.DB(ctx).Model(r.model).Create(sGroup).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("create group error: %v", err)
		return err
	}
	t.Id = sGroup.Id
	return nil
}

func (r *groupRepo) Update(ctx context.Context, t *bizV2.Group) error {
	sGroup := common.ConvertData[bizV2.Group, schema.Group](ctx, t)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id = ?", *t.Id).
		Updates(sGroup).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("update group error: %v", err)
		return err
	}
	return nil
}

func (r *groupRepo) Delete(ctx context.Context, userId, groupId int64) error {
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0 AND id = ?", groupId).
		Updates(map[string]interface{}{
			"updated_by":        userId,
			"deleted_at":        time.Now(),
			"deleted_unique_id": gorm.Expr("id"),
		}).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("delete group error: %v", err)
		return err
	}
	return nil
}

func (r *groupRepo) FindById(ctx context.Context, id int64) (*bizV2.Group, error) {
	g := new(schema.Group)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id = ?", id).
		First(g).Error
	if err != nil {
		return nil, err
	}
	return common.ConvertData[schema.Group, bizV2.Group](ctx, g), nil
}

func (r *groupRepo) FindByIds(ctx context.Context, ids []int64) ([]*bizV2.Group, error) {
	pos := new([]*schema.Group)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id IN (?)", ids).
		Order("id desc").
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find groups by ids err: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.Group)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper groups err: %v", err)
		return nil, err
	}
	return *bos, nil
}

func (r *groupRepo) FindByPage(ctx context.Context, cond bizV2.GroupCondition, offset, limit int) ([]*bizV2.Group, int64, error) {
	pos := new([]*schema.Group)
	db := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("system = 1").
		Order("id desc")
	if cond.Name != nil {
		db = common.QueryWithEscapedLike(db, "name", *cond.Name)
	}
	if cond.GroupCode != nil {
		db = common.QueryWithEscapedLike(db, "group_code", *cond.GroupCode)
	}
	if cond.Status != nil {
		db = db.Where("status = ?", *cond.Status)
	}
	if len(cond.Ids) > 0 {
		db = db.Where("id IN (?)", cond.Ids)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		r.log.WithContext(ctx).Errorf("count groups error: %v", err)
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(limit).Find(pos).Error; err != nil {
		r.log.WithContext(ctx).Errorf("find groups by page error: %v", err)
		return nil, 0, err
	}
	bos := new([]*bizV2.Group)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper groups error: %v", err)
		return nil, 0, err
	}
	return *bos, total, nil
}

func (r *groupRepo) ExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("name = ?", name).
		Count(&count).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("exists by name error: %v", err)
		return false, err
	}
	return count > 0, nil
}

func (r *groupRepo) ExistsByCode(ctx context.Context, code string) (bool, error) {
	var count int64
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("group_code = ?", code).
		Count(&count).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("exists by code error: %v", err)
		return false, err
	}
	return count > 0, nil
}

// NewGroupRepo .
func NewGroupRepo(data *common.Data, logger log.Logger) bizV2.GroupRepo {
	return &groupRepo{
		data:  data,
		log:   log.NewHelper(logger),
		model: schema.Group{},
	}
}

