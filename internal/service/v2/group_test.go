package v2

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	pb "wuwen.com/wwuc/api_gen/user/v2"

	biz_common "wuwen.com/wwuc/internal/biz/common"
)

// testGroupId / testGroupName / testGroupCode 由 TestCreateGroupV2 创建后回填,
// 供同文件后续 get/update/list/delete 测试复用.
var (
	testGroupId   int64  = 1848974093960425900
	testGroupName string = "v2testGroup_1786172028546113000"
	testGroupCode string = "VC1786172028546118000"
)

// uniqGroupName 生成唯一企业名, 保证 create 测试可重复运行.
func uniqGroupName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// int64SliceSameSet 比较两个 int64 切片是否含相同元素集合 (忽略顺序).
// SetGroupPermissions 回查的 permission_ids 顺序依赖 DB 主键序, 用集合比较更稳健.
func int64SliceSameSet(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	sa := append([]int64(nil), a...)
	sb := append([]int64(nil), b...)
	sort.Slice(sa, func(i, j int) bool { return sa[i] < sa[j] })
	sort.Slice(sb, func(i, j int) bool { return sb[i] < sb[j] })
	for i := range sa {
		if sa[i] != sb[i] {
			return false
		}
	}
	return true
}

// TestCreateGroupV2 超管创建企业 (对应 design 2.2 step1)
func TestCreateGroupV2(t *testing.T) {
	ins := initTestV2Env().groupSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	name := uniqGroupName("v2testGroup")
	code := fmt.Sprintf("VC%d", time.Now().UnixNano())
	req := &pb.CreateGroupRequest{
		Name:      name,
		GroupCode: code,
		Remarks:   "v2单测创建",
	}
	res, err := ins.CreateGroup(ctx, req)
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	g := res.GetData().GetGroup()
	if g == nil || g.GetGroupId() == 0 {
		t.Fatalf("CreateGroup reply missing group: %+v", res)
	}
	// 关键逻辑: 校验回填字段 GroupName/GroupCode/Status/GroupCreatedAt (企业创建时间)
	if g.GetGroupName() != name {
		t.Fatalf("CreateGroup groupName want %s, got %s", name, g.GetGroupName())
	}
	if g.GetGroupCode() != code {
		t.Fatalf("CreateGroup groupCode want %s, got %s", code, g.GetGroupCode())
	}
	if g.GetStatus() != 1 {
		t.Fatalf("CreateGroup status want 1(启用), got %d", g.GetStatus())
	}
	if g.GetGroupCreatedAt() == 0 {
		t.Fatalf("CreateGroup groupCreatedAt should be populated (企业创建时间), got 0")
	}
	testGroupId = g.GetGroupId()
	testGroupName = name
	testGroupCode = code
	log.Infof("CreateGroup ok, groupId=%d name=%s groupCreatedAt=%d", testGroupId, name, g.GetGroupCreatedAt())
}

// TestGetGroupV2 企业详情 (校验回显创建时的 name/code/status)
func TestGetGroupV2(t *testing.T) {
	if testGroupId == 0 {
		t.Skip("依赖 TestCreateGroupV2 先创建企业")
	}
	ins := initTestV2Env().groupSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	res, err := ins.GetGroup(ctx, &pb.GetGroupRequest{GroupId: testGroupId})
	if err != nil {
		t.Fatalf("GetGroup failed: %v", err)
	}
	g := res.GetData().GetGroup()
	if g == nil {
		t.Fatalf("GetGroup reply missing group")
	}
	if g.GetGroupName() != testGroupName {
		t.Fatalf("GetGroup groupName want %s, got %s", testGroupName, g.GetGroupName())
	}
	if g.GetGroupCode() != testGroupCode {
		t.Fatalf("GetGroup groupCode want %s, got %s", testGroupCode, g.GetGroupCode())
	}
	if g.GetStatus() != 1 {
		t.Fatalf("GetGroup status want 1, got %d", g.GetStatus())
	}
	log.Infof("GetGroup ok, groupId=%d, group=%+v", testGroupId, g)
}

