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

type roleRepo struct {
    data  *common.Data
    log   *log.Helper
    model schema.Role
}

func (r *roleRepo) IsAdmin(ctx context.Context, id int64) ([]*user.Role, error) {
    pos := new([]*schema.Role)
    err := r.data.DB(ctx).Model(r.model).
        Joins("inner join ref_user_role on role.id = ref_user_role.role_id and ref_user_role.deleted_unique_id = 0 AND ref_user_role.user_id = ?", id).
        Where("role.deleted_unique_id = 0").
        Find(pos).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("find user error:%v", err)
        return nil, err
    }
    bos := new([]*user.Role)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper to biz error:%v", err)
        return nil, err
    }
    return *bos, nil
}

func (r *roleRepo) FindByIds(ctx context.Context, groupId int64, ids []int64) ([]*user.Role, error) {
    pos := new([]*schema.Role)
    db := r.data.DB(ctx).Model(r.model)
    db = db.Where("deleted_unique_id = 0 and group_id = ? and id IN (?)", groupId, ids)
    err := db.Find(pos).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("find user error:%v", err)
        return nil, err
    }
    bos := new([]*user.Role)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper to biz error:%v", err)
        return nil, err
    }
    return *bos, nil
}

func (r *roleRepo) FindByIdsAndAppId(ctx context.Context, appId int64, ids []int64) ([]*user.Role, error) {
    pos := new([]*schema.Role)
    db := r.data.DB(ctx).Model(r.model)
    db = db.Where("deleted_unique_id = 0 and app = ? and id IN (?)", appId, ids)
    err := db.Find(pos).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("find role by appId error:%v", err)
        return nil, err
    }
    bos := new([]*user.Role)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper to biz error:%v", err)
        return nil, err
    }
    return *bos, nil
}

func (r *roleRepo) Delete(ctx context.Context, userId, roleId int64) error {
    err := r.data.DB(ctx).Model(r.model).Where(" deleted_unique_id = 0 AND id = ? ", roleId).
        Updates(map[string]interface{}{
            "updated_by":        userId,
            "deleted_at":        time.Now(),
            "deleted_unique_id": gorm.Expr("id"),
        }).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("delete user error:%v", err)
        return err
    }
    return nil
}

func (r *roleRepo) List(ctx context.Context, groupId int64, name string, offset, limit int) ([]*user.Role, int64, error) {
    pos := new([]*schema.Role)
    db := r.data.DB(ctx).Model(r.model).
        Where("deleted_unique_id = 0").
        Where("group_id = ?", groupId)
    if len(name) > 0 {
        db = common.QueryWithEscapedLike(db, "name", name)
    }
    db.Order("id desc")
    var total int64
    db.Count(&total)
    err := db.Offset(offset).Limit(limit).
        Find(pos).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("find user error:%v", err)
        return nil, 0, err
    }
    bos := new([]*user.Role)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper user error:%v", err)
        return nil, 0, err
    }
    return *bos, total, nil
}

func (r *roleRepo) ListByAppId(ctx context.Context, appId int64, name string, offset, limit int) ([]*user.Role, int64, error) {
    pos := new([]*schema.Role)
    db := r.data.DB(ctx).Model(r.model).
        Where("deleted_unique_id = 0").
        Where("app = ?", appId)
    if len(name) > 0 {
        db = common.QueryWithEscapedLike(db, "name", name)
    }
    db.Order("id desc")
    var total int64
    db.Count(&total)
    err := db.Offset(offset).Limit(limit).
        Find(pos).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("ListByAppId error:%v", err)
        return nil, 0, err
    }
    bos := new([]*user.Role)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper user error:%v", err)
        return nil, 0, err
    }
    return *bos, total, nil
}
func (r *roleRepo) First(ctx context.Context, id int64) (*user.Role, error) {
    u := &schema.Role{}
    err := r.data.DB(ctx).Model(r.model).
        Where("deleted_unique_id = 0").
        Where("id = ?  ", id).First(u).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("update user error:%v", err)
        return nil, err
    }
    biz := common.ConvertData[schema.Role, user.Role](ctx, u)
    return biz, nil
}

func (r *roleRepo) Update(ctx context.Context, ro *user.Role) error {
    po := common.ConvertData[user.Role, schema.Role](ctx, ro)
    err := r.data.DB(ctx).Model(r.model).
        Where("deleted_unique_id = 0").
        Where("id = ?", po.Id).
        Updates(po).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("update role error:%v", err)
        return err
    }
    return nil
}

func (r *roleRepo) Create(ctx context.Context, role *user.Role) error {
    sr := common.ConvertData[user.Role, schema.Role](ctx, role)
    err := r.data.DB(ctx).Model(r.model).Create(sr).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("create role error:%v", err)
        return err
    }
    role.Id = sr.Id
    return nil
}
func (r *roleRepo) BatchCreate(ctx context.Context, role []*user.Role) error {
    pos := make([]*schema.Role, 0)
    err := mapper.MapperSlice(role, &pos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper role error:%v", err)
        return err
    }
    err = r.data.DB(ctx).Model(r.model).CreateInBatches(pos, 100).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("create role error:%v", err)
        return err
    }
    for i, _ := range role {
        role[i].Id = pos[i].Id
    }
    return nil
}

// NewRoleRepo .
func NewRoleRepo(data *common.Data, logger log.Logger) user.RoleRepo {
    return &roleRepo{
        data:  data,
        log:   log.NewHelper(logger),
        model: schema.Role{},
    }
}
