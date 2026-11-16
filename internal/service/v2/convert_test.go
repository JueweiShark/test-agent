package v2

import (
	"testing"
	"time"

	pb "wuwen.com/wwuc/api_gen/user/v2"
	bizV2 "wuwen.com/wwuc/internal/biz/v2"
)

// convert_test.go 覆盖 service 层的纯转换/装配函数 (convert.go + group.go/user.go/menu.go 内的 helper).
// 这些函数不依赖 DB/Redis, 直接构造 biz 入参校验 proto 出参, 是 service 层关键逻辑的单测主体.

// --- 指针 helper (构造 biz 结构里的可选字段) ---
func int64Ptr(v int64) *int64 { return &v }

// TestStrPtrOrNil 空串返回 nil, 非空返回 *string
func TestStrPtrOrNil(t *testing.T) {
	if got := strPtrOrNil(""); got != nil {
		t.Fatalf("strPtrOrNil(\"\") want nil, got %v", got)
	}
	got := strPtrOrNil("GET")
	if got == nil || *got != "GET" {
		t.Fatalf("strPtrOrNil(\"GET\") want *\"GET\", got %v", got)
	}
}

// TestBuildMenuTree 递归构建菜单树 (含空根 / 多层子节点)
func TestBuildMenuTree(t *testing.T) {
	// 1. 空 roots → 空(非 nil) 切片
	if got := buildMenuTree(nil, nil); len(got) != 0 {
		t.Fatalf("buildMenuTree(nil) want empty slice, got %v", got)
	}

	// 2. 两层树: root(1) -> child(2) -> leaf(3); root(4) 无子节点
	// 注意: dependence_parents 存的是父节点的 code, 因此 children map 的 key 用 code.
	root1 := &bizV2.Permission{Id: 1, Code: 100, Name: "root1"}
	root4 := &bizV2.Permission{Id: 4, Code: 400, Name: "root4"}
	roots := []*bizV2.Permission{root1, root4}
	child2 := &bizV2.Permission{Id: 2, Code: 200, DependenceParents: 100, Name: "child2"}
	leaf3 := &bizV2.Permission{Id: 3, Code: 300, DependenceParents: 200, Name: "leaf3"}
	children := map[int64][]*bizV2.Permission{
		100: {child2}, // key = root1.Code
		200: {leaf3},  // key = child2.Code
	}

	tree := buildMenuTree(roots, children)
	if len(tree) != 2 {
		t.Fatalf("want 2 roots, got %d", len(tree))
	}
	// root1 -> child2 -> leaf3
	if tree[0].Info.Name != "root1" {
		t.Fatalf("tree[0] want root1, got %s", tree[0].Info.Name)
	}
	if len(tree[0].Children) != 1 || tree[0].Children[0].Info.Name != "child2" {
		t.Fatalf("root1 child want child2, got %+v", tree[0].Children)
	}
	if len(tree[0].Children[0].Children) != 1 || tree[0].Children[0].Children[0].Info.Name != "leaf3" {
		t.Fatalf("child2 leaf want leaf3, got %+v", tree[0].Children[0].Children)
	}
	// root4 无子节点 → 空(非 nil) 切片
	if tree[1].Info.Name != "root4" {
		t.Fatalf("tree[1] want root4, got %s", tree[1].Info.Name)
	}
	if len(tree[1].Children) != 0 {
		t.Fatalf("root4 want no children, got %d", len(tree[1].Children))
	}
}

