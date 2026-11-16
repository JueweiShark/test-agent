package v2

import (
	"context"
	"time"
	"wuwen.com/wwuc/internal/pkg/oss"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"

	"wuwen.com/wwuc/internal/biz/common"
)

// SystemConfig 系统配置 (key-value)
type SystemConfig struct {
	Id              *int64     `mapper:"id"`
	ConfigKey       string     `mapper:"configKey"`
	ConfigValue     string     `mapper:"configValue"`
	Description     string     `mapper:"description"`
	CreatedBy       int64      `mapper:"createdBy"`
	CreatedAt       *time.Time `mapper:"createdAt"`
	UpdatedBy       int64      `mapper:"updatedBy"`
	UpdatedAt       *time.Time `mapper:"updatedAt"`
	DeletedUniqueId int64      `mapper:"deleteUniqueId"`
	DeletedAt       *time.Time `mapper:"deletedAt"`
}

// SystemConfigRepo 系统配置仓储
type SystemConfigRepo interface {
	GetAll(ctx context.Context) ([]*SystemConfig, error)
	GetByKey(ctx context.Context, key string) (*SystemConfig, error)
	Upsert(ctx context.Context, c *SystemConfig) error
}

// SystemUsecase 系统配置业务用例
type SystemUsecase struct {
	log *log.Helper
	tm  common.Transaction
	scr SystemConfigRepo
	ar  AppRepo
	oss *oss.Usecase
}

// NewSystemUsecase .
func NewSystemUsecase(logger log.Logger, tm common.Transaction, scr SystemConfigRepo, ar AppRepo, oss *oss.Usecase) *SystemUsecase {
	return &SystemUsecase{log: log.NewHelper(logger), tm: tm, scr: scr, ar: ar, oss: oss}
}

// GetSystemConfig 获取配置 (key 为空则返回全部)
func (uc *SystemUsecase) GetSystemConfig(ctx context.Context, key string) (map[string]string, *errors.Error) {
	if key == "" {
		all, err := uc.scr.GetAll(ctx)
		if err != nil {
			uc.log.WithContext(ctx).Errorf("get all config failed: %v", err)
			return nil, errInternalError
		}
		m := make(map[string]string, len(all))
		for _, c := range all {
			m[c.ConfigKey] = c.ConfigValue
		}
		return m, nil
	}
	c, err := uc.scr.GetByKey(ctx, key)
	if err != nil {
		return nil, errSystemConfigNotFound
	}
	return map[string]string{c.ConfigKey: c.ConfigValue}, nil
}

// SetSystemConfig 设置配置 (批量 upsert)
func (uc *SystemUsecase) SetSystemConfig(ctx context.Context, userId int64, configs map[string]string) *errors.Error {
	for k, v := range configs {
		if err := uc.scr.Upsert(ctx, &SystemConfig{ConfigKey: k, ConfigValue: v, UpdatedBy: userId}); err != nil {
			uc.log.WithContext(ctx).Errorf("upsert config %s failed: %v", k, err)
			return errInternalError
		}
	}
	return nil
}

// ListApps 产品列表
func (uc *SystemUsecase) ListApps(ctx context.Context) ([]*App, *errors.Error) {
	apps, err := uc.ar.FindAll(ctx)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("list apps failed: %v", err)
		return nil, errInternalError
	}
	for idx, item := range apps {
		if item.Icon == "" {
			continue
		}
		apps[idx].IconUrl, _ = uc.oss.PresignedGetUrl(ctx, item.Icon)
	}
	return apps, nil
}

// ListAppsByPage 分页查询产品列表 (管理端)
func (uc *SystemUsecase) ListAppsByPage(ctx context.Context, cond AppCondition, offset, limit int) ([]*App, int64, *errors.Error) {
	apps, total, err := uc.ar.FindByPage(ctx, cond, offset, limit)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("list apps by page failed: %v", err)
		return nil, 0, errInternalError
	}
	for idx, item := range apps {
		if item.Icon == "" {
			continue
		}
		apps[idx].IconUrl, _ = uc.oss.PresignedGetUrl(ctx, item.Icon)
	}
	return apps, total, nil
}

// CreateApp 创建产品
func (uc *SystemUsecase) CreateApp(ctx context.Context, a *App) *errors.Error {
	a.Status = 1
	if err := uc.ar.Create(ctx, a); err != nil {
		uc.log.WithContext(ctx).Errorf("create app failed: %v", err)
		return errInternalError
	}
	return nil
}

// UpdateApp 更新产品
func (uc *SystemUsecase) UpdateApp(ctx context.Context, a *App) *errors.Error {
	if a.Id == nil {
		return errParamsInvalid
	}
	old, err := uc.ar.FindById(ctx, *a.Id)
	if err != nil || old == nil {
		return errAppNotFound
	}
	if err := uc.ar.Update(ctx, a); err != nil {
		uc.log.WithContext(ctx).Errorf("update app failed: %v", err)
		return errInternalError
	}
	return nil
}

// DeleteApp 软删产品
func (uc *SystemUsecase) DeleteApp(ctx context.Context, appId, userId int64) *errors.Error {
	if appId <= 0 {
		return errParamsInvalid
	}
	old, err := uc.ar.FindById(ctx, appId)
	if err != nil || old == nil {
		return errAppNotFound
	}
	if err := uc.ar.Delete(ctx, appId, userId); err != nil {
		uc.log.WithContext(ctx).Errorf("delete app failed: %v", err)
		return errInternalError
	}
	return nil
}
