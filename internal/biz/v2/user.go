package v2

import (
	"context"
	"github.com/go-kratos/kratos/v2/errors"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"wuwen.com/wwuc/internal/biz/common"
	"wuwen.com/wwuc/internal/conf"
)

// User 系统用户 (v2 复用 user 表, 不读写 App 字段)
type User struct {
	Id              *int64     `mapper:"id"`
	Username        string     `mapper:"username"`
	Password        string     `mapper:"password"`
	Nickname        string     `mapper:"nickname"`
	CreatedAt       time.Time  `mapper:"createdAt"`
	CreatedBy       int64      `mapper:"createdBy"`
	UpdatedAt       *time.Time `mapper:"updatedAt"`
	UpdatedBy       int64      `mapper:"updatedBy"`
	LastLoginAt     *time.Time `mapper:"lastLoginAt"`
	DeletedUniqueId int64      `mapper:"uniqueDeletedKey"`
	DeletedAt       *time.Time `mapper:"deletedAt"`
	Provider        string     `mapper:"provider"`
	App             int64      `mapper:"app"`
	Status          int64      `mapper:"status"`
	IsAll           int64      `mapper:"isAll"`
	Remarks         string     `mapper:"remarks"`
}

// UserCondition v2 用户分页查询条件
type UserCondition struct {
	GroupId   *int64   // 可选, nil=不限企业 (跨企业搜索)
	GroupName *string   // 关联企业名称 (模糊)
	Username  *string   // 用户名 (模糊)
	Nickname  *string   // 姓名 (模糊)
	Status    *int32    // 状态 (0=禁用 1=启用)
}

// UserRepo v2 用户仓储
type UserRepo interface {
	FindByUserName(ctx context.Context, username string) (*User, error)
	FindById(ctx context.Context, id int64) (*User, error)
	FindByIds(ctx context.Context, ids []int64) ([]*User, error)
	Create(ctx context.Context, u *User) error
	UpdatePassword(ctx context.Context, userId int64, password string) error
	Forbidden(ctx context.Context, userId int64, status int64) error
	UpdateInfo(ctx context.Context, userId int64, nickname string, status int64, remarks string, updatedBy int64) error
	FindByPage(ctx context.Context, cond UserCondition, offset, limit int) ([]*User, int64, error)
	// FindByGroupId 查某企业下的用户 (join ref_user_role), 支持用户名/昵称/状态过滤与分页
	FindByGroupId(ctx context.Context, groupId int64, cond UserCondition, offset, limit int) ([]*User, int64, error)
	UpdateLastLoginAt(ctx context.Context, userId int64) error
}

// UserUsecase 超管-用户管理
type UserUsecase struct {
	log      *log.Helper
	tm       common.Transaction
	ur       UserRepo
	pwdConf  *conf.Password
}

// NewUserUsecase .
func NewUserUsecase(logger log.Logger, tm common.Transaction, ur UserRepo, pwdConf *conf.Password) *UserUsecase {
	return &UserUsecase{
		log:      log.NewHelper(logger),
		tm:       tm,
		ur:       ur,
		pwdConf:  pwdConf,
	}
}

// CreateUser 超管创建系统用户 (不登记手机号, 不绑企业)
// 返回自动生成的明文密码 (仅供本次回传给前端)
func (uc *UserUsecase) CreateUser(ctx context.Context, u *User) (string, *errors.Error) {
	if len(u.Username) <= 0 {
		return "", errValidateInvalid
	}
	if exist, _ := uc.ur.FindByUserName(ctx, u.Username); exist != nil {
		return "", errUserDuplicate
	}
	plainPwd, err := uc.generateInitialPassword(ctx)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("generate initial password failed: %v", err)
		return "", errInternalError
	}
	u.Password = plainPwd
	if err := uc.generatePwd(u); err != nil {
		uc.log.WithContext(ctx).Errorf("hash password failed: %v", err)
		return "", errInternalError
	}

	// 默认状态为 1 (启用)
	if u.Status == 0 {
		u.Status = 1
	}
	u.Provider = "wuwen"
	// v2 用户均为全平台用户 (is_all=1)
	u.IsAll = 1
	if err := uc.ur.Create(ctx, u); err != nil {
		uc.log.WithContext(ctx).Errorf("create user failed: %v", err)
		if err == gorm.ErrDuplicatedKey {
			return "", errUserDuplicate
		}
		return "", errInternalError
	}
	return plainPwd, nil
}