// TestToPermissionInfo 权限转换 (含可选字段 nil/非 nil 与 hidden 默认值)
func TestToPermissionInfo(t *testing.T) {
	// nil 入参
	if got := toPermissionInfo(nil); got != nil {
		t.Fatalf("toPermissionInfo(nil) want nil, got %v", got)
	}

	// 全字段填充
	menuType := uint8(2)
	act := "POST"
	hidden := uint8(0)
	desc := "备注"
	p := &bizV2.Permission{
		Id: 11, Name: "菜单A", Type: 1, MenuType: &menuType, RoutePath: "/r", Icon: "i",
		Sort: 5, Hidden: &hidden, App: 7, Path: "/api", Act: &act, Description: &desc,
		DependenceChildren: []int64{22}, DependenceParents: 99, Code: 1234,
	}
	got := toPermissionInfo(p)
	if got.Id != 11 || got.Name != "菜单A" || got.Type != 1 || got.MenuType != 2 {
		t.Fatalf("basic fields mismatch: %+v", got)
	}
	if got.RoutePath != "/r" || got.Icon != "i" || got.Sort != 5 || got.App != 7 {
		t.Fatalf("route fields mismatch: %+v", got)
	}
	if got.Path != "/api" || got.Act != "POST" || got.Code != 1234 {
		t.Fatalf("path/act/code mismatch: %+v", got)
	}
	if got.Hidden != 0 || got.Description != "备注" {
		t.Fatalf("hidden/desc mismatch: %+v", got)
	}
	if len(got.DependenceChildren) != 1 || got.DependenceChildren[0] != 22 || got.DependenceParents != 99 {
		t.Fatalf("dependence mismatch: %+v", got)
	}

	// 可选字段全 nil: MenuType=0, Act="", Hidden=1(默认显示), Description=""
	got2 := toPermissionInfo(&bizV2.Permission{Id: 1, Name: "n"})
	if got2.MenuType != 0 || got2.Act != "" || got2.Hidden != 1 || got2.Description != "" {
		t.Fatalf("default optional fields mismatch: %+v", got2)
	}
}

// TestToAppInfo 产品转换 (nil / 无 Id / 有 Id)
func TestToAppInfo(t *testing.T) {
	if got := toAppInfo(nil); got != nil {
		t.Fatalf("toAppInfo(nil) want nil, got %v", got)
	}
	// 无 Id
	a := &bizV2.App{Name: "产品", Code: "C", EntryUrl: "u", Icon: "i", Sort: 3, Status: 1, Type: "t", Desc: "d"}
	got := toAppInfo(a)
	if got.Id != 0 || got.Name != "产品" || got.Code != "C" || got.EntryUrl != "u" {
		t.Fatalf("app fields mismatch: %+v", got)
	}
	if got.Icon != "i" || got.Sort != 3 || got.Status != 1 || got.Type != "t" || got.Desc != "d" {
		t.Fatalf("app fields mismatch: %+v", got)
	}
	// 有 Id
	got2 := toAppInfo(&bizV2.App{Id: int64Ptr(9), Name: "x"})
	if got2.Id != 9 {
		t.Fatalf("app Id want 9, got %d", got2.Id)
	}
}

// TestToRoleInfo 角色基本信息转换 (nil / Id=nil / 正常)
func TestToRoleInfo(t *testing.T) {
	if got := toRoleInfo(nil); got != nil {
		t.Fatalf("toRoleInfo(nil) want nil, got %v", got)
	}
	if got := toRoleInfo(&bizV2.Role{}); got != nil {
		t.Fatalf("toRoleInfo(Id=nil) want nil, got %v", got)
	}
	r := &bizV2.Role{Id: int64Ptr(5), Name: "角色", Code: "role00005", Remarks: "r", System: 1, GroupId: 8, App: 2}
	got := toRoleInfo(r)
	if got.RoleId != 5 || got.RoleName != "角色" || got.RoleCode != "role00005" {
		t.Fatalf("role fields mismatch: %+v", got)
	}
	if got.Remarks != "r" || got.System != 1 || got.GroupId != 8 || got.App != 2 {
		t.Fatalf("role fields mismatch: %+v", got)
	}
}

