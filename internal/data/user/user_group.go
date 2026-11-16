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

type refUserGroupRepo struct {
    data  *common.Data
    model schema.RefUserGroup
    log   *log.Helper
}

func (r *refUserGroupRepo) RemoveUser(ctx context.Context, updater, groupId int64, memberId []int64) error {
    err := r.data.DB(ctx).Model(r.model).
        Where("deleted_unique_id = 0").
        Where("group_id = ?", groupId).
        Where("user_id IN (?)", memberId).
        Updates(map[string]interface{}{
            "updated_by":        updater,
            "deleted_at":        time.Now(),
            "deleted_unique_id": gorm.Expr("id"),
        }).Error
    if err == nil {
        r.log.WithContext(ctx).Errorf("BatchDeleteByIds err: %v", err)
        return err
    }
    return nil
}

func (r *refUserGroupRepo) FindByUserIds(ctx context.Context, id []int64) ([]*user.RefUserGroup, error) {
    s := new([]*schema.RefUserGroup)
    err := r.data.DB(ctx).Model(r.model).
        Where("deleted_unique_id = 0").
        Where("user_id IN (?)", id).
        Find(s).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("find ref userGroup err:%v", err)
        return nil, err
    }
    bos := new([]*user.RefUserGroup)
    err = mapper.MapperSlice(s, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper ref userGroup err:%v", err)
        return nil, err
    }
    return *bos, nil
}

func (r *refUserGroupRepo) FindByGroupId(ctx context.Context, id int64) ([]*user.RefUserGroup, error) {
    s := new([]*schema.RefUserGroup)
    err := r.data.DB(ctx).Model(r.model).
        Where("deleted_unique_id = 0").
        Where("group_id = ?", id).
        Find(s).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("find ref userGroup err:%v", err)
        return nil, err
    }
    bos := new([]*user.RefUserGroup)
    err = mapper.MapperSlice(s, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper ref userGroup err:%v", err)
        return nil, err
    }
    return *bos, nil
}

func (r *refUserGroupRepo) BatchDeleteByIds(ctx context.Context, deleteBy, delUserId int64) error {
    err := r.data.DB(ctx).Model(r.model).
        Where("deleted_unique_id = 0").
        Where("user_id = ?", delUserId).
        Updates(map[string]interface{}{
            "updated_by":        deleteBy,
            "deleted_at":        time.Now(),
            "deleted_unique_id": gorm.Expr("id"),
        }).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("BatchDeleteByIds err: %v", err)
        return err
    }
    return nil
}

func (r *refUserGroupRepo) FindByUser(ctx context.Context, id int64) ([]*user.RefUserGroup, error) {
    s := new([]*schema.RefUserGroup)
    err := r.data.DB(ctx).Model(r.model).
        Where("deleted_unique_id = 0").
        Where("user_id = ?", id).
        Find(s).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("find ref userGroup err:%v", err)
        return nil, err
    }
    bos := new([]*user.RefUserGroup)
    err = mapper.MapperSlice(s, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper ref userGroup err:%v", err)
        return nil, err
    }
    return *bos, nil
}

func (r *refUserGroupRepo) BatchCreate(ctx context.Context, refUserGroups []*user.RefUserGroup) error {
    s := new([]*schema.RefUserGroup)
    err := mapper.MapperSlice(refUserGroups, s)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper.MapperSlice failed: %v", err)
        return err
    }
    err = r.data.DB(ctx).Model(r.model).CreateInBatches(s, 1000).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("create user error:%v", err)
        return err
    }
    return nil
}

func (r *refUserGroupRepo) BatchDelete(ctx context.Context, updatedBy int64, groups []*user.RefUserGroup) error {
    updateSql := "update ref_user_group set deleted_unique_id = ? ,updated_by = ?, deleted_time = ? WHERE deleted_unique_id = 0 AND (user_id,group_id) IN "
    var params []interface{}
    params = append(params, gorm.Expr("id"), updatedBy, time.Now())
    updateSql += "("
    for i, cond := range groups {
        if i > 0 {
            updateSql += ", "
        }
        updateSql += "(?, ?)"
        params = append(params, cond.UserId, cond.GroupId)
    }
    updateSql += ")"

    var pos []schema.RefUserGroup
    err := r.data.DB(ctx).Raw(updateSql, params...).Scan(&pos).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("Raw query failed: %v", err)
        return err
    }
    return nil
}

func (r *refUserGroupRepo) FindMembersByGroup(ctx context.Context, c user.UserCondition, id, offset, limit int64) ([]*user.RefUserGroup, int64, error) {
    pos := new([]*schema.RefUserGroup)
    db := r.data.DB(ctx).Model(r.model).
        Where("ref_user_group.deleted_unique_id = 0").
        Where("ref_user_group.group_id = ?", id)

    if c.NickName != nil {
        db = db.Joins("inner join user on ref_user_group.user_id = user.id")
        db = common.QueryWithEscapedLike(db, "user.nickname", *c.NickName)
    }
    if c.Username != nil {
        db = db.Joins("inner join user on ref_user_group.user_id = user.id")
        db = common.QueryWithEscapedLike(db, "user.username", *c.Username)
    }
    var total int64
    db = db.Order("id desc")
    db.Count(&total)
    err := db.Limit(int(limit)).Offset(int(offset)).Find(pos).Error

    if err != nil {
        r.log.WithContext(ctx).Errorf("find user error:%v", err)
        return nil, 0, err
    }
    bos := new([]*user.RefUserGroup)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper.MapperSlice failed: %v", err)
        return nil, 0, err
    }
    return *bos, total, nil

}

func NewRefUserGroupRepo(data *common.Data, logger log.Logger) user.RefUserGroupRepo {
    return &refUserGroupRepo{
        data:  data,
        log:   log.NewHelper(logger),
        model: schema.RefUserGroup{},
    }
}