// ResetPassword 超管重置用户密码
// 返回自动生成的新明文密码 (仅供本次回传给前端)
func (uc *UserUsecase) ResetPassword(ctx context.Context, userId int64) (string, *errors.Error) {
	if _, err := uc.ur.FindById(ctx, userId); err != nil {
		return "", errUserNotFound
	}
	plainPwd, err := uc.generateInitialPassword(ctx)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("generate initial password failed: %v", err)
		return "", errInternalError
	}
	u := &User{Id: &userId, Password: plainPwd}
	if err := uc.generatePwd(u); err != nil {
		uc.log.WithContext(ctx).Errorf("hash password failed: %v", err)
		return "", errInternalError
	}
	if err := uc.ur.UpdatePassword(ctx, userId, u.Password); err != nil {
		uc.log.WithContext(ctx).Errorf("reset password failed: %v", err)
		return "", errInternalError
	}
	return plainPwd, nil
}

// UpdatePassword 超管指定新密码修改用户密码 (不回显明文)
// 流程: FindById → ValidatePassword 校验规则 → bcrypt 哈希 → UpdatePassword 落库
func (uc *UserUsecase) UpdatePassword(ctx context.Context, userId int64, plainPwd string) *errors.Error {
	if _, err := uc.ur.FindById(ctx, userId); err != nil {
		return errUserNotFound
	}
	var minLen, maxLen int32
	if uc.pwdConf != nil {
		minLen = uc.pwdConf.GetMinLen()
		maxLen = uc.pwdConf.GetMaxLen()
	}
	if err := ValidatePassword(plainPwd, int(minLen), int(maxLen)); err != nil {
		uc.log.WithContext(ctx).Errorf("update password violates rule: %v", err)
		return errPwdRuleInvalid
	}
	u := &User{Id: &userId, Password: plainPwd}
	if err := uc.generatePwd(u); err != nil {
		uc.log.WithContext(ctx).Errorf("hash password failed: %v", err)
		return errInternalError
	}
	if err := uc.ur.UpdatePassword(ctx, userId, u.Password); err != nil {
		uc.log.WithContext(ctx).Errorf("update password failed: %v", err)
		return errInternalError
	}
	return nil
}

// generateInitialPassword 根据配置生成初始密码 (固定值或随机), 并校验规则
func (uc *UserUsecase) generateInitialPassword(ctx context.Context) (string, error) {
	var plain string
	var minLen, maxLen int32
	if uc.pwdConf != nil {
		minLen = uc.pwdConf.GetMinLen()
		maxLen = uc.pwdConf.GetMaxLen()
		if uc.pwdConf.GetRandom() {
			random, err := GenerateRandomPassword(int(minLen), int(maxLen))
			if err != nil {
				return "", errInternalError
			}
			plain = random
		} else {
			plain = uc.pwdConf.GetDefault()
		}
	} else {
		// 配置缺失兜底
		plain = "Wuwen2026!"
	}
	if err := ValidatePassword(plain, int(minLen), int(maxLen)); err != nil {
		uc.log.WithContext(ctx).Errorf("generated password violates rule: %v plain=%s", err, plain)
		return "", errPwdRuleInvalid
	}
	return plain, nil
}

// Forbidden 禁用/启用用户
func (uc *UserUsecase) Forbidden(ctx context.Context, userId int64, status int64) *errors.Error {
	if _, err := uc.ur.FindById(ctx, userId); err != nil {
		return errUserNotFound
	}
	if err := uc.ur.Forbidden(ctx, userId, status); err != nil {
		uc.log.WithContext(ctx).Errorf("forbidden user failed: %v", err)
		return errInternalError
	}
	return nil
}

// UpdateInfo 更新用户信息 (昵称/显隐状态/备注)
func (uc *UserUsecase) UpdateInfo(ctx context.Context, userId, operator int64, nickname string, status int64, remarks string) *errors.Error {
	if _, err := uc.ur.FindById(ctx, userId); err != nil {
		return errUserNotFound
	}
	if err := uc.ur.UpdateInfo(ctx, userId, nickname, status, remarks, operator); err != nil {
		uc.log.WithContext(ctx).Errorf("update user info failed: %v", err)
		return errInternalError
	}
	return nil
}

// FindById 查用户详情
func (uc *UserUsecase) FindById(ctx context.Context, userId int64) (*User, *errors.Error) {
	u, err := uc.ur.FindById(ctx, userId)
	if err != nil {
		return nil, errUserNotFound
	}
	return u, nil
}

// FindByPage 用户分页列表
func (uc *UserUsecase) FindByPage(ctx context.Context, cond UserCondition, offset, limit int) ([]*User, int64, *errors.Error) {
	users, total, err := uc.ur.FindByPage(ctx, cond, offset, limit)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("find users by page failed: %v", err)
		return nil, 0, errInternalError
	}
	return users, total, nil
}

// UpdateLastLoginAt 更新最后登录时间 (登录时调用, 失败仅记日志)
func (uc *UserUsecase) UpdateLastLoginAt(ctx context.Context, userId int64) {
	if err := uc.ur.UpdateLastLoginAt(ctx, userId); err != nil {
		uc.log.WithContext(ctx).Errorf("update last login at failed: %v", err)
	}
}

func (uc *UserUsecase) generatePwd(u *User) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(bytes)
	return nil
}