// TestToRoleDetailInfo 角色详情转换 (含企业名/权限/时间戳)
func TestToRoleDetailInfo(t *testing.T) {
	if got := toRoleDetailInfo(nil, "g", nil); got != nil {
		t.Fatalf("toRoleDetailInfo(nil) want nil, got %v", got)
	}
	if got := toRoleDetailInfo(&bizV2.Role{}, "g", nil); got != nil {
		t.Fatalf("toRoleDetailInfo(Id=nil) want nil, got %v", got)
	}
	now := time.Now()
	r := &bizV2.Role{Id: int64Ptr(7), Name: "R", Code: "c", Remarks: "rm", System: 0, GroupId: 3, App: 4, CreateAt: &now, UpdateAt: &now}
	perms := []*pb.PermissionInfo{{Id: 1, Name: "p1"}}
	got := toRoleDetailInfo(r, "企业X", perms)
	if got.RoleId != 7 || got.GroupId != 3 || got.GroupName != "企业X" || got.RoleName != "R" {
		t.Fatalf("detail fields mismatch: %+v", got)
	}
	if got.RoleCode != "c" || got.Remarks != "rm" || got.System != 0 || got.App != 4 {
		t.Fatalf("detail fields mismatch: %+v", got)
	}
	if len(got.Permissions) != 1 || got.Permissions[0].Id != 1 {
		t.Fatalf("permissions mismatch: %+v", got.Permissions)
	}
	if got.CreatedAt != now.Unix() || got.UpdatedAt != now.Unix() {
		t.Fatalf("timestamps mismatch: created=%d updated=%d", got.CreatedAt, got.UpdatedAt)
	}
	// 时间为 nil → 时间戳为 0
	got2 := toRoleDetailInfo(&bizV2.Role{Id: int64Ptr(1)}, "", nil)
	if got2.CreatedAt != 0 || got2.UpdatedAt != 0 {
		t.Fatalf("nil timestamps want 0, got created=%d updated=%d", got2.CreatedAt, got2.UpdatedAt)
	}
}

// TestToUserInfo 用户基本信息转换 (nil / Id=nil / 正常)
func TestToUserInfo(t *testing.T) {
	if got := toUserInfo(nil); got != nil {
		t.Fatalf("toUserInfo(nil) want nil, got %v", got)
	}
	if got := toUserInfo(&bizV2.User{}); got != nil {
		t.Fatalf("toUserInfo(Id=nil) want nil, got %v", got)
	}
	u := &bizV2.User{Id: int64Ptr(12), Username: "u", Nickname: "n"}
	got := toUserInfo(u)
	if got.UserId != 12 || got.Username != "u" || got.Nickname != "n" {
		t.Fatalf("userInfo mismatch: %+v", got)
	}
}

// TestToUserDetailInfo 用户详情转换 (含零值时间 / LastLoginAt nil)
func TestToUserDetailInfo(t *testing.T) {
	if got := toUserDetailInfo(nil); got != nil {
		t.Fatalf("toUserDetailInfo(nil) want nil, got %v", got)
	}
	if got := toUserDetailInfo(&bizV2.User{}); got != nil {
		t.Fatalf("toUserDetailInfo(Id=nil) want nil, got %v", got)
	}
	// CreatedAt 零值 → 不填; LastLoginAt nil → 不填
	u := &bizV2.User{Id: int64Ptr(1), Username: "u", Nickname: "n", Status: 1}
	got := toUserDetailInfo(u)
	if got.UserId != 1 || got.Username != "u" || got.Nickname != "n" || got.Status != 1 {
		t.Fatalf("detail mismatch: %+v", got)
	}
	if got.CreatedAt != 0 || got.LastLoginAt != 0 {
		t.Fatalf("zero-time timestamps want 0, got created=%d last=%d", got.CreatedAt, got.LastLoginAt)
	}
	// CreatedAt 非零 + LastLoginAt 非 nil → 填时间戳
	now := time.Now()
	u2 := &bizV2.User{Id: int64Ptr(2), CreatedAt: now, LastLoginAt: &now}
	got2 := toUserDetailInfo(u2)
	if got2.CreatedAt != now.Unix() || got2.LastLoginAt != now.Unix() {
		t.Fatalf("timestamps mismatch: created=%d last=%d", got2.CreatedAt, got2.LastLoginAt)
	}
}

