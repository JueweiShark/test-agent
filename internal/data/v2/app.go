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

type appRepo struct {
	data  *common.Data
	log   *log.Helper
	model schema.App
}

func (r *appRepo) FindAll(ctx context.Context) ([]*bizV2.App, error) {
	pos := new([]*schema.App)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Order("sort asc, id asc").
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find all app error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.App)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper app err: %v", err)
		return nil, err
	}
	return *bos, nil
}

func (r *appRepo) FindByIds(ctx context.Context, ids []int64) ([]*bizV2.App, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	pos := new([]*schema.App)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id IN (?)", ids).
		Order("sort asc, id asc").
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find app by ids error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.App)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper app err: %v", err)
		return nil, err
	}
	return *bos, nil
}

func (r *appRepo) FindById(ctx context.Context, id int64) (*bizV2.App, error) {
	a := new(schema.App)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id = ?", id).
		First(a).Error
	if err != nil {
		return nil, err
	}
	return common.ConvertData[schema.App, bizV2.App](ctx, a), nil
}

func (r *appRepo) FindByPage(ctx context.Context, cond bizV2.AppCondition, offset, limit int) ([]*bizV2.App, int64, error) {
	pos := new([]*schema.App)
	db := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Order("sort asc, id asc")
	if cond.Name != nil {
		db = common.QueryWithEscapedLike(db, "name", *cond.Name)
	}
	if cond.Code != nil {
		db = common.QueryWithEscapedLike(db, "code", *cond.Code)
	}
	if cond.Status != nil {
		db = db.Where("status = ?", *cond.Status)
	}
	if cond.Type != nil && *cond.Type != "" {
		db = db.Where("type = ?", *cond.Type)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		r.log.WithContext(ctx).Errorf("count apps error: %v", err)
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(limit).Find(pos).Error; err != nil {
		r.log.WithContext(ctx).Errorf("find apps by page error: %v", err)
		return nil, 0, err
	}
	bos := new([]*bizV2.App)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper app by page err: %v", err)
		return nil, 0, err
	}
	return *bos, total, nil
}

func (r *appRepo) Create(ctx context.Context, a *bizV2.App) error {
	sApp := common.ConvertData[bizV2.App, schema.App](ctx, a)
	err := r.data.DB(ctx).Model(r.model).Create(sApp).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("create app error: %v", err)
		return err
	}
	a.Id = sApp.Id
	return nil
}

func (r *appRepo) Update(ctx context.Context, a *bizV2.App) error {
	// 使用 map 更新，避免 GORM Updates(struct) 跳过 status=0 等零值字段
	updates := map[string]interface{}{
		"name":       a.Name,
		"code":       a.Code,
		"entry_url":  a.EntryUrl,
		"icon":       a.Icon,
		"sort":       a.Sort,
		"status":     a.Status,
		"desc":       a.Desc,
		"detail":     a.Detail,
		"type":       a.Type,
		"updated_by": a.UpdatedBy,
	}
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id = ?", *a.Id).
		Updates(updates).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("update app error: %v", err)
		return err
	}
	return nil
}

func (r *appRepo) Delete(ctx context.Context, id, userId int64) error {
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0 AND id = ?", id).
		Updates(map[string]interface{}{
			"updated_by":        userId,
			"deleted_at":        time.Now(),
			"deleted_unique_id": gorm.Expr("id"),
		}).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("delete app error: %v", err)
		return err
	}
	return nil
}

// NewAppRepo .
func NewAppRepo(data *common.Data, logger log.Logger) bizV2.AppRepo {
	return &appRepo{
		data:  data,
		log:   log.NewHelper(logger),
		model: schema.App{},
	}
}
