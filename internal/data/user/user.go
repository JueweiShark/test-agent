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

type userRepo struct {
	data  *common.Data
	log   *log.Helper
	model schema.User
}

func (r *userRepo) FindByUserName(ctx context.Context, userName []string) ([]*user.User, error) {
	sUsers := new([]*schema.User)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("username IN (?)", userName).
		Order("id desc").
		Find(sUsers).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find users err: %v", err)
		return nil, err
	}
	bUsers := new([]*user.User)
	err = mapper.MapperSlice(sUsers, bUsers)
	if err != nil {
		r.log.WithContext(ctx).Errorf("mapper users err: %v", err)
		return nil, err
	}
	return *bUsers, nil
}

func (r *userRepo) FindByIds(ctx context.Context, ids []int64) ([]*user.User, error) {
	sUsers := new([]*schema.User)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id IN (?)", ids).
		Order("id desc").
		Find(sUsers).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find users err: %v", err)
		return nil, err
	}
	bUsers := new([]*user.User)
	err = mapper.MapperSlice(sUsers, bUsers)
	if err != nil {
		r.log.WithContext(ctx).Errorf("mapper users err: %v", err)
		return nil, err
	}
	return *bUsers, nil
}

func (r *userRepo) Create(ctx context.Context, u *user.User) error {
	sUser := common.ConvertData[user.User, schema.User](ctx, u)
	err := r.data.DB(ctx).Model(r.model).Create(sUser).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("create user error:%v", err)
		return err
	}
	u.Id = sUser.Id
	return nil
}

func (r *userRepo) Update(ctx context.Context, u *user.User) error {
	sUser := common.ConvertData[user.User, schema.User](ctx, u)
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id = ?", u.Id).Updates(sUser).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("update user error:%v", err)
		return err
	}
	return nil
}

func (r *userRepo) Forbidden(ctx context.Context, id, status int64) error {
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("Forbidden user error:%v", err)
		return err
	}
	return nil
}

func (r *userRepo) FindOne(ctx context.Context, id int64) (*user.User, error) {
	u := &schema.User{}
	err := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Where("id = ?", id).First(u).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("FindOne user error:%v", err)
		return nil, err
	}
	biz := common.ConvertData[schema.User, user.User](ctx, u)
	return biz, nil
}

func (r *userRepo) FindByPage(ctx context.Context, condition user.UserCondition, offset, limit int) ([]*user.User, int64, error) {
	pos := new([]*schema.User)
	db := r.data.DB(ctx).Model(r.model).
		Where("deleted_unique_id = 0").
		Order("id desc")
	if condition.App >= 2 {
		db = db.Where("app = ?",condition.App)
	}else{
		db = db.Where("app = 0  and type = 1")
	}
	if condition.NickName != nil {
		db = common.QueryWithEscapedLike(db, "nickname", *condition.NickName)
	}
	if condition.Username != nil {
		db = common.QueryWithEscapedLike(db, "username", *condition.Username)
	}
	if condition.Status != nil {
		db = db.Where("status = ?", *condition.Status)
	}
	if len(condition.ExcludeIds) > 0 {
		db = db.Where("id NOT IN (?)", condition.ExcludeIds)
	}
	var total int64
	db.Count(&total)
	err := db.Offset(offset).Limit(limit).
		Find(pos).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find user error:%v", err)
		return nil, 0, err
	}
	bos := new([]*user.User)
	err = mapper.MapperSlice(pos, bos)
	if err != nil {
		r.log.WithContext(ctx).Errorf("mapper user error:%v", err)
		return nil, 0, err
	}
	return *bos, total, nil
}

func (r *userRepo) Delete(ctx context.Context, userId, id int64) error {
	err := r.data.DB(ctx).Model(r.model).Where(" deleted_unique_id = 0  AND  id = ?", id).
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

func (r *userRepo) FindOneByUserName(ctx context.Context, userName string, app int64) (*user.User, error) {
	u := new(schema.User)
	err := r.data.DB(ctx).Model(r.model).
		First(u, "username = ? and deleted_unique_id = 0 and app = ?", userName, app).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find user error:%v", err)
		return nil, err
	}
	biz := common.ConvertData[schema.User, user.User](ctx, u)
	return biz, nil
}

func (r *userRepo) FindManager(ctx context.Context, ids []int64) ([]*user.GroupManager, error) {

	results := new([]*user.GroupManager)
	err := r.data.DB(ctx).Table("user").
		Select("DISTINCT(user.id) as id, group.id as group_id, user.nickname as nickname").
		Joins("INNER JOIN ref_user_role ON ref_user_role.deleted_unique_id = ? AND user.id = ref_user_role.user_id", 0).
		Joins("INNER JOIN role ON ref_user_role.role_id = role.id AND role.system = ?", 1).
		Joins("INNER JOIN `group` ON role.group_id = `group`.id AND `group`.id IN (?)", ids).
		Scan(&results).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("Find  user error:%v", err)
		return nil, err
	}
	return *results, nil
}
func (r *userRepo) FindGroupManager(ctx context.Context, ids []int64) ([]*user.GroupManager, error) {
	results := new([]*user.GroupManager)
	err := r.data.DB(ctx).Table("user").
		Select("DISTINCT(user.id) as id, group.id as group_id, user.nickname as nickname").
		Joins("INNER JOIN `ref_user_group` ON `ref_user_group`.user_id = user.id and `ref_user_group`.deleted_unique_id = 0").
		Joins("INNER JOIN `group` ON ref_user_group.group_id = `group`.id AND `group`.id IN (?)", ids).
		Scan(&results).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("Find  user error:%v", err)
		return nil, err
	}
	return *results, nil
}

// NewUserRepo .
func NewUserRepo(data *common.Data, logger log.Logger) user.UserRepo {
	return &userRepo{
		data:  data,
		log:   log.NewHelper(logger),
		model: schema.User{},
	}
}

func (r *userRepo) FindByCond(ctx context.Context, condition *user.UserSearchCondition) ([]*user.User, error) {
	sUsers := new([]*schema.User)
	db := r.data.DB(ctx).Model(r.model).Where("deleted_unique_id = 0")
	if condition != nil && len(condition.NickName) > 0 {
		db = common.QueryWithEscapedLike(db, "nickname", condition.NickName)
	}
	if condition != nil && len(condition.Username) > 0 {
		db = common.QueryWithEscapedLike(db, "username", condition.Username)
	}
	if condition != nil && condition.Ids != nil {
		db = db.Where("id IN (?)", condition.Ids)
	}
	//默认查找appId=0的用户信息
	if condition != nil && condition.AppId != 0 {
		db = db.Where("app = ? and is_all = 1", condition.AppId)
	} else {
		db = db.Where("app = 0")
	}
	db = db.Order("id desc")
	err := db.Find(sUsers).Error
	if err != nil {
		r.log.WithContext(ctx).Errorf("find users err: %v", err)
		return nil, err
	}
	bUsers := new([]*user.User)
	err = mapper.MapperSlice(sUsers, bUsers)
	if err != nil {
		r.log.WithContext(ctx).Errorf("mapper users err: %v", err)
		return nil, err
	}
	return *bUsers, nil
}