// TestToGroupGroupInfo 企业信息转换 (group.go, 含 apps 与 GroupCreatedAt 可选)
func TestToGroupGroupInfo(t *testing.T) {
	if got := toGroupGroupInfo(nil, nil); got != nil {
		t.Fatalf("toGroupGroupInfo(nil) want nil, got %v", got)
	}
	if got := toGroupGroupInfo(&bizV2.Group{}, nil); got != nil {
		t.Fatalf("toGroupGroupInfo(Id=nil) want nil, got %v", got)
	}
	// 无 GroupCreatedAt/CreatedAt/UpdatedAt → 三个时间戳均为 0; apps 含 nil 元素被跳过
	g := &bizV2.Group{Id: int64Ptr(3), Name: "企业", GroupCode: "GC", Status: 1}
	got := toGroupGroupInfo(g, []*bizV2.App{nil, {Id: int64Ptr(5), Name: "a"}})
	if got.GroupId != 3 || got.GroupName != "企业" || got.GroupCode != "GC" || got.Status != 1 {
		t.Fatalf("group fields mismatch: %+v", got)
	}
	if got.GroupCreatedAt != 0 || got.CreatedAt != 0 || got.UpdatedAt != 0 {
		t.Fatalf("nil timestamps want all 0, got groupCreated=%d created=%d updated=%d",
			got.GroupCreatedAt, got.CreatedAt, got.UpdatedAt)
	}
	if len(got.Apps) != 1 || got.Apps[0].Id != 5 {
		t.Fatalf("apps mismatch (nil should be skipped): %+v", got.Apps)
	}
	// 有 GroupCreatedAt → 填 GroupCreatedAt (field 4, 企业创建时间); 无 CreatedAt → CreatedAt=0
	now := time.Now()
	g2 := &bizV2.Group{Id: int64Ptr(4), GroupCreatedAt: &now}
	got2 := toGroupGroupInfo(g2, nil)
	if got2.GroupCreatedAt != now.Unix() {
		t.Fatalf("GroupCreatedAt timestamp mismatch: got %d", got2.GroupCreatedAt)
	}
	if got2.CreatedAt != 0 {
		t.Fatalf("nil CreatedAt want 0, got %d", got2.CreatedAt)
	}
	if len(got2.Apps) != 0 {
		t.Fatalf("nil apps want empty slice, got %d", len(got2.Apps))
	}
	// GroupCreatedAt/CreatedAt/UpdatedAt 同时存在 → 三个字段各自独立填充 (不可混淆)
	cAt, uAt := time.Now(), time.Now()
	g3 := &bizV2.Group{Id: int64Ptr(6), GroupCreatedAt: &now, CreatedAt: &cAt, UpdatedAt: &uAt}
	got3 := toGroupGroupInfo(g3, nil)
	if got3.GroupCreatedAt != now.Unix() || got3.CreatedAt != cAt.Unix() || got3.UpdatedAt != uAt.Unix() {
		t.Fatalf("timestamps mismatch: groupCreated=%d created=%d updated=%d",
			got3.GroupCreatedAt, got3.CreatedAt, got3.UpdatedAt)
	}
}

