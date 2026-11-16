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

// role 测试需要的关联实体: 一个企业 + 一个角色 + 一个被分配的用户.
// 全部由 create 测试在文件内自建, 后续测试复用.
var (
	testRoleGroupId int64 // TestCreateRoleV2 创建的企业
	testRoleId      int64 // TestCreateRoleV2 创建的角色
	testRoleUserId  int64 // TestAssignRoleToUserV2 创建并分配角色的用户
)

// TestCreateRoleV2 为指定企业建角色 (前置: 自建一个企业供挂角色)
func TestCreateRoleV2(t *testing.T) {
	env := initTestV2Env()
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	// 1. 自建企业 (避免依赖其他文件的执行顺序)
	groupReq := &pb.CreateGroupRequest{
		Name:      uniqGroupName("v2role租户"),
		GroupCode: fmt.Sprintf("VR%d", time.Now().UnixNano()),
		Remarks:   "role单测企业",
	}
	groupRes, err := env.groupSvc.CreateGroup(ctx, groupReq)
	if err != nil {
		t.Fatalf("prepare group failed: %v", err)
	}
	testRoleGroupId = groupRes.GetData().GetGroup().GetGroupId()

	// 1.5 给企业建立权限 scope (SetGroupPermissions 不校验权限是否存在, 直接写 ref_group_permission).
	// CreateRole 绑定权限时须 ⊆ 企业 scope, 否则 403 PermissionOutOfScope.
	if _, err := env.groupSvc.SetGroupPermissions(ctx, &pb.SetGroupPermissionsRequest{
		GroupId:       testRoleGroupId,
		PermissionIds: []int64{1, 2, 3},
	}); err != nil {
		t.Fatalf("prepare group scope failed: %v", err)
	}

	// 2. 在该企业下建角色并绑定初始权限 [1,2]
	roleReq := &pb.CreateRoleV2Request{
		Name:          uniqGroupName("v2角色"),
		GroupId:       testRoleGroupId,
		Remarks:       "v2单测角色",
		PermissionIds: []int64{1, 2},
	}
	res, err := env.roleSvc.CreateRole(ctx, roleReq)
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}
	if res.GetData().GetRole() == nil || res.GetData().GetRole().GetRoleId() == 0 {
		t.Fatalf("CreateRole reply missing role: %+v", res)
	}
	testRoleId = res.GetData().GetRole().GetRoleId()
	log.Infof("CreateRole ok, groupId=%d roleId=%d", testRoleGroupId, testRoleId)
}

// TestUpdateRoleV2 编辑角色
func TestUpdateRoleV2(t *testing.T) {
	if testRoleId == 0 {
		t.Skip("依赖 TestCreateRoleV2 先创建角色")
	}
	ins := initTestV2Env().roleSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	req := &pb.UpdateRoleV2Request{
		RoleId:        testRoleId,
		Name:          uniqGroupName("v2角色-改"),
		Remarks:       "v2单测更新角色",
		PermissionIds: []int64{2, 3}, // 全量覆盖 (⊆ 企业 scope [1,2,3])
	}
	if _, err := ins.UpdateRole(ctx, req); err != nil {
		t.Fatalf("UpdateRole failed: %v", err)
	}
	log.Infof("UpdateRole ok, roleId=%d perms=%v", testRoleId, req.GetPermissionIds())
}

