package v2

import (
	"context"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/crypto/bcrypt"
	"time"
	"wuwen.com/wwuc/internal/pkg/oss"

	bizUser "wuwen.com/wwuc/internal/biz/user"
)

// tokenGenerateTypeAPI v2 登录签发的 token 类型 (与 v1 保持一致, 走网关鉴权)
const tokenGenerateTypeGateWay = "GATEWAY"

// GroupGroup 登录响应中 "用户在某企业的角色聚合" 结构
type GroupGroup struct {
	Group *Group
	Roles []*Role
}

// LoginResult LoginV2 用例返回的聚合结果, 由 service 层转成 proto
type LoginResult struct {
	Token        string
	ExpireTime   int64
	User         *User
	Groups       []*GroupGroup
	Permissions  []*Permission
	IsSuperAdmin bool
}

// AuthUsecase 登录与总览用例
type AuthUsecase struct {
	log *log.Helper
	ur  UserRepo
	tr  GroupRepo
	rr  RoleRepo
	rur RefUserRoleRepo
	pr  PermissionRepo
	ar  AppRepo
	tuc *bizUser.TokenUsecase
	uuc *UserUsecase
	oss *oss.Usecase
}

// NewAuthUsecase .
func NewAuthUsecase(logger log.Logger,
		ur UserRepo, tr GroupRepo, rr RoleRepo, rur RefUserRoleRepo,
		pr PermissionRepo, ar AppRepo,
		tuc *bizUser.TokenUsecase, uuc *UserUsecase, oss *oss.Usecase) *AuthUsecase {
	return &AuthUsecase{
		log: log.NewHelper(logger),
		ur:  ur,
		tr:  tr,
		rr:  rr,
		rur: rur,
		pr:  pr,
		ar:  ar,
		tuc: tuc,
		uuc: uuc,
		oss: oss,
	}
}

// LoginV2 登录主流程 (对应 design 2.1)
// 事务边界: 无显式事务. token 持久化是单表 insert, 失败即报错; lastLoginAt 更新失败不阻塞.
func (uc *AuthUsecase) LoginV2(ctx context.Context, username, password string) (*LoginResult, *errors.Error) {
	// 2. 查用户 (v2 不传 App, 全平台共用一份用户数据)
	u, err := uc.ur.FindByUserName(ctx, username)
	if err != nil || u == nil {
		uc.log.WithContext(ctx).Errorf("login find user failed: %v", err)
		return nil, errUserNotExist
	}
	// 3. 状态校验 不等于1 则不能登录
	if u.Status != 1 {
		return nil, errUseIsForbidden
	}
	// 4. 密码比对
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		uc.log.WithContext(ctx).Errorf("login password mismatch: %v", err)
		return nil, errUserPWDNotMatch
	}
	// 5. 生成 token (JWT 只装 userId + 过期时间 + 签发时间)
	token, expTime, err := uc.generateToken(ctx, *u.Id)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("generate token failed: %v", err)
		return nil, errUserTokenGenerateFail
	}

	// 6. 查用户角色绑定 [{roleId, groupId}]
	refs, err := uc.rur.FindByUserId(ctx, *u.Id)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("find user roles failed: %v", err)
		return nil, errInternalError
	}
	roleIds := make([]int64, 0, len(refs))
	roleIdSet := make(map[int64]bool, len(refs))
	for _, r := range refs {
		if !roleIdSet[r.RoleId] {
			roleIdSet[r.RoleId] = true
			roleIds = append(roleIds, r.RoleId)
		}
	}

	// 7. 超管分支 (Role.Id == 1)
	isSuper := IsSuperAdmin(roleIds)
	var groups []*GroupGroup
	var perms []*Permission
	if isSuper {
		// 超管 不用返回菜单 默认拥有全部
		groups = []*GroupGroup{}
		//allPerms, perr := uc.pr.FindAll(ctx)
		//if perr != nil {
		//	uc.log.WithContext(ctx).Errorf("super admin find all permissions failed: %v", perr)
		//	return nil, errInternalError
		//}
		//perms = allPerms
	} else {
		// 8. 普通员工分支
		if len(roleIds) == 0 {
			// 无任何角色绑定 -> 未关联任何企业
			return nil, errUserNoGroup
		}
		// 权限继承: 有效权限 = 角色权限 ∩ 企业 scope (3表 join)
		perms, err = uc.pr.FindEffectiveByUserId(ctx, *u.Id)
		if err != nil {
			uc.log.WithContext(ctx).Errorf("find effective permissions by user id failed: %v", err)
			return nil, errInternalError
		}
		// 按 groupId 聚合 ref_user_role, 拿去重后的 groupIds
		groupIdSet := make(map[int64]bool, len(refs))
		for _, r := range refs {
			if r.GroupId > 0 {
				groupIdSet[r.GroupId] = true
			}
		}
		groupIds := make([]int64, 0, len(groupIdSet))
		for gid := range groupIdSet {
			groupIds = append(groupIds, gid)
		}
		// 9. 组装 groups: 查 group + 按 group 聚合 roles
		groups, err = uc.assembleGroups(ctx, groupIds, refs)
		if err != nil {
			return nil, errInternalError
		}
	}

	// 10. 更新最后登录时间 (失败仅记日志)
	uc.uuc.UpdateLastLoginAt(ctx, *u.Id)

	return &LoginResult{
		Token:        token,
		ExpireTime:   expTime,
		User:         u,
		Groups:       groups,
		Permissions:  perms,
		IsSuperAdmin: isSuper,
	}, nil
}

