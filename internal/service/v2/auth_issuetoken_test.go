package v2

import (
	"testing"
	"time"

	bizV2 "wuwen.com/wwuc/internal/biz/v2"
)

func nowPtr() *time.Time {
	t := time.Now()
	return &t
}

// TestToIssueGroupTokenReply_普通员工 换发响应装配: 返回当前选中企业 + 权限
func TestToIssueGroupTokenReply_普通员工(t *testing.T) {
	s := &AuthV2Service{}
	uid := int64(1001)
	gid := int64(8001)
	roleName := "研发"
	result := &bizV2.LoginResult{
		Token:      "group-token-x",
		ExpireTime: 1735689600,
		User: &bizV2.User{
			Id:       &uid,
			Username: "alice",
			Nickname: "Alice",
		},
		Groups: []*bizV2.GroupGroup{
			{
				Group: &bizV2.Group{Id: &gid, Name: "租户A", GroupCode: "GA", CreatedAt: nowPtr()},
				Roles: []*bizV2.Role{{Id: &gid, Name: roleName}},
			},
		},
		Permissions: []*bizV2.Permission{
			{Id: 1, Name: "查看场景", Code: 100},
		},
		IsSuperAdmin: false,
	}

	reply := s.toIssueGroupTokenReply(result)
	if reply.Code != 200 {
		t.Errorf("code = %d, want 200", reply.Code)
	}
	if reply.Data == nil {
		t.Fatal("data is nil")
	}
	if reply.Data.Token != "group-token-x" {
		t.Errorf("token = %s", reply.Data.Token)
	}
	if reply.Data.ExpireTime != 1735689600 {
		t.Errorf("expireTime = %d", reply.Data.ExpireTime)
	}
	if reply.Data.UserInfo == nil || reply.Data.UserInfo.UserId != 1001 {
		t.Error("userInfo missing or wrong")
	}
	if reply.Data.Group == nil || reply.Data.Group.GroupId != 8001 {
		t.Error("group missing or wrong")
	}
	if len(reply.Data.Group.Roles) != 1 || reply.Data.Group.Roles[0].RoleName != roleName {
		t.Error("group roles wrong")
	}
	if len(reply.Data.Permissions) != 1 || reply.Data.Permissions[0].Name != "查看场景" {
		t.Error("permissions wrong")
	}
	if reply.Data.IsSuperAdmin {
		t.Error("IsSuperAdmin should be false")
	}
}

// TestToIssueGroupTokenReply_超管 超管分支: groupId 为空, group 为 nil, permissions 为空切片
func TestToIssueGroupTokenReply_超管(t *testing.T) {
	s := &AuthV2Service{}
	uid := int64(1)
	result := &bizV2.LoginResult{
		Token:       "global-token",
		ExpireTime:  1735689600,
		User:        &bizV2.User{Id: &uid, Username: "admin"},
		Groups:      []*bizV2.GroupGroup{},
		Permissions: []*bizV2.Permission{},
		IsSuperAdmin: true,
	}

	reply := s.toIssueGroupTokenReply(result)
	if !reply.Data.IsSuperAdmin {
		t.Error("IsSuperAdmin should be true")
	}
	if reply.Data.Group != nil {
		t.Error("super admin group should be nil")
	}
	if reply.Data.Permissions == nil || len(reply.Data.Permissions) != 0 {
		t.Error("permissions should be empty slice, not nil")
	}
}
