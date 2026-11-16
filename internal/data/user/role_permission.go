package user

import (
    "context"
    "time"
    
    "github.com/go-kratos/kratos/v2/log"
    "gorm.io/gorm"
    
    "wuwen.com/wwuc/internal/biz/user"
    "wuwen.com/wwuc/internal/data/common"
    "wuwen.com/wwuc/internal/data/schema"
    "wuwen.com/wwuc/util/wuwen/mapper"
)

type rolePermissionRepo struct {
    data  *common.Data
    model schema.RefRolePermission
    log   *log.Helper
}

func (r *rolePermissionRepo) DeleteByRoleId(ctx context.Context, userId, roleId int64) error {
    err := r.data.DB(ctx).Model(r.model).
        Where("deleted_unique_id = 0").
        Where("role_id = ?", roleId).
        Updates(map[string]interface{}{
            "deleted_unique_id": gorm.Expr("id"),
            "updated_by":        userId,
            "deleted_at":        time.Now(),
        }).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("delete ref role permission fail %+v", err)
        return err
    }
    return nil
}

func (r *rolePermissionRepo) GetRolePermissionByRoleId(ctx context.Context, int64s []int64) ([]*user.RefRolePermission, error) {
    pos := new([]*schema.RefRolePermission)
    err := r.data.DB(ctx).Model(r.model).
        Where("deleted_unique_id = 0").
        Where("role_id in (?)", int64s).
        Find(pos).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("get role permission err:%v", err)
        return nil, err
    }
    bos := new([]*user.RefRolePermission)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper to biz error:%v", err)
        return nil, err
    }
    return *bos, nil
}


func (r *rolePermissionRepo) GetRolePermissionByAppId(ctx context.Context, appId int64) ([]*user.Permission, error) {
    pos := new([]*schema.Permission)
    err := r.data.DB(ctx).Model(schema.Permission{}).
        Where("deleted_unique_id = 0").
        Where("app = ?", appId).
        Find(pos).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("get permission err:%v", err)
        return nil, err
    }
    bos := new([]*user.Permission)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper to biz error:%v", err)
        return nil, err
    }

    return *bos, nil
}

func (r *rolePermissionRepo) CreateRolePermission(ctx context.Context, permissions []*user.RefRolePermission) error {
    pos := new([]*schema.RefRolePermission)
    err := mapper.MapperSlice(permissions, pos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper to schema error:%v", err)
        return err
    }
    err = r.data.DB(ctx).Model(r.model).CreateInBatches(pos, 1000).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("create role permission err:%v", err)
        return err
    }
    return nil
}

// NewRolePermissionRepo .
func NewRolePermissionRepo(data *common.Data, logger log.Logger) user.RefRolePermissionRepo {
    return &rolePermissionRepo{
        data:  data,
        log:   log.NewHelper(logger),
        model: schema.RefRolePermission{},
    }
}