// GetUserInfo 查询当前用户信息 (登录后单独请求, 对应 proto AuthV2.GetUserInfo)
// 超管: isSuperAdmin=true, groups/permissions 为空 (默认拥有全部, 与 LoginV2 一致)
// 非超管: groupId 必填且用户须归属该企业, 只返回当前 groupId 对应的有效权限与企业信息
func (uc *AuthUsecase) GetUserInfo(ctx context.Context, userId, groupId int64) (*LoginResult, *errors.Error) {
	u, err := uc.ur.FindById(ctx, userId)
	if err != nil || u == nil {
		return nil, errUserNotExist
	}
	if u.Status != 1 {
		return nil, errUseIsForbidden
	}

	refs, err := uc.rur.FindByUserId(ctx, *u.Id)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("get user info find roles failed: %v", err)
		return nil, errInternalError
	}
	roleIds := make([]int64, 0, len(refs))
	roleIdSet := make(map[int64]bool, len(refs))
	for _, r := range refs {
		if !roleIdSet[r.RoleId] {
			roleIdSet[r.RoleId] = true
			roleIds = append(roleIds, r.RoleId)
		}
	}

	isSuper := IsSuperAdmin(roleIds)
	var groups []*GroupGroup
	var perms []*Permission
	if isSuper {
		// 超管 不返回 groups/permissions, 默认拥有全部 (与 LoginV2 保持一致)
		groups = []*GroupGroup{}
	} else {
		// 非超管 groupId 必填
		if groupId <= 0 {
			return nil, errParamsInvalid
		}
		// 校验用户是否归属该企业, 同时收集本企业内的角色绑定
		belongs := false
		scopedRefs := make([]*RefUserRole, 0, len(refs))
		for _, r := range refs {
			if r.GroupId == groupId {
				belongs = true
				scopedRefs = append(scopedRefs, r)
			}
		}
		if !belongs {
			return nil, errUserNotInGroup
		}
		// 只取当前企业下的有效权限 (3表 join, 固定 group_id)
		perms, err = uc.pr.FindEffectiveByUserIdAndGroupId(ctx, *u.Id, groupId)
		if err != nil {
			uc.log.WithContext(ctx).Errorf("get user info find effective permissions failed: %v", err)
			return nil, errInternalError
		}
		// 只组装当前企业 (含该企业下的角色)
		groups, err = uc.assembleGroups(ctx, []int64{groupId}, scopedRefs)
		if err != nil {
			return nil, errInternalError
		}
	}

	return &LoginResult{
		User:         u,
		Groups:       groups,
		Permissions:  perms,
		IsSuperAdmin: isSuper,
	}, nil
}

// ListMyApps 总览查我可进的平台 (对应 design 2.4.2)
// 超管: 返回全部启用平台; 普通员工: 取用户在该企业有效权限涉及的 app 去重后查 app 表
// appType 非空时按 type 过滤 (platform=平台, link=链接)
func (uc *AuthUsecase) ListMyApps(ctx context.Context, userId, groupId int64, isSuper bool, appType string) ([]*App, *errors.Error) {
	// 超管看所有平台
	if isSuper {
		apps, err := uc.ar.FindAll(ctx)
		if err != nil {
			uc.log.WithContext(ctx).Errorf("super admin list apps failed: %v", err)
			return nil, errInternalError
		}
		for idx, item := range apps {
			if item.Icon == "" {
				continue
			}
			apps[idx].IconUrl, _ = uc.oss.PresignedGetUrl(ctx, item.Icon)
		}
		return filterAppsByType(apps, appType), nil
	}
	// 普通员工: 从该企业内有效权限中提取 app 去重
	perms, err := uc.pr.FindEffectiveByUserIdAndGroupId(ctx, userId, groupId)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("list my apps find effective permissions failed: %v", err)
		return nil, errInternalError
	}
	if len(perms) == 0 {
		return nil, nil
	}
	appIdSet := make(map[int64]bool, len(perms))
	appIds := make([]int64, 0, len(perms))
	for _, p := range perms {
		if p.App > 0 && !appIdSet[p.App] {
			appIdSet[p.App] = true
			appIds = append(appIds, p.App)
		}
	}
	if len(appIds) == 0 {
		return nil, nil
	}
	apps, err := uc.ar.FindByIds(ctx, appIds)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("list my apps find apps failed: %v", err)
		return nil, errInternalError
	}
	for idx, item := range apps {
		if item.Icon == "" {
			continue
		}
		apps[idx].IconUrl, _ = uc.oss.PresignedGetUrl(ctx, item.Icon)
	}
	return filterAppsByType(apps, appType), nil
}

