package v2

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm/clause"

	bizV2 "wuwen.com/wwuc/internal/biz/v2"
	"wuwen.com/wwuc/internal/data/common"
	"wuwen.com/wwuc/internal/data/schema"
	"wuwen.com/wwuc/util/wuwen/mapper"
)

type systemConfigRepo struct {
	data  *common.Data
	log   *log.Helper
	model schema.SystemConfig
}

func (r *systemConfigRepo) GetAll(ctx context.Context) ([]*bizV2.SystemConfig, error) {
	pos := new([]*schema.SystemConfig)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("get all system config error: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.SystemConfig)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper system config err: %v", err)
		return nil, err
	}
	return *bos, nil
}

func (r *systemConfigRepo) GetByKey(ctx context.Context, key string) (*bizV2.SystemConfig, error) {
	c := new(schema.SystemConfig)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("config_key = ?", key).
		First(c).Error
	if err != nil {
		return nil, err
	}
	return common.ConvertData[schema.SystemConfig, bizV2.SystemConfig](ctx, c), nil
}

// Upsert 按 config_key 唯一约束 upsert: 存在则更新 value/updated_by, 不存在则插入.
// 用 OnConflict DoNothing + 失败后 Update 的两步法不优雅, 这里直接先查后决定 insert/update.
func (r *systemConfigRepo) Upsert(ctx context.Context, c *bizV2.SystemConfig) error {
	existing, err := r.GetByKey(ctx, c.ConfigKey)
	if err == nil && existing != nil {
		// 更新已存在配置
		err = r.data.DB(ctx).Model(r.model).
			Where("deleted_unique_id = 0").
			Where("config_key = ?", c.ConfigKey).
			Updates(map[string]interface{}{
				"config_value": c.ConfigValue,
				"updated_by":   c.UpdatedBy,
				"updated_at":   time.Now(),
			}).Error
		if err != nil {
			r.log.WithContext(ctx).Errorf("upsert update config error: %v", err)
		}
		return err
	}
	// 不存在则插入, 用 OnConflict 兜底并发竞争 (config_key 唯一索引)
	sConfig := common.ConvertData[bizV2.SystemConfig, schema.SystemConfig](ctx, c)
	err = r.data.DB(ctx).Model(r.model).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "config_key"}, {Name: "deleted_unique_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"config_value", "updated_by", "updated_at"}),
	}).Create(sConfig).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("upsert insert config error: %v", err)
	}
	return err
}

// NewSystemConfigRepo .
func NewSystemConfigRepo(data *common.Data, logger log.Logger) bizV2.SystemConfigRepo {
	return &systemConfigRepo{
		data:  data,
		log:   log.NewHelper(logger),
		model: schema.SystemConfig{},
	}
}
