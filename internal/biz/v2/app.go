package v2

import (
	"context"
	"time"
)

// App 产品表
type App struct {
	Id              *int64     `mapper:"id"`
	Name            string     `mapper:"name"`
	Code            string     `mapper:"code"`
	EntryUrl        string     `mapper:"entryUrl"`
	Icon            string     `mapper:"icon"`
	IconUrl string
	Sort            int32      `mapper:"sort"`
	Status          int8       `mapper:"status"`
	Desc            string     `mapper:"desc"`
	Detail          string     `mapper:"detail"`
	Type            string     `mapper:"type"`
	CreatedBy       int64      `mapper:"createdBy"`
	CreatedAt       *time.Time `mapper:"createdAt"`
	UpdatedBy       int64      `mapper:"updatedBy"`
	UpdatedAt       *time.Time `mapper:"updatedAt"`
	DeletedUniqueId int64      `mapper:"deleteUniqueId"`
	DeletedAt       *time.Time `mapper:"deletedAt"`
}

// AppRepo 产品仓储
type AppRepo interface {
	FindAll(ctx context.Context) ([]*App, error)
	FindByIds(ctx context.Context, ids []int64) ([]*App, error)
	FindById(ctx context.Context, id int64) (*App, error)
	FindByPage(ctx context.Context, cond AppCondition, offset, limit int) ([]*App, int64, error)
	Create(ctx context.Context, a *App) error
	Update(ctx context.Context, a *App) error
	Delete(ctx context.Context, id, userId int64) error
}

// AppCondition 产品分页查询条件
type AppCondition struct {
	Name   *string
	Code   *string
	Status *int32
	Type   *string
}