// TestListRolesByGroupV2 查企业下角色列表 (验证 GroupName/Permissions 装配 + groupName 模糊搜)
func TestListRolesByGroupV2(t *testing.T) {
	if testRoleGroupId == 0 {
		t.Skip("依赖 TestCreateRoleV2 先创建企业")
	}
	ins := initTestV2Env().roleSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	// 1. 按 groupId 精准搜, 校验 RoleDetailInfo 装配了 groupName 与 permissions
	res, err := ins.ListRolesByGroup(ctx, &pb.ListRolesByGroupRequest{
		GroupId:  testRoleGroupId,
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListRolesByGroup failed: %v", err)
	}
	if res.GetData().GetTotal() == 0 || len(res.GetData().GetList()) == 0 {
		t.Fatalf("ListRolesByGroup want roles, got total=%d len=%d", res.GetData().GetTotal(), len(res.GetData().GetList()))
	}
	// 找到本测试创建的角色, 校验企业名/权限已批量装配 (UpdateRole 已把权限全量覆盖为 [2,3])
	var got *pb.RoleDetailInfo
	for _, r := range res.GetData().GetList() {
		if r.GetRoleId() == testRoleId {
			got = r
			break
		}
	}
	if got == nil {
		t.Fatalf("ListRolesByGroup result missing roleId=%d", testRoleId)
	}
	if got.GetGroupName() == "" {
		t.Fatalf("RoleDetailInfo.groupName should be populated, got empty")
	}
	if len(got.GetPermissions()) == 0 {
		t.Fatalf("RoleDetailInfo.permissions should be populated (expect [2,3]), got empty")
	}
	log.Infof("ListRolesByGroup(byGroupId) ok, roleId=%d groupName=%s perms=%d, role=%+v",
		testRoleId, got.GetGroupName(), len(got.GetPermissions()), got)

	// 2. 按 groupName 模糊搜 (覆盖 cond.GroupName 分支), 应能命中本企业
	byName, err := ins.ListRolesByGroup(ctx, &pb.ListRolesByGroupRequest{
		GroupName: "v2role租户",
		Page:      1,
		PageSize:  10,
	})
	if err != nil {
		t.Fatalf("ListRolesByGroup(groupName) failed: %v", err)
	}
	if byName.GetData().GetTotal() == 0 {
		t.Fatalf("ListRolesByGroup(groupName) want matches, got total=0")
	}
	log.Infof("ListRolesByGroup(byGroupName=v2role租户) ok, total=%d, roles=%+v", byName.GetData().GetTotal(), byName.GetData().GetList())
}

// TestAssignRoleToUserV2 超管给企业加员工 (前置: 自建一个系统用户)
func TestAssignRoleToUserV2(t *testing.T) {
	if testRoleId == 0 || testRoleGroupId == 0 {
		t.Skip("依赖 TestCreateRoleV2 先创建角色/企业")
	}
	env := initTestV2Env()
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	// 1. 自建系统用户
	userReq := &pb.CreateUserV2Request{
		Username: fmt.Sprintf("v2role_%d@test.com", time.Now().UnixNano()),
		Password: "Test12345678",
		Nickname: "v2role用户",
	}
	userRes, err := env.userSvc.CreateUser(ctx, userReq)
	if err != nil {
		t.Fatalf("prepare user failed: %v", err)
	}
	testRoleUserId = userRes.GetData().GetUserInfo().GetUserId()

	// 2. 分配角色 (幂等)
	req := &pb.AssignRoleRequest{
		UserId:  testRoleUserId,
		RoleIds: []int64{testRoleId},
		GroupId: testRoleGroupId,
	}
	if _, err := env.roleSvc.AssignRoleToUser(ctx, req); err != nil {
		t.Fatalf("AssignRoleToUser failed: %v", err)
	}
	log.Infof("AssignRoleToUser ok, userId=%d roleId=%d groupId=%d",
		testRoleUserId, testRoleId, testRoleGroupId)
}

// TestListUserRolesByGroupV2 查某用户在某企业的角色
func TestListUserRolesByGroupV2(t *testing.T) {
	if testRoleUserId == 0 {
		t.Skip("依赖 TestAssignRoleToUserV2 先分配角色")
	}
	ins := initTestV2Env().roleSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	res, err := ins.ListUserRolesByGroup(ctx, &pb.ListUserRolesRequest{
		UserId:  testRoleUserId,
		GroupId: testRoleGroupId,
	})
	if err != nil {
		t.Fatalf("ListUserRolesByGroup failed: %v", err)
	}
	log.Infof("ListUserRolesByGroup ok, userId=%d groupId=%d roles=%d, roles=%+v",
		testRoleUserId, testRoleGroupId, len(res.GetData().GetRoles()), res.GetData().GetRoles())
}

// TestListUsersByGroupV2 查企业下员工列表 (验证 user/group 快照 + 按 (userId,groupId) 聚合角色)
func TestListUsersByGroupV2(t *testing.T) {
	if testRoleGroupId == 0 {
		t.Skip("依赖 TestCreateRoleV2 先创建企业")
	}
	ins := initTestV2Env().roleSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	res, err := ins.ListUsersByGroup(ctx, &pb.ListUsersByGroupRequest{
		GroupId:  testRoleGroupId,
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListUsersByGroup failed: %v", err)
	}
	// 找到本测试分配的员工, 校验快照字段与角色聚合 (该员工已被分配 testRoleId)
	var got *pb.GroupUserInfo
	for _, u := range res.GetData().GetList() {
		if u.GetUserId() == testRoleUserId && u.GetGroupId() == testRoleGroupId {
			got = u
			break
		}
	}
	if got == nil {
		t.Fatalf("ListUsersByGroup result missing userId=%d groupId=%d, total=%d",
			testRoleUserId, testRoleGroupId, res.GetData().GetTotal())
	}
	if got.GetUsername() == "" || got.GetGroupName() == "" || got.GetGroupCode() == "" {
		t.Fatalf("GroupUserInfo snapshot fields should be populated: %+v", got)
	}
	foundRole := false
	for _, r := range got.GetRoles() {
		if r.GetRoleId() == testRoleId {
			foundRole = true
			break
		}
	}
	if !foundRole {
		t.Fatalf("GroupUserInfo.roles should contain roleId=%d, got %+v", testRoleId, got.GetRoles())
	}
	log.Infof("ListUsersByGroup ok, total=%d, users=%+v", res.GetData().GetTotal(), res.GetData().GetList())
}

// TestRemoveRoleFromUserV2 移除用户角色绑定
func TestRemoveRoleFromUserV2(t *testing.T) {
	if testRoleUserId == 0 {
		t.Skip("依赖 TestAssignRoleToUserV2 先分配角色")
	}
	ins := initTestV2Env().roleSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	req := &pb.RemoveRoleRequest{
		UserId:  testRoleUserId,
		RoleId:  testRoleId,
		GroupId: testRoleGroupId,
	}
	if _, err := ins.RemoveRoleFromUser(ctx, req); err != nil {
		t.Fatalf("RemoveRoleFromUser failed: %v", err)
	}
	log.Infof("RemoveRoleFromUser ok, userId=%d roleId=%d", testRoleUserId, testRoleId)
}

// TestUpdateGroupEmployeeV2 修改企业员工的角色/状态/备注 (全量覆盖角色, 空=清空角色但不删 ref_user_group)
// 前置: TestAssignRoleToUserV2 已把 testRoleUserId 加入 testRoleGroupId;
//
//	TestRemoveRoleFromUserV2 已清空其角色绑定, 留下"在册无角色"状态便于验证全量覆盖语义.
func TestUpdateGroupEmployeeV2(t *testing.T) {
	if testRoleUserId == 0 || testRoleGroupId == 0 || testRoleId == 0 {
		t.Skip("依赖 TestAssignRoleToUserV2 先分配角色")
	}
	env := initTestV2Env()
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	// 1. 再建一个角色 role2 (同企业), 用于验证"全量覆盖"会移除旧角色绑定
	role2Res, err := env.roleSvc.CreateRole(ctx, &pb.CreateRoleV2Request{
		Name: uniqGroupName("v2角色2"), GroupId: testRoleGroupId,
	})
	if err != nil {
		t.Fatalf("prepare role2 failed: %v", err)
	}
	role2Id := role2Res.GetData().GetRole().GetRoleId()

	// 2. 全量覆盖为 [testRoleId], 同步改 status/remarks
	if _, err := env.roleSvc.UpdateGroupEmployee(ctx, &pb.UpdateGroupEmployeeRequest{
		UserId:  testRoleUserId,
		GroupId: testRoleGroupId,
		RoleIds: []int64{testRoleId},
		Status:  1,
		Remarks: "upd-by-test",
	}); err != nil {
		t.Fatalf("UpdateGroupEmployee(assign) failed: %v", err)
	}
	// 验证: 只剩 testRoleId
	listRes, err := env.roleSvc.ListUserRolesByGroup(ctx, &pb.ListUserRolesRequest{
		UserId: testRoleUserId, GroupId: testRoleGroupId,
	})
	if err != nil {
		t.Fatalf("ListUserRolesByGroup after assign failed: %v", err)
	}
	if len(listRes.GetData().GetRoles()) != 1 || listRes.GetData().GetRoles()[0].GetRoleId() != testRoleId {
		t.Fatalf("after assign want only [testRoleId=%d], got %+v", testRoleId, listRes.GetData().GetRoles())
	}

	// 3. 全量覆盖为 [role2Id] → 证明旧绑定 testRoleId 被移除 (而非追加)
	if _, err := env.roleSvc.UpdateGroupEmployee(ctx, &pb.UpdateGroupEmployeeRequest{
		UserId:  testRoleUserId,
		GroupId: testRoleGroupId,
		RoleIds: []int64{role2Id},
	}); err != nil {
		t.Fatalf("UpdateGroupEmployee(overwrite) failed: %v", err)
	}
	listRes2, err := env.roleSvc.ListUserRolesByGroup(ctx, &pb.ListUserRolesRequest{
		UserId: testRoleUserId, GroupId: testRoleGroupId,
	})
	if err != nil {
		t.Fatalf("ListUserRolesByGroup after overwrite failed: %v", err)
	}
	if len(listRes2.GetData().GetRoles()) != 1 || listRes2.GetData().GetRoles()[0].GetRoleId() != role2Id {
		t.Fatalf("after overwrite want only [role2Id=%d], got %+v", role2Id, listRes2.GetData().GetRoles())
	}

	// 4. 空 roleIds → 清空角色 (ref_user_group 不删)
	if _, err := env.roleSvc.UpdateGroupEmployee(ctx, &pb.UpdateGroupEmployeeRequest{
		UserId:  testRoleUserId,
		GroupId: testRoleGroupId,
		RoleIds: nil,
	}); err != nil {
		t.Fatalf("UpdateGroupEmployee(clear) failed: %v", err)
	}
	listRes3, err := env.roleSvc.ListUserRolesByGroup(ctx, &pb.ListUserRolesRequest{
		UserId: testRoleUserId, GroupId: testRoleGroupId,
	})
	if err != nil {
		t.Fatalf("ListUserRolesByGroup after clear failed: %v", err)
	}
	if len(listRes3.GetData().Roles) != 0 {
		t.Fatalf("after clear want no roles, got %+v", listRes3.GetData().Roles)
	}
	log.Infof("UpdateGroupEmployee ok (assign→overwrite→clear), userId=%d", testRoleUserId)
}

// TestDeleteRoleV2 删除角色 (System=1 不可删; 放在文件末尾清理)
func TestDeleteRoleV2(t *testing.T) {
	if testRoleId == 0 {
		t.Skip("依赖 TestCreateRoleV2 先创建角色")
	}
	ins := initTestV2Env().roleSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	if _, err := ins.DeleteRole(ctx, &pb.DeleteRoleV2Request{RoleId: testRoleId}); err != nil {
		t.Fatalf("DeleteRole failed: %v", err)
	}
	log.Infof("DeleteRole ok, roleId=%d", testRoleId)
	// 顺带清理自建的企业
	if testRoleGroupId != 0 {
		groupIns := initTestV2Env().groupSvc
		if _, e := groupIns.DeleteGroup(ctx, &pb.DeleteGroupRequest{GroupId: testRoleGroupId}); e != nil {
			log.Infof("cleanup group err: %v", e)
		}
	}
	testRoleId = 0
	testRoleGroupId = 0
	testRoleUserId = 0
}