// filterAppsByType 按 type 过滤 app 列表 (type 为空时原样返回)
func filterAppsByType(apps []*App, appType string) []*App {
	if appType == "" {
		return apps
	}
	out := make([]*App, 0, len(apps))
	for _, a := range apps {
		if a != nil && a.Type == appType {
			out = append(out, a)
		}
	}
	return out
}

// CheckUserGroupAccess 校验用户是否归属某企业 (gRPC, 供下游/网关)
// 逻辑: 查 ref_user_role 表, user_id=X AND group_id=Y AND deleted_unique_id=0 存在即 true.
func (uc *AuthUsecase) CheckUserGroupAccess(ctx context.Context, userId, groupId int64) (bool, *errors.Error) {
	u, err := uc.ur.FindById(ctx, userId)
	if err != nil || u == nil {
		return false, nil
	}
	refs, err := uc.rur.FindByUserIdAndGroupId(ctx, userId, groupId)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("check user group access failed: %v", err)
		return false, errInternalError
	}
	for _, r := range refs {
		if r.RoleId == roleAdminId {
			return true, nil
		}
	}
	return len(refs) > 0, nil
}

// IssueGroupToken 数据隔离改造 (groupId 入 Token): 校验基础 token 并换发携带 groupId claim 的组 token。
// 流程:
//  1. 服务端解析基础 token 得到 userId (不信任前端直接传 userId);
//  2. 超管 (Role.Id==1): groupId 置 0, 签发全局 token, groups/permissions 为空;
//  3. 普通员工: 校验 userId+groupId 绑定存在于 ref_user_role, 绑定不存在返回 403;
//  4. 签发 JWT (claims 含 userId+groupId), 写 user_token 表 (含 group_id, 保留按组吊销与审计能力);
//  5. 返回该组有效权限 (FindEffectiveByUserIdAndGroupId)。
//
// 切换分组即前端携带新 groupId 重复调用本接口重新签发, 无需重新输入密码。
// 原 LoginV2 及存量 GenerateToken 签发路径零改动。
func (uc *AuthUsecase) IssueGroupToken(ctx context.Context, baseToken string, groupId int64) (*LoginResult, *errors.Error) {
	// 1. 校验基础 token, 服务端解析出 userId
	parsed, err := uc.tuc.ParseToken(ctx, baseToken)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("issue group token parse base token failed: %v", err)
		return nil, errUserTokenInvalid
	}
	userId := parsed.UserId

	// 2. 用户存在性与状态校验 (与 LoginV2 一致)
	u, uerr := uc.ur.FindById(ctx, userId)
	if uerr != nil || u == nil {
		return nil, errUserNotExist
	}
	if u.Status != 1 {
		return nil, errUseIsForbidden
	}

	// 3. 查角色绑定, 判定超管
	refs, rerr := uc.rur.FindByUserId(ctx, userId)
	if rerr != nil {
		uc.log.WithContext(ctx).Errorf("issue group token find user roles failed: %v", rerr)
		return nil, errInternalError
	}
	roleIds := make([]int64, 0, len(refs))
	for _, r := range refs {
		roleIds = append(roleIds, r.RoleId)
	}
	isSuper := IsSuperAdmin(roleIds)

	var groups []*GroupGroup
	var perms []*Permission
	if isSuper {
		// 超管不挂企业: groupId 置空, 签发全局 token, 跳过租户隔离过滤
		groupId = 0
		groups = []*GroupGroup{}
	} else {
		// 普通员工: groupId 必填
		if groupId <= 0 {
			return nil, errParamsInvalid
		}
		// 绑定校验: 复用 CheckUserGroupAccess 的查询路径 (ref_user_role, userId+groupId)
		scopedRefs, gerr := uc.rur.FindByUserIdAndGroupId(ctx, userId, groupId)
		if gerr != nil {
			uc.log.WithContext(ctx).Errorf("issue group token check group access failed: %v", gerr)
			return nil, errInternalError
		}
		if len(scopedRefs) == 0 {
			return nil, errUserNotInGroupForbidden
		}
		// 组内有效权限
		perms, rerr = uc.pr.FindEffectiveByUserIdAndGroupId(ctx, userId, groupId)
		if rerr != nil {
			uc.log.WithContext(ctx).Errorf("issue group token find effective permissions failed: %v", rerr)
			return nil, errInternalError
		}
		groups, rerr = uc.assembleGroups(ctx, []int64{groupId}, scopedRefs)
		if rerr != nil {
			return nil, errInternalError
		}
	}

	// 4. 签发携带 groupId claim 的 token
	exp := time.Now().Add(uc.tuc.ExpireDuration())
	token, terr := uc.tuc.GenerateTokenWithGroup(ctx, userId, groupId, tokenGenerateTypeGateWay, &exp)
	if terr != nil {
		uc.log.WithContext(ctx).Errorf("issue group token generate failed: %v", terr)
		return nil, errUserTokenGenerateFail
	}

	return &LoginResult{
		Token:        token,
		ExpireTime:   exp.Unix(),
		User:         u,
		Groups:       groups,
		Permissions:  perms,
		IsSuperAdmin: isSuper,
	}, nil
}

