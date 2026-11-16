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

type userRepo struct {
	data  *common.Data
	log   *log.Helper
	model schema.User
}

// FindByUserName v2 不传 App, 全平台共用一份用户数据 (对应 design 2.1 step2)
// 仅返回 is_all=1 的全平台用户
func (r *userRepo) FindByUserName(ctx context.Context, username string) (*bizV2.User, error) {
	u := new(schema.User)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("is_all = 1").
		Where("username = ?", username).
		First(u).Error
	if err != nil {
		return nil, err
	}
	return common.ConvertData[schema.User, bizV2.User](ctx, u), nil
}

func (r *userRepo) FindById(ctx context.Context, id int64) (*bizV2.User, error) {
	u := new(schema.User)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("is_all = 1").
		Where("id = ?", id).
		First(u).Error
	if err != nil {
		return nil, err
	}
	return common.ConvertData[schema.User, bizV2.User](ctx, u), nil
}

func (r *userRepo) FindByIds(ctx context.Context, ids []int64) ([]*bizV2.User, error) {
	pos := new([]*schema.User)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("is_all = 1").
		Where("id IN (?)", ids).
		Order("id desc").
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find users by ids err: %v", err)
		return nil, err
	}
	bos := new([]*bizV2.User)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper users err: %v", err)
		return nil, err
	}
	return *bos, nil
}

func (r *userRepo) Create(ctx context.Context, u *bizV2.User) error {
	sUser := common.ConvertData[bizV2.User, schema.User](ctx, u)
	// v2 创建用户强制 is_all=1 (全平台), 与 biz 层 CreateUser 保持一致, 防止外部直接调 Repo 漏设
	if sUser.IsAll == 0 {
		sUser.IsAll = 1
	}
	err := r.data.DB(ctx).Model(r.model).Create(sUser).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("create user error: %v", err)
		return err
	}
	u.Id = sUser.Id
	u.IsAll = sUser.IsAll
	return nil
}

func (r *userRepo) UpdatePassword(ctx context.Context, userId int64, password string) error {
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id = ?", userId).
		Updates(map[string]interface{}{
			"password":   password,
			"updated_at": time.Now(),
		}).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("update password error: %v", err)
		return err
	}
	return nil
}

func (r *userRepo) Forbidden(ctx context.Context, userId int64, status int64) error {
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id = ?", userId).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("forbidden user error: %v", err)
		return err
	}
	return nil
}

// UpdateInfo 更新用户信息 (昵称/显隐状态/备注)
func (r *userRepo) UpdateInfo(ctx context.Context, userId int64, nickname string, status int64, remarks string, updatedBy int64) error {
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id = ?", userId).
		Updates(map[string]interface{}{
			"nickname":    nickname,
			"status":      status,
			"remarks":     remarks,
			"updated_by":  updatedBy,
			"updated_at":  time.Now(),
		}).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("update user info error: %v", err)
		return err
	}
	return nil
}

// FindByPage 用户分页列表 (v2 不按 app 过滤, 仅返回 is_all=1 的全平台用户)
func (r *userRepo) FindByPage(ctx context.Context, cond bizV2.UserCondition, offset, limit int) ([]*bizV2.User, int64, error) {
	pos := new([]*schema.User)
	db := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("is_all = 1").
		Order("id desc")
	if cond.Username != nil {
		db = common.QueryWithEscapedLike(db, "username", *cond.Username)
	}
	if cond.Nickname != nil {
		db = common.QueryWithEscapedLike(db, "nickname", *cond.Nickname)
	}
	if cond.Status != nil {
		db = db.Where("status = ?", *cond.Status)
	}
	var total int64
	db.Count(&total)
	if err := db.Offset(offset).Limit(limit).Find(pos).Error; err != nil {
		r.log.WithContext(ctx).Errorf("find user by page error: %v", err)
		return nil, 0, err
	}
	bos := new([]*bizV2.User)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper user error: %v", err)
		return nil, 0, err
	}
	return *bos, total, nil
}

// FindByGroupId 查某企业下的用户 (通过 ref_user_group 子查询去重), 支持用户名/昵称/状态过滤与分页
// 注: 员工-企业关系以 ref_user_group 为准 (ref_user_role 仅记录多角色绑定)
// 仅返回 is_all=1 的全平台用户
func (r *userRepo) FindByGroupId(ctx context.Context, groupId int64, cond bizV2.UserCondition, offset, limit int) ([]*bizV2.User, int64, error) {
	// 子查询: 该企业下未删除的用户 id (去重, 来自 ref_user_group)
	subQuery := r.data.DB(ctx).Table("ref_user_group").
		Select("DISTINCT user_id").
		Where("deleted_unique_id = 0").
		Where("group_id = ?", groupId)

	pos := new([]*schema.User)
	db := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("is_all = 1").
		Where("id IN (?)", subQuery).
		Order("id desc")
	if cond.Username != nil {
		db = common.QueryWithEscapedLike(db, "username", *cond.Username)
	}
	if cond.Nickname != nil {
		db = common.QueryWithEscapedLike(db, "nickname", *cond.Nickname)
	}
	if cond.Status != nil {
		db = db.Where("status = ?", *cond.Status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		r.log.WithContext(ctx).Errorf("count users by group error: %v", err)
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(limit).Find(pos).Error; err != nil {
		r.log.WithContext(ctx).Errorf("find users by group error: %v", err)
		return nil, 0, err
	}
	bos := new([]*bizV2.User)
	if err := mapper.MapperSlice(pos, bos); err != nil {
		r.log.WithContext(ctx).Errorf("mapper user error: %v", err)
		return nil, 0, err
	}
	return *bos, total, nil
}

func (r *userRepo) UpdateLastLoginAt(ctx context.Context, userId int64) error {
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id = ?", userId).
		Updates(map[string]interface{}{
			"last_login_at": time.Now(),
		}).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("update last login at error: %v", err)
		return err
	}
	return nil
}

// SoftDelete 软删用户 (预留, 当前 v2 不暴露删除用户接口)
func (r *userRepo) SoftDelete(ctx context.Context, userId, id int64) error {
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0 AND id = ?", id).
		Updates(map[string]interface{}{
			"updated_by":        userId,
			"deleted_at":        time.Now(),
			"deleted_unique_id": gorm.Expr("id"),
		}).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("delete user error: %v", err)
		return err
	}
	return nil
}

// NewUserRepo .
func NewUserRepo(data *common.Data, logger log.Logger) bizV2.UserRepo {
	return &userRepo{
		data:  data,
		log:   log.NewHelper(logger),
		model: schema.User{},
	}
}
