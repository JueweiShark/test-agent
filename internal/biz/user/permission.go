package user

import (
    "context"

    "github.com/go-kratos/kratos/v2/errors"
    "github.com/go-kratos/kratos/v2/log"
    "gorm.io/gorm"

    apiErr "wuwen.com/wwuc/api/errors"
    "wuwen.com/wwuc/internal/biz/common"
)

type Permission struct {
    Id             int64   ` mapper:"id"`
    Domain         string  `mapper:"domain"`
    PermissionCode int64   `mapper:"permissionCode"`
    Name           string  `mapper:"name"`
    Type           uint8   `mapper:"type"`
    Act            *string `mapper:"Act"`
    Path           string  `mapper:"path"`
    Parents        int64   `mapper:"dependenceParents"`
    CreatedBy      int64   `mapper:"createdBy"`
    UpdatedBy      int64   `mapper:"updatedBy"`
    Description    *string `mapper:"description"`
}

type PermissionRepo interface {
    Create(ctx context.Context, p []*Permission) error
    Update(ctx context.Context, p *Permission) error
    Delete(ctx context.Context, userId int, id int64) error

    FindByCode(ctx context.Context, codes []int64) ([]*Permission, error)
    FindByIds(ctx context.Context, ids []int64) ([]*Permission, error)
    FindByPage(ctx context.Context, offset int64, limit int64) ([]*Permission, error)

    FindByUser(ctx context.Context, id int64) ([]*Permission, error)
}
type PermissionUsecase struct {
    log *log.Helper
    tm  common.Transaction
    pr  PermissionRepo
}

func NewPermissionUsecase(logger log.Logger, tm common.Transaction, pr PermissionRepo) *PermissionUsecase {
    return &PermissionUsecase{
        log: log.NewHelper(logger),
        tm:  tm,
        pr:  pr,
    }
}
func (uc *PermissionUsecase) List(ctx context.Context) ([]*Permission, error) {
    page, err := uc.pr.FindByPage(ctx, -1, -1)
    if err != nil {
        return nil, err
    }
    return page, nil
}

func (uc *PermissionUsecase) Create(ctx context.Context, p []*Permission) *errors.Error {
    err := uc.pr.Create(ctx, p)
    if errors.Is(err, gorm.ErrDuplicatedKey) {
        uc.log.WithContext(ctx).Errorf("Permission duplicate ")
        return apiErr.PermissionNameDuplicate
    } else if err != nil {
        uc.log.WithContext(ctx).Errorf("create Permission failed, %v", err)
        return apiErr.PermissionCreateFail
    }
    return nil
}

func (uc *PermissionUsecase) Update(ctx context.Context, p *Permission) *errors.Error {
    //_, err := uc.pr.FindOne(ctx, p.Id)
    //if err != nil {
    //    uc.log.WithContext(ctx).Errorf("find Permission failed, %v", err)
    //    return apiErr.PermissionUpdateFail
    //}

    err := uc.pr.Update(ctx, p)
    if errors.Is(err, gorm.ErrDuplicatedKey) {
        uc.log.WithContext(ctx).Errorf("metric set name duplicate ")
        return apiErr.PermissionNameDuplicate
    } else if err != nil {
        uc.log.WithContext(ctx).Errorf("update Permission failed, %v", err)
        return apiErr.PermissionUpdateFail
    }
    return nil

}

func (uc *PermissionUsecase) Delete(ctx context.Context, userId int, id int64) *errors.Error {
    err := uc.pr.Delete(ctx, userId, id)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("delete Permission failed, %v", err)
        return apiErr.PermissionDeleteFail
    }
    return nil
}
