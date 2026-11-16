package v2

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	pb "wuwen.com/wwuc/api_gen/user/v2"

	biz_common "wuwen.com/wwuc/internal/biz/common"
)

// TestLoginV2V2 登录 (白名单接口, 无需 token)
// 前置: 自建 企业+角色+用户 并分配角色 (普通员工无角色会被 errUserNoGroup 拦截)
func TestLoginV2V2(t *testing.T) {
	env := initTestV2Env()
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	// 1. 自建企业
	groupRes, err := env.groupSvc.CreateGroup(ctx, &pb.CreateGroupRequest{
		Name:      uniqGroupName("v2login租户"),
		GroupCode: fmt.Sprintf("VL%d", time.Now().UnixNano()),
		Remarks:   "login单测企业",
	})
	if err != nil {
		t.Fatalf("prepare group failed: %v", err)
	}
	groupId := groupRes.GetData().GetGroup().GetGroupId()

	// 2. 在该企业下建角色
	roleRes, err := env.roleSvc.CreateRole(ctx, &pb.CreateRoleV2Request{
		Name:    uniqGroupName("v2login角色"),
		GroupId: groupId,
	})
	if err != nil {
		t.Fatalf("prepare role failed: %v", err)
	}
	roleId := roleRes.GetData().GetRole().GetRoleId()

	// 3. 自建用户 (密码已知)
	password := "Test12345678"
	userRes, err := env.userSvc.CreateUser(ctx, &pb.CreateUserV2Request{
		Username: fmt.Sprintf("v2login_%d@test.com", time.Now().UnixNano()),
		Password: password,
		Nickname: "v2login用户",
	})
	if err != nil {
		t.Fatalf("prepare user failed: %v", err)
	}
	userId := userRes.GetData().GetUserInfo().GetUserId()

	// 4. 分配角色给用户
	if _, err := env.roleSvc.AssignRoleToUser(ctx, &pb.AssignRoleRequest{
		UserId: userId, RoleIds: []int64{roleId}, GroupId: groupId,
	}); err != nil {
		t.Fatalf("assign role failed: %v", err)
	}

	// 5. 登录 (主流程)
	res, err := env.authSvc.LoginV2(ctx, &pb.LoginV2Request{
		Username: userRes.GetData().GetUserInfo().GetUsername(),
		Password: password,
	})
	if err != nil {
		t.Fatalf("LoginV2 failed: %v", err)
	}
	if res.GetData().GetToken() == "" {
		t.Fatalf("LoginV2 reply missing token: %+v", res)
	}
	log.Infof("LoginV2 ok, userId=%d tokenLen=%d groups=%d perms=%d isSuper=%v, groups=%+v perms=%+v",
		userId, len(res.GetData().GetToken()), len(res.GetData().GetGroups()), len(res.GetData().GetPermissions()), res.GetData().GetIsSuperAdmin(),
		res.GetData().GetGroups(), res.GetData().GetPermissions())

	// 6. 错误密码应失败
	if _, err := env.authSvc.LoginV2(ctx, &pb.LoginV2Request{
		Username: userRes.GetData().GetUserInfo().GetUsername(),
		Password: "WrongPwd99999",
	}); err == nil {
		t.Fatalf("LoginV2 with wrong password should fail")
	}
	log.Infof("LoginV2 wrong-password check passed")
}

