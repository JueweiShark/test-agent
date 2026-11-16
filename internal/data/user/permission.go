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

type permissionRepo struct {
    data  *common.Data
    model schema.Permission
    log   *log.Helper
}

func (r *permissionRepo) FindByUser(ctx context.Context, id int64) ([]*user.Permission, error) {
    pos := new([]*schema.Permission)
    err := r.data.DB(ctx).Model(r.model).
        Select("permission.code").
        Joins("inner join ref_role_permission on permission.id = ref_role_permission.permission_id and ref_role_permission.deleted_unique_id = 0 and permission.deleted_unique_id = 0").
        Joins("inner join wwuc.role r on ref_role_permission.role_id = r.id and r.deleted_unique_id = 0").
        Joins("inner join wwuc.ref_user_role rur on r.id = rur.role_id and rur.deleted_unique_id = 0").
        Where("rur.user_id = ?", id).
        Scan(&pos).Error
    if err != nil {
        return nil, err
    }
    bos := new([]*user.Permission)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        return nil, err
    }
    return *bos, err
}

func (r *permissionRepo) FindByPage(ctx context.Context, offset int64, limit int64) ([]*user.Permission, error) {
    pos := new([]*schema.Permission)
    err := r.data.DB(ctx).Model(r.model).Where("deleted_unique_id = 0").
        Find(pos).Error
    if err != nil {
        return nil, err
    }
    bos := new([]*user.Permission)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        return nil, err
    }
    return *bos, err
}

func (r *permissionRepo) FindByIds(ctx context.Context, ids []int64) ([]*user.Permission, error) {
    pos := new([]*schema.Permission)
    err := r.data.DB(ctx).Model(r.model).
        Find(pos, "id IN (?) and deleted_unique_id = 0", ids).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("find permission error:%v", err)
        return nil, err
    }
    bos := new([]*user.Permission)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper permission error:%v", err)
        return nil, err
    }
    return *bos, nil
}

func (r *permissionRepo) FindByCode(ctx context.Context, codes []int64) ([]*user.Permission, error) {
    pos := new([]*schema.Permission)
    err := r.data.DB(ctx).Model(r.model).
        Find(pos, "deleted_unique_id = 0 and code IN (?)  ", codes).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("find permission error:%v", err)
        return nil, err
    }
    bos := new([]*user.Permission)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper permission error:%v", err)
        return nil, err
    }
    return *bos, nil
}

func (r *permissionRepo) Create(ctx context.Context, p []*user.Permission) error {
    var pos []*schema.Permission
    err := mapper.MapperSlice(&p, &pos)
    if err != nil {
        r.log.Errorf("batch create Permission ,mapper Permission to schema fail ,%+v", err)
        return err
    }
    err = r.data.DB(ctx).Model(r.model).CreateInBatches(pos, 100).Error
    if err != nil {
        r.log.Errorf("Create metric set error:%v", err)
        return err
    }
    for i, _ := range p {
        p[i].Id = pos[i].Id
    }
    return nil
}

func (r *permissionRepo) Update(ctx context.Context, p *user.Permission) error {
    po := common.ConvertData[user.Permission, schema.Permission](ctx, p)
    err := r.data.DB(ctx).Model(r.model).
        Where("deleted_unique_id = ?", 0).
        Where("id = ?", po.Id).
        Updates(po).Error
    if err != nil {
        r.log.Errorf("update Permission error:%v", err)
        return err
    }
    return nil
}

func (r *permissionRepo) Delete(ctx context.Context, userId int, id int64) error {
    err := r.data.DB(ctx).Model(r.model).Where("deleted_unique_id = 0 AND id = ? ", id).
        Updates(map[string]interface{}{
            "updated_by":        userId,
            "deleted_at":        time.Now(),
            "deleted_unique_id": gorm.Expr("id"),
        }).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("delete Permission  error:%v", err)
        return err
    }
    return nil
}

// NewPermissionRepo .
func NewPermissionRepo(data *common.Data, logger log.Logger) user.PermissionRepo {
    return &permissionRepo{
        data:  data,
        log:   log.NewHelper(logger),
        model: schema.Permission{},
    }
}