// TestToGroupInfo 登录响应企业聚合转换 (convert.go, 含 roles 与时间戳)
func TestToGroupInfo(t *testing.T) {
	// nil 链
	if got := toGroupInfo(nil); got != nil {
		t.Fatalf("toGroupInfo(nil) want nil, got %v", got)
	}
	if got := toGroupInfo(&bizV2.GroupGroup{}); got != nil {
		t.Fatalf("toGroupInfo(Group=nil) want nil, got %v", got)
	}
	if got := toGroupInfo(&bizV2.GroupGroup{Group: &bizV2.Group{}}); got != nil {
		t.Fatalf("toGroupInfo(Id=nil) want nil, got %v", got)
	}

	// 正常: CreatedAt(Group.CreatedAt *time.Time 需非 nil) + GroupCreatedAt + UpdatedAt + Roles
	cAt := time.Now()
	gcAt := time.Now()
	uAt := time.Now()
	gg := &bizV2.GroupGroup{
		Group: &bizV2.Group{
			Id: int64Ptr(2), Name: "G", GroupCode: "C", Status: 1,
			CreatedAt: &cAt, GroupCreatedAt: &gcAt, UpdatedAt: &uAt,
		},
		Roles: []*bizV2.Role{{Id: int64Ptr(9), Name: "R"}},
	}
	got := toGroupInfo(gg)
	if got.GroupId != 2 || got.GroupName != "G" || got.GroupCode != "C" || got.Status != 1 {
		t.Fatalf("group fields mismatch: %+v", got)
	}
	if got.CreatedAt != cAt.Unix() || got.GroupCreatedAt != gcAt.Unix() || got.UpdatedAt != uAt.Unix() {
		t.Fatalf("timestamps mismatch: %+v", got)
	}
	if len(got.Roles) != 1 || got.Roles[0].RoleId != 9 {
		t.Fatalf("roles mismatch: %+v", got.Roles)
	}
	// UpdatedAt nil → 不填 (其它时间仍填)
	gg2 := &bizV2.GroupGroup{
		Group: &bizV2.Group{Id: int64Ptr(3), CreatedAt: &cAt},
	}
	got2 := toGroupInfo(gg2)
	if got2.UpdatedAt != 0 {
		t.Fatalf("nil UpdatedAt want 0, got %d", got2.UpdatedAt)
	}
}

// TestToLoginReply 登录响应装配 (AuthV2Service.toLoginReply, 不依赖 authUC, 可直接构造 service)
func TestToLoginReply(t *testing.T) {
	svc := &AuthV2Service{}

	// User nil → 无 UserInfo; Groups/Permissions 空 → 空(非 nil) 切片
	r := &bizV2.LoginResult{Token: "tk", ExpireTime: 100, IsSuperAdmin: true}
	got := svc.toLoginReply(r)
	if got.GetData().GetToken() != "tk" || got.GetData().GetExpireTime() != 100 || !got.GetData().GetIsSuperAdmin() {
		t.Fatalf("login base fields mismatch: %+v", got.GetData())
	}
	if got.GetData().GetUserInfo() != nil {
		t.Fatalf("nil user want nil UserInfo, got %+v", got.GetData().GetUserInfo())
	}
	if got.GetData().GetGroups() == nil || got.GetData().GetPermissions() == nil {
		t.Fatalf("empty groups/permissions want non-nil empty slice")
	}
	if len(got.GetData().GetGroups()) != 0 || len(got.GetData().GetPermissions()) != 0 {
		t.Fatalf("empty groups/permissions want len 0")
	}

	// User.Id=nil → 仍无 UserInfo
	got2 := svc.toLoginReply(&bizV2.LoginResult{User: &bizV2.User{}})
	if got2.GetData().GetUserInfo() != nil {
		t.Fatalf("user with nil Id want nil UserInfo, got %+v", got2.GetData().GetUserInfo())
	}

	// User 有 Id + Groups + Permissions 全填
	uid := int64(42)
	got3 := svc.toLoginReply(&bizV2.LoginResult{
		User:        &bizV2.User{Id: &uid, Username: "u", Nickname: "n"},
		Groups:      []*bizV2.GroupGroup{{Group: &bizV2.Group{Id: int64Ptr(1), CreatedAt: &time.Time{}}}},
		Permissions: []*bizV2.Permission{{Id: 7, Name: "p"}},
	})
	if got3.GetData().GetUserInfo() == nil || got3.GetData().GetUserInfo().GetUserId() != 42 || got3.GetData().GetUserInfo().GetUsername() != "u" {
		t.Fatalf("UserInfo mismatch: %+v", got3.GetData().GetUserInfo())
	}
	if len(got3.GetData().GetGroups()) != 1 || got3.GetData().GetGroups()[0].GetGroupId() != 1 {
		t.Fatalf("Groups mismatch: %+v", got3.GetData().GetGroups())
	}
	if len(got3.GetData().GetPermissions()) != 1 || got3.GetData().GetPermissions()[0].GetId() != 7 {
		t.Fatalf("Permissions mismatch: %+v", got3.GetData().GetPermissions())
	}
}