// TestListMyAppsV2 总览查我可进的平台
// 前置: 自建 企业+角色+用户+分配角色+建产品+为企业开通产品
func TestListMyAppsV2(t *testing.T) {
	env := initTestV2Env()
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	// 1. 企业 + 角色
	groupRes, err := env.groupSvc.CreateGroup(ctx, &pb.CreateGroupRequest{
		Name:      uniqGroupName("v2apps租户"),
		GroupCode: fmt.Sprintf("VA%d", time.Now().UnixNano()),
	})
	if err != nil {
		t.Fatalf("prepare group failed: %v", err)
	}
	groupId := groupRes.GetData().GetGroup().GetGroupId()

	roleRes, err := env.roleSvc.CreateRole(ctx, &pb.CreateRoleV2Request{
		Name: uniqGroupName("v2apps角色"), GroupId: groupId,
	})
	if err != nil {
		t.Fatalf("prepare role failed: %v", err)
	}
	roleId := roleRes.GetData().GetRole().GetRoleId()

	// 2. 用户 + 分配角色
	userRes, err := env.userSvc.CreateUser(ctx, &pb.CreateUserV2Request{
		Username: fmt.Sprintf("v2apps_%d@test.com", time.Now().UnixNano()),
		Password: "Test12345678",
		Nickname: "v2apps用户",
	})
	if err != nil {
		t.Fatalf("prepare user failed: %v", err)
	}
	userId := userRes.GetData().GetUserInfo().GetUserId()
	if _, err := env.roleSvc.AssignRoleToUser(ctx, &pb.AssignRoleRequest{
		UserId: userId, RoleIds: []int64{roleId}, GroupId: groupId,
	}); err != nil {
		t.Fatalf("assign role failed: %v", err)
	}

	// 3. 产品注册 + 为企业配置权限范围 (用 appId 作为 permissionId 占位演示)
	// 注意: 正常流程应先在 permission 表建菜单节点, 再 SetGroupPermissions. 这里仅演示调用链路.
	appId := time.Now().Unix()
	if _, err := env.systemSvc.CreateApp(ctx, &pb.CreateAppRequest{
		Name: uniqGroupName("v2apps产品"), Code: fmt.Sprintf("VAP%d", appId),
		EntryUrl: "https://v2-apps.example.com",
		Detail:   `{}`, // app.detail 为 MySQL JSON 列, 须传合法 JSON
	}); err != nil {
		t.Fatalf("prepare app failed: %v", err)
	}
	if _, err := env.groupSvc.SetGroupPermissions(ctx, &pb.SetGroupPermissionsRequest{
		GroupId:       groupId,
		PermissionIds: []int64{appId},
	}); err != nil {
		t.Fatalf("set group permissions failed: %v", err)
	}

	// 4. 调用 ListMyApps (用该用户身份)
	userCtx := biz_common.NewReqCtx(ctx, userId, "")
	res, err := env.authSvc.ListMyApps(userCtx, &pb.ListMyAppsRequest{GroupId: groupId})
	if err != nil {
		t.Fatalf("ListMyApps failed: %v", err)
	}
	log.Infof("ListMyApps ok, userId=%d groupId=%d apps=%d, apps=%+v", userId, groupId, len(res.GetData().GetApps()), res.GetData().GetApps())
}

// TestGetUserInfoV2 登录后查询当前用户信息
// 前置: 自建 企业+角色+用户+分配角色 (普通员工路径)
// 覆盖: 非超管 groupId 必填校验、归属校验、只返回当前企业权限
func TestGetUserInfoV2(t *testing.T) {
	env := initTestV2Env()
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	// 1. 企业 + 角色
	groupRes, err := env.groupSvc.CreateGroup(ctx, &pb.CreateGroupRequest{
		Name:      uniqGroupName("v2info租户"),
		GroupCode: fmt.Sprintf("VI%d", time.Now().UnixNano()),
	})
	if err != nil {
		t.Fatalf("prepare group failed: %v", err)
	}
	groupId := groupRes.GetData().GetGroup().GetGroupId()

	roleRes, err := env.roleSvc.CreateRole(ctx, &pb.CreateRoleV2Request{
		Name: uniqGroupName("v2info角色"), GroupId: groupId,
	})
	if err != nil {
		t.Fatalf("prepare role failed: %v", err)
	}
	roleId := roleRes.GetData().GetRole().GetRoleId()

	// 2. 用户 + 分配角色
	userRes, err := env.userSvc.CreateUser(ctx, &pb.CreateUserV2Request{
		Username: fmt.Sprintf("v2info_%d@test.com", time.Now().UnixNano()),
		Password: "Test12345678",
		Nickname: "v2info用户",
	})
	if err != nil {
		t.Fatalf("prepare user failed: %v", err)
	}
	userId := userRes.GetData().GetUserInfo().GetUserId()
	if _, err := env.roleSvc.AssignRoleToUser(ctx, &pb.AssignRoleRequest{
		UserId: userId, RoleIds: []int64{roleId}, GroupId: groupId,
	}); err != nil {
		t.Fatalf("assign role failed: %v", err)
	}

	// 3. 以该用户身份请求 GetUserInfo
	userCtx := biz_common.NewReqCtx(ctx, userId, "")

	// 3a. 非超管 groupId=0 应失败 (params invalid)
	if _, err := env.authSvc.GetUserInfo(userCtx, &pb.GetUserInfoRequest{GroupId: 0}); err == nil {
		t.Fatalf("GetUserInfo with groupId=0 for non-super should fail")
	}
	log.Infof("GetUserInfo groupId=0 rejected as expected")

	// 3b. 非超管 传不归属的 groupId 应失败 (user not in group)
	if _, err := env.authSvc.GetUserInfo(userCtx, &pb.GetUserInfoRequest{GroupId: 999999999}); err == nil {
		t.Fatalf("GetUserInfo with unrelated groupId should fail")
	}
	log.Infof("GetUserInfo unrelated groupId rejected as expected")

	// 3c. 正常请求: 返回当前企业信息与有效权限
	res, err := env.authSvc.GetUserInfo(userCtx, &pb.GetUserInfoRequest{GroupId: groupId})
	if err != nil {
		t.Fatalf("GetUserInfo failed: %v", err)
	}
	if res.GetData().GetIsSuperAdmin() {
		t.Fatalf("non-super user should not be super admin")
	}
	if res.GetData().GetUserInfo().GetUserId() != userId {
		t.Fatalf("userId mismatch: got=%d want=%d", res.GetData().GetUserInfo().GetUserId(), userId)
	}
	if len(res.GetData().GetGroups()) != 1 || res.GetData().GetGroups()[0].GetGroupId() != groupId {
		t.Fatalf("groups should contain only current groupId=%d, got=%+v", groupId, res.GetData().GetGroups())
	}
	log.Infof("GetUserInfo ok, userId=%d groupId=%d isSuper=%v groups=%d perms=%d",
		userId, groupId, res.GetData().GetIsSuperAdmin(),
		len(res.GetData().GetGroups()), len(res.GetData().GetPermissions()))
}