// TestUpdateGroupV2 编辑企业:
//   - status=0 视为未变更 (改名后状态仍为 1)
//   - status=2 停用 → 回查应为 2
//   - status=1 重新启用, 恢复后续测试依赖
func TestUpdateGroupV2(t *testing.T) {
	if testGroupId == 0 {
		t.Skip("依赖 TestCreateGroupV2 先创建企业")
	}
	ins := initTestV2Env().groupSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	// 1. 改名 + status=0(不变更状态)
	newName := uniqGroupName("v2testGroup-改")
	if _, err := ins.UpdateGroup(ctx, &pb.UpdateGroupRequest{
		GroupId: testGroupId,
		Name:    newName,
		Remarks: "v2单测更新",
		Status:  0,
	}); err != nil {
		t.Fatalf("UpdateGroup(rename, status=0) failed: %v", err)
	}
	testGroupName = newName
	got, err := ins.GetGroup(ctx, &pb.GetGroupRequest{GroupId: testGroupId})
	if err != nil {
		t.Fatalf("GetGroup after rename failed: %v", err)
	}
	if got.GetData().GetGroup().GetGroupName() != newName {
		t.Fatalf("after rename want name=%s, got %s", newName, got.GetData().GetGroup().GetGroupName())
	}
	if got.GetData().GetGroup().GetStatus() != 1 {
		t.Fatalf("status=0 should keep status=1, got %d", got.GetData().GetGroup().GetStatus())
	}

	// 2. 停用 (status=2) → 回查应为 2
	if _, err := ins.UpdateGroup(ctx, &pb.UpdateGroupRequest{
		GroupId: testGroupId,
		Status:  2,
	}); err != nil {
		t.Fatalf("UpdateGroup(disable status=2) failed: %v", err)
	}
	got2, err := ins.GetGroup(ctx, &pb.GetGroupRequest{GroupId: testGroupId})
	if err != nil {
		t.Fatalf("GetGroup after disable failed: %v", err)
	}
	if got2.GetData().GetGroup().GetStatus() != 2 {
		t.Fatalf("after disable status want 2, got %d", got2.GetData().GetGroup().GetStatus())
	}

	// 3. 重新启用 (status=1), 恢复后续测试依赖
	if _, err := ins.UpdateGroup(ctx, &pb.UpdateGroupRequest{
		GroupId: testGroupId,
		Status:  1,
	}); err != nil {
		t.Fatalf("UpdateGroup(enable status=1) failed: %v", err)
	}
	log.Infof("UpdateGroup ok, groupId=%d name=%s", testGroupId, newName)
}

