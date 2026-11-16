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

type refUserRoleRepo struct {
    data  *common.Data
    model schema.RefUserRole
    log   *log.Helper
}

func (r *refUserRoleRepo) FindByRole(ctx context.Context, roleId int64) ([]*user.RefUserRole, error) {
    pos := new([]*schema.RefUserRole)
    err := r.data.DB(ctx).Model(r.model).
        Where(" deleted_unique_id = 0").
        Where(" role_id = ? ", roleId).
        Find(pos).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("query ref user role fail ,%+v", err)
        return nil, err
    }
    bos := new([]*user.RefUserRole)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapp to schema failed, err:%v", err)
        return nil, err
    }
    return *bos, nil
}

func (r *refUserRoleRepo) BatchCreate(ctx context.Context, refUserGroups []*user.RefUserRole) error {
    pos := new([]*schema.RefUserRole)
    err := mapper.MapperSlice(refUserGroups, pos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapp to schema failed, err:%v", err)
        return err
    }
    err = r.data.DB(ctx).Model(r.model).
        CreateInBatches(pos, 1000).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("create ref user role error:%v", err)
        return err
    }
    return nil
}

func (r *refUserRoleRepo) BatchDelete(ctx context.Context, deletedBy, userId, groupId int64) error {
    db := r.data.DB(ctx).Model(r.model).Where("deleted_unique_id = 0 AND user_id = ?", userId)
    
    // 如果groupId为0，则删除所有组的角色关联，否则只删除指定组的角色关联
    if groupId != 0 {
        db = db.Where("group_id = ?", groupId)
    }
    err := db.Updates(map[string]interface{}{
        "updated_by":        deletedBy,
        "deleted_at":        time.Now(),
        "deleted_unique_id": gorm.Expr("id"),
    }).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("delete user role ref error:%v", err)
        return err
    }
    return nil
}

func (r *refUserRoleRepo) Create(ctx context.Context, refUserGroups *user.RefUserRole) error {
    sRef := common.ConvertData[user.RefUserRole, schema.RefUserRole](ctx, refUserGroups)
    err := r.data.DB(ctx).Model(r.model).Create(sRef).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("create user error:%v", err)
        return err
    }
    refUserGroups.Id = sRef.Id
    return nil
}

func (r *refUserRoleRepo) FindByUserId(ctx context.Context, userId, groupId int64) ([]*user.RefUserRole, error) {
    return r.FindByUserIds(ctx, []int64{userId}, groupId)
}

func (r *refUserRoleRepo) FindByUserIds(ctx context.Context, userIds []int64, groupId int64) ([]*user.RefUserRole, error) {
    pos := new([]*schema.RefUserRole)
    err := r.data.DB(ctx).Model(r.model).
        Where("deleted_unique_id = 0").
        Where("user_id IN (?)", userIds).
        Where("group_id = ?", groupId).
        Find(pos).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("find user roles failed, err:%v", err)
        return nil, err
    }
    bos := new([]*user.RefUserRole)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper user error:%v", err)
        return nil, err
    }
    return *bos, nil

}

func (r *refUserRoleRepo) FindByUserIdsAndAppId(ctx context.Context, userIds []int64, appId int64) ([]*user.RefUserRole, error) {
    pos := new([]*schema.RefUserRole)
    err := r.data.DB(ctx).Model(r.model).
        Joins("JOIN role ON role.id = ref_user_role.role_id AND role.app = ? AND role.deleted_unique_id = 0", appId).
        Where("ref_user_role.deleted_unique_id = 0").
        Where("ref_user_role.user_id IN (?)", userIds).
        Find(pos).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("find user roles by appId failed, err:%v", err)
        return nil, err
    }
    bos := new([]*user.RefUserRole)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper user error:%v", err)
        return nil, err
    }
    return *bos, nil
}

func (r *refUserRoleRepo) Delete(ctx context.Context, userId, refId int64) error {
    err := r.data.DB(ctx).Model(r.model).Where("deleted_unique_id = 0 AND id = ?   ", refId).
        Updates(map[string]interface{}{
            "updated_by":        userId,
            "delete_time":       time.Now(),
            "deleted_unique_id": gorm.Expr("id"),
        }).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("delete user error:%v", err)
        return err
    }
    return nil
}

func (r *refUserRoleRepo) BatchDeleteByUserId(ctx context.Context, deletedBy, userId int64) error {
    err := r.data.DB(ctx).Model(r.model).Where("deleted_unique_id = 0 AND user_id = ?", userId).
        Updates(map[string]interface{}{
            "updated_by":        deletedBy,
            "deleted_at":        time.Now(),
            "deleted_unique_id": gorm.Expr("id"),
        }).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("delete user role by userId error:%v", err)
        return err
    }
    return nil
}

func NewRefUserRoleRepo(data *common.Data, logger log.Logger) user.RefUserRoleRepo {
    return &refUserRoleRepo{
        data:  data,
        log:   log.NewHelper(logger),
        model: schema.RefUserRole{},
    }
}