// TestCheckUserGroupAccessV2 校验用户租户归属 (gRPC, 供下游/网关)
func TestCheckUserGroupAccessV2(t *testing.T) {
	env := initTestV2Env()
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	// 1. 企业 + 角色 + 用户A (分配角色) + 用户B (不分配)
	groupRes, err := env.groupSvc.CreateGroup(ctx, &pb.CreateGroupRequest{
		Name:      uniqGroupName("v2access租户"),
		GroupCode: fmt.Sprintf("VAC%d", time.Now().UnixNano()),
	})
	if err != nil {
		t.Fatalf("prepare group failed: %v", err)
	}
	groupId := groupRes.GetData().GetGroup().GetGroupId()

	roleRes, err := env.roleSvc.CreateRole(ctx, &pb.CreateRoleV2Request{
		Name: uniqGroupName("v2access角色"), GroupId: groupId,
	})
	if err != nil {
		t.Fatalf("prepare role failed: %v", err)
	}
	roleId := roleRes.GetData().GetRole().GetRoleId()

	userARes, err := env.userSvc.CreateUser(ctx, &pb.CreateUserV2Request{
		Username: fmt.Sprintf("v2accessA_%d@test.com", time.Now().UnixNano()),
		Password: "Test12345678", Nickname: "用户A",
	})
	if err != nil {
		t.Fatalf("prepare userA failed: %v", err)
	}
	userAId := userARes.GetData().GetUserInfo().GetUserId()
	if _, err := env.roleSvc.AssignRoleToUser(ctx, &pb.AssignRoleRequest{
		UserId: userAId, RoleIds: []int64{roleId}, GroupId: groupId,
	}); err != nil {
		t.Fatalf("assign role to userA failed: %v", err)
	}

	userBRes, err := env.userSvc.CreateUser(ctx, &pb.CreateUserV2Request{
		Username: fmt.Sprintf("v2accessB_%d@test.com", time.Now().UnixNano()),
		Password: "Test12345678", Nickname: "用户B",
	})
	if err != nil {
		t.Fatalf("prepare userB failed: %v", err)
	}
	userBId := userBRes.GetData().GetUserInfo().GetUserId()

	// 2. 用户A 有归属 → true
	resA, err := env.authSvc.CheckUserGroupAccess(ctx, &pb.CheckUserGroupAccessRequest{
		UserId: userAId, GroupId: groupId,
	})
	if err != nil {
		t.Fatalf("CheckUserGroupAccess(userA) failed: %v", err)
	}
	if !resA.GetData().GetHasAccess() {
		t.Fatalf("userA should have access, got false")
	}
	log.Infof("CheckUserGroupAccess(userA) ok, hasAccess=%v", resA.GetData().GetHasAccess())

	// 3. 用户B 无归属 → false
	resB, err := env.authSvc.CheckUserGroupAccess(ctx, &pb.CheckUserGroupAccessRequest{
		UserId: userBId, GroupId: groupId,
	})
	if err != nil {
		t.Fatalf("CheckUserGroupAccess(userB) failed: %v", err)
	}
	if resB.GetData().GetHasAccess() {
		t.Fatalf("userB should NOT have access, got true")
	}
	log.Infof("CheckUserGroupAccess(userB) ok, hasAccess=%v", resB.GetData().GetHasAccess())
}