// IsSuperAdminByUserId 通过 userId 判断是否超管 (供 service 层 ListMyApps/CheckUserGroupAccess 上下文使用)
func (uc *AuthUsecase) IsSuperAdminByUserId(ctx context.Context, userId int64) bool {
	refs, err := uc.rur.FindByUserId(ctx, userId)
	if err != nil {
		return false
	}
	roleIds := make([]int64, 0, len(refs))
	for _, r := range refs {
		roleIds = append(roleIds, r.RoleId)
	}
	return IsSuperAdmin(roleIds)
}

// generateToken 调 TokenUsecase 签发 token, 返回 token 字符串与过期时间(unix 秒)
func (uc *AuthUsecase) generateToken(ctx context.Context, userId int64) (string, int64, error) {
	exp := time.Now().Add(uc.tuc.ExpireDuration())
	token, err := uc.tuc.GenerateToken(ctx, userId, tokenGenerateTypeGateWay, &exp)
	if err != nil {
		return "", 0, err
	}
	return token, exp.Unix(), nil
}

// assembleGroups 按 groupId 聚合 ref_user_role, 查 group 信息, 组装 GroupGroup 列表
func (uc *AuthUsecase) assembleGroups(ctx context.Context, groupIds []int64, refs []*RefUserRole) ([]*GroupGroup, error) {
	if len(groupIds) == 0 {
		return []*GroupGroup{}, nil
	}
	groups, err := uc.tr.FindByIds(ctx, groupIds)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("assemble groups find groups failed: %v", err)
		return nil, errInternalError
	}
	groupMap := make(map[int64]*Group, len(groups))
	for _, t := range groups {
		if t.Id != nil {
			groupMap[*t.Id] = t
		}
	}
	// 收集所有 roleId 批量查 role 信息 (一次查询, 避免每组都查)
	roleIdSet := make(map[int64]bool, len(refs))
	allRoleIds := make([]int64, 0, len(refs))
	for _, r := range refs {
		if !roleIdSet[r.RoleId] {
			roleIdSet[r.RoleId] = true
			allRoleIds = append(allRoleIds, r.RoleId)
		}
	}
	roleMap := make(map[int64]*Role, len(allRoleIds))
	if len(allRoleIds) > 0 {
		roles, rerr := uc.rr.FindByIds(ctx, allRoleIds)
		if rerr != nil {
			uc.log.WithContext(ctx).Errorf("assemble groups find roles failed: %v", rerr)
			return nil, errInternalError
		}
		for _, r := range roles {
			if r.Id != nil {
				roleMap[*r.Id] = r
			}
		}
	}
	// 按 groupId 聚合 roles
	groupRoles := make(map[int64][]*Role, len(groupIds))
	for _, ref := range refs {
		if ref.GroupId <= 0 {
			continue
		}
		if role, ok := roleMap[ref.RoleId]; ok && role != nil {
			groupRoles[ref.GroupId] = append(groupRoles[ref.GroupId], role)
		}
	}
	out := make([]*GroupGroup, 0, len(groupIds))
	for _, gid := range groupIds {
		t, ok := groupMap[gid]
		if !ok || t == nil {
			// 企业可能已被软删, 跳过 (登录时该 group 查不到, 自然无法进)
			continue
		}
		roles := groupRoles[gid]
		if roles == nil {
			roles = []*Role{}
		}
		out = append(out, &GroupGroup{Group: t, Roles: roles})
	}
	return out, nil
}
