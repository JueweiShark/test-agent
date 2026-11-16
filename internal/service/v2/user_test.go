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

// testUserId 由 TestCreateUserV2 创建后回填, 供同文件后续 get/reset/list/forbidden 测试复用.
var testUserId int64

// TestCreateUserV2 超管创建系统用户
func TestCreateUserV2(t *testing.T) {
	ins := initTestV2Env().userSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	req := &pb.CreateUserV2Request{
		Username: fmt.Sprintf("v2user_%d@test.com", time.Now().UnixNano()),
		Nickname: "v2单测用户",
	}
	res, err := ins.CreateUser(ctx, req)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if res.GetData().GetUserInfo() == nil || res.GetData().GetUserInfo().GetUserId() == 0 {
		t.Fatalf("CreateUser reply missing user: %+v", res)
	}
	if res.GetData().GetPassword() == "" {
		t.Fatalf("CreateUser reply missing generated password")
	}
	testUserId = res.GetData().GetUserInfo().GetUserId()
	log.Infof("CreateUser ok, userId=%d username=%s plainPwd=%s", testUserId, req.GetUsername(), res.GetData().GetPassword())
}

// TestGetUserV2 查用户详情 (顺带校验 CreateUser 的 status 默认值: status=0 → 1 启用)
func TestGetUserV2(t *testing.T) {
	if testUserId == 0 {
		t.Skip("依赖 TestCreateUserV2 先创建用户")
	}
	ins := initTestV2Env().userSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	res, err := ins.GetUser(ctx, &pb.GetUserV2Request{UserId: testUserId})
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	// TestCreateUserV2 未传 status (默认 0), service 层会兜底为 1 (启用)
	if res.GetData().GetUser().GetStatus() != 1 {
		t.Fatalf("CreateUser status default want 1(enabled), got %d", res.GetData().GetUser().GetStatus())
	}
	log.Infof("GetUser ok, user=%+v", res.GetData().GetUser())
}

// TestResetPasswordV2 超管重置用户密码
func TestResetPasswordV2(t *testing.T) {
	if testUserId == 0 {
		t.Skip("依赖 TestCreateUserV2 先创建用户")
	}
	ins := initTestV2Env().userSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	req := &pb.ResetPasswordRequest{
		UserId: testUserId,
	}
	res, err := ins.ResetPassword(ctx, req)
	if err != nil {
		t.Fatalf("ResetPassword failed: %v", err)
	}
	if res.GetData().GetPassword() == "" {
		t.Fatalf("ResetPassword reply missing generated password")
	}
	log.Infof("ResetPassword ok, userId=%d plainPwd=%s", testUserId, res.GetData().GetPassword())
}

// TestUpdatePasswordV2 超管指定新密码修改用户密码 (不回显明文)
func TestUpdatePasswordV2(t *testing.T) {
	if testUserId == 0 {
		t.Skip("依赖 TestCreateUserV2 先创建用户")
	}
	ins := initTestV2Env().userSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	// 合法新密码: 满足 8-32 且含数字/大写/小写/特殊字符
	validPwd := "V2NewPwd@2026"
	res, err := ins.UpdatePassword(ctx, &pb.UpdatePasswordRequest{
		UserId:   testUserId,
		Password: validPwd,
	})
	if err != nil {
		t.Fatalf("UpdatePassword failed: %v", err)
	}
	if res.GetCode() != 200 {
		t.Fatalf("UpdatePassword want code 200, got %d msg=%s", res.GetCode(), res.GetMessage())
	}
	log.Infof("UpdatePassword ok, userId=%d", testUserId)

	// 非法新密码: 缺特殊字符, 应返回密码规则错误 (422)
	badRes, err := ins.UpdatePassword(ctx, &pb.UpdatePasswordRequest{
		UserId:   testUserId,
		Password: "NoSpecial2026",
	})
	if err != nil {
		t.Fatalf("UpdatePassword(bad) unexpected err: %v", err)
	}
	if badRes.GetCode() != 422 {
		t.Fatalf("UpdatePassword(bad) want code 422, got %d msg=%s", badRes.GetCode(), badRes.GetMessage())
	}
	log.Infof("UpdatePassword(bad) rejected as expected, code=%d", badRes.GetCode())
}

// TestListUsersV2 用户名单 (分页)
func TestListUsersV2(t *testing.T) {
	ins := initTestV2Env().userSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	res, err := ins.ListUsers(ctx, &pb.ListUsersV2Request{
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	log.Infof("ListUsers ok, total=%d len=%d, users=%+v", res.GetData().GetTotal(), len(res.GetData().GetList()), res.GetData().GetList())
}

// TestUpdateUserV2 更新用户信息 (昵称/显隐状态/备注)
func TestUpdateUserV2(t *testing.T) {
	if testUserId == 0 {
		t.Skip("依赖 TestCreateUserV2 先创建用户")
	}
	ins := initTestV2Env().userSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	req := &pb.UpdateUserV2Request{
		UserId:   testUserId,
		Nickname: "v2更新昵称",
		Status:   0, // 0=隐藏
		Remarks:  "v2单测备注",
	}
	if _, err := ins.UpdateUser(ctx, req); err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}

	// 校验更新结果
	getRes, err := ins.GetUser(ctx, &pb.GetUserV2Request{UserId: testUserId})
	if err != nil {
		t.Fatalf("GetUser after update failed: %v", err)
	}
	if getRes.GetData().GetUser().GetNickname() != req.GetNickname() {
		t.Fatalf("nickname want %s, got %s", req.GetNickname(), getRes.GetData().GetUser().GetNickname())
	}
	if getRes.GetData().GetUser().GetStatus() != req.GetStatus() {
		t.Fatalf("status want %d, got %d", req.GetStatus(), getRes.GetData().GetUser().GetStatus())
	}
	if getRes.GetData().GetUser().GetRemarks() != req.GetRemarks() {
		t.Fatalf("remarks want %s, got %s", req.GetRemarks(), getRes.GetData().GetUser().GetRemarks())
	}
	// 恢复 status=1 (显示), 不影响后续测试
	if _, err := ins.UpdateUser(ctx, &pb.UpdateUserV2Request{
		UserId:   testUserId,
		Nickname: req.GetNickname(),
		Status:   1,
		Remarks:  req.GetRemarks(),
	}); err != nil {
		t.Fatalf("UpdateUser(restore) failed: %v", err)
	}
	log.Infof("UpdateUser ok, userId=%d", testUserId)
}