// TestListGroupsV2 企业分页列表 (覆盖 name/groupCode/id/status 过滤条件构建分支)
func TestListGroupsV2(t *testing.T) {
	ins := initTestV2Env().groupSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	// 1. 无过滤分页查询
	res, err := ins.ListGroups(ctx, &pb.ListGroupsRequest{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListGroups failed: %v", err)
	}
	log.Infof("ListGroups(all) ok, total=%d len=%d, groups=%+v", res.GetData().GetTotal(), len(res.GetData().GetList()), res.GetData().GetList())

	if testGroupId != 0 {
		// 2. 按 id 精准过滤 (覆盖 cond.Ids 分支)
		byId, err := ins.ListGroups(ctx, &pb.ListGroupsRequest{Page: 1, PageSize: 10, Id: testGroupId})
		if err != nil {
			t.Fatalf("ListGroups(id) failed: %v", err)
		}
		if byId.GetData().GetTotal() == 0 {
			t.Fatalf("ListGroups(id=%d) want match, got total=0", testGroupId)
		}
		log.Infof("ListGroups(byId=%d) ok, total=%d, groups=%+v", testGroupId, byId.GetData().GetTotal(), byId.GetData().GetList())

		// 3. 按 ids 批量过滤 (覆盖 cond.Ids 批量分支), 命中 testGroupId 并排除不存在 id
		byIds, err := ins.ListGroups(ctx, &pb.ListGroupsRequest{Page: 1, PageSize: 10, Ids: []int64{testGroupId, 999999999}})
		if err != nil {
			t.Fatalf("ListGroups(ids) failed: %v", err)
		}
		if byIds.GetData().GetTotal() != 1 || byIds.GetData().GetList()[0].GetGroupId() != testGroupId {
			t.Fatalf("ListGroups(ids=[%d,999999999]) want only groupId=%d, got total=%d list=%+v",
				testGroupId, testGroupId, byIds.GetData().GetTotal(), byIds.GetData().GetList())
		}
		log.Infof("ListGroups(byIds=[%d,999999999]) ok, total=%d", testGroupId, byIds.GetData().GetTotal())

		// 4. ids + id 合并过滤 (覆盖合并分支)
		byIdsAndId, err := ins.ListGroups(ctx, &pb.ListGroupsRequest{Page: 1, PageSize: 10, Ids: []int64{testGroupId}, Id: 999999999})
		if err != nil {
			t.Fatalf("ListGroups(ids+id) failed: %v", err)
		}
		if byIdsAndId.GetData().GetTotal() != 1 {
			t.Fatalf("ListGroups(ids=[%d],id=999999999) want merged total=1, got total=%d",
				testGroupId, byIdsAndId.GetData().GetTotal())
		}
		log.Infof("ListGroups(byIds+Id merged) ok, total=%d", byIdsAndId.GetData().GetTotal())

		// 5. 按 name 模糊过滤 (覆盖 cond.Name 分支), 用创建名的固定前缀片段命中
		byName, err := ins.ListGroups(ctx, &pb.ListGroupsRequest{Page: 1, PageSize: 10, Name: "v2testGroup"})
		if err != nil {
			t.Fatalf("ListGroups(name) failed: %v", err)
		}
		if byName.GetData().GetTotal() == 0 {
			t.Fatalf("ListGroups(name=v2testGroup) want matches, got total=0")
		}
		log.Infof("ListGroups(byName=v2testGroup) ok, total=%d, groups=%+v", byName.GetData().GetTotal(), byName.GetData().GetList())

		// 6. 按 groupCode 精准过滤 (覆盖 cond.GroupCode 分支)
		byCode, err := ins.ListGroups(ctx, &pb.ListGroupsRequest{Page: 1, PageSize: 10, GroupCode: testGroupCode})
		if err != nil {
			t.Fatalf("ListGroups(groupCode) failed: %v", err)
		}
		if byCode.GetData().GetTotal() == 0 {
			t.Fatalf("ListGroups(groupCode=%s) want match, got total=0", testGroupCode)
		}
		log.Infof("ListGroups(byGroupCode=%s) ok, total=%d, groups=%+v", testGroupCode, byCode.GetData().GetTotal(), byCode.GetData().GetList())
	}

	// 7. 按 status=1 过滤 (覆盖 cond.Status 分支)
	byStatus, err := ins.ListGroups(ctx, &pb.ListGroupsRequest{Page: 1, PageSize: 10, Status: 1})
	if err != nil {
		t.Fatalf("ListGroups(status) failed: %v", err)
	}
	log.Infof("ListGroups(byStatus=1) ok, total=%d, groups=%+v", byStatus.GetData().GetTotal(), byStatus.GetData().GetList())
}

// TestListGroupPermissionsV2 查企业权限范围 (新建企业预期为空)
func TestListGroupPermissionsV2(t *testing.T) {
	if testGroupId == 0 {
		t.Skip("依赖 TestCreateGroupV2 先创建企业")
	}
	ins := initTestV2Env().groupSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	res, err := ins.ListGroupPermissions(ctx, &pb.ListGroupPermissionsRequest{GroupId: testGroupId})
	if err != nil {
		t.Fatalf("ListGroupPermissions failed: %v", err)
	}
	// 关键逻辑: 新建企业尚未配置权限范围, 应返回空
	if len(res.GetData().GetPermissionIds()) != 0 {
		t.Fatalf("ListGroupPermissions new group want empty, got %v", res.GetData().GetPermissionIds())
	}
	log.Infof("ListGroupPermissions ok, groupId=%d permIds=%+v (empty as expected)", testGroupId, res.GetData().GetPermissionIds())
}

// TestSetGroupPermissionsV2 配置企业权限范围 (key logic: 全量覆盖 + 级联软删越界角色权限):
//   - 全量覆盖为 [1,2] → 企业 scope 回查应为 [1,2]
//   - 在该企业建一个角色并绑定 [1,2] (⊆ scope)
//   - 再次全量覆盖为 [2,3] → 企业 scope 变为 [2,3], 且角色的 perm-1 绑定应被级联软删
//     (验证 CascadeDeleteRolePermissions 真实删除越界的 ref_role_permission 记录)
func TestSetGroupPermissionsV2(t *testing.T) {
	if testGroupId == 0 {
		t.Skip("依赖 TestCreateGroupV2 先创建企业")
	}
	env := initTestV2Env()
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	// 1. 全量覆盖为 [1, 2]
	if _, err := env.groupSvc.SetGroupPermissions(ctx, &pb.SetGroupPermissionsRequest{
		GroupId:       testGroupId,
		PermissionIds: []int64{1, 2},
	}); err != nil {
		t.Fatalf("SetGroupPermissions([1,2]) failed: %v", err)
	}
	got, err := env.groupSvc.ListGroupPermissions(ctx, &pb.ListGroupPermissionsRequest{GroupId: testGroupId})
	if err != nil {
		t.Fatalf("ListGroupPermissions after set [1,2] failed: %v", err)
	}
	if !int64SliceSameSet(got.GetData().GetPermissionIds(), []int64{1, 2}) {
		t.Fatalf("after set [1,2] want {1,2}, got %v", got.GetData().GetPermissionIds())
	}

	// 2. 在该企业建角色并绑定 [1,2] (须 ⊆ 企业 scope, 否则 403)
	roleRes, err := env.roleSvc.CreateRole(ctx, &pb.CreateRoleV2Request{
		Name:          uniqGroupName("v2cascade角色"),
		GroupId:       testGroupId,
		PermissionIds: []int64{1, 2},
	})
	if err != nil {
		t.Fatalf("CreateRole for cascade failed: %v", err)
	}
	cascadeRoleId := roleRes.GetData().GetRole().GetRoleId()
	if !int64SliceSameSet(rolePermissionIds(ctx, env, testGroupId, cascadeRoleId), []int64{1, 2}) {
		t.Fatalf("role perms before cascade want {1,2}, got %v",
			rolePermissionIds(ctx, env, testGroupId, cascadeRoleId))
	}

	// 3. 全量覆盖企业 scope 为 [2, 3] → perm 1 越界, 角色的 perm-1 绑定应被级联软删
	if _, err := env.groupSvc.SetGroupPermissions(ctx, &pb.SetGroupPermissionsRequest{
		GroupId:       testGroupId,
		PermissionIds: []int64{2, 3},
	}); err != nil {
		t.Fatalf("SetGroupPermissions([2,3]) failed: %v", err)
	}
	// 3a. 企业 scope 应为 [2,3]
	got2, err := env.groupSvc.ListGroupPermissions(ctx, &pb.ListGroupPermissionsRequest{GroupId: testGroupId})
	if err != nil {
		t.Fatalf("ListGroupPermissions after overwrite [2,3] failed: %v", err)
	}
	if !int64SliceSameSet(got2.GetData().GetPermissionIds(), []int64{2, 3}) {
		t.Fatalf("after overwrite [2,3] want {2,3}, got %v", got2.GetData().GetPermissionIds())
	}
	// 3b. 角色权限应只剩 [2] (perm 1 被级联软删; perm 3 不会自动加到角色)
	rolePermsAfter := rolePermissionIds(ctx, env, testGroupId, cascadeRoleId)
	if !int64SliceSameSet(rolePermsAfter, []int64{2}) {
		t.Fatalf("after cascade want role perms {2} (perm 1 cascade-deleted), got %v", rolePermsAfter)
	}
	log.Infof("SetGroupPermissions ok, groupId=%d scope=%v rolePerms=%v (perm 1 cascade-deleted)",
		testGroupId, got2.GetData().GetPermissionIds(), rolePermsAfter)
}

// rolePermissionIds 通过 ListRolesByGroup 读取指定角色的权限 id 集合 (用于验证级联软删效果).
func rolePermissionIds(ctx context.Context, env *testV2Env, groupId, roleId int64) []int64 {
	res, err := env.roleSvc.ListRolesByGroup(ctx, &pb.ListRolesByGroupRequest{
		GroupId: groupId, Page: 1, PageSize: 100,
	})
	if err != nil {
		return nil
	}
	for _, r := range res.GetData().GetList() {
		if r.GetRoleId() == roleId {
			out := make([]int64, 0, len(r.GetPermissions()))
			for _, p := range r.GetPermissions() {
				out = append(out, p.GetId())
			}
			return out
		}
	}
	return nil
}

// TestDeleteGroupV2 软删企业 (放在文件末尾, 清理本测试创建的企业)
func TestDeleteGroupV2(t *testing.T) {
	if testGroupId == 0 {
		t.Skip("依赖 TestCreateGroupV2 先创建企业")
	}
	ins := initTestV2Env().groupSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	if _, err := ins.DeleteGroup(ctx, &pb.DeleteGroupRequest{GroupId: testGroupId}); err != nil {
		t.Fatalf("DeleteGroup failed: %v", err)
	}
	// 关键逻辑: 软删后再查应返回 not-found 错误
	if _, err := ins.GetGroup(ctx, &pb.GetGroupRequest{GroupId: testGroupId}); err == nil {
		t.Fatalf("GetGroup after delete want not-found error, got nil")
	}
	log.Infof("DeleteGroup ok, groupId=%d", testGroupId)
	testGroupId = 0
	testGroupName = ""
	testGroupCode = ""
}
