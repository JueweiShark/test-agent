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

// testAppId 由 TestCreateAppV2 创建后回填, 供同文件后续 update/list 测试复用.
var testAppId int64

// TestCreateAppV2 创建产品
func TestCreateAppV2(t *testing.T) {
	ins := initTestV2Env().systemSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	req := &pb.CreateAppRequest{
		Name:     uniqGroupName("v2产品"),
		Code:     fmt.Sprintf("VAPP%d", time.Now().UnixNano()),
		EntryUrl: "https://v2-test.example.com",
		Icon:     "test-icon",
		Sort:     50,
		Type:     "platform",
		Desc:     "v2单测产品",
		Detail:   `{"k":"v"}`, // app.detail 为 MySQL JSON 列, 须传合法 JSON
	}
	res, err := ins.CreateApp(ctx, req)
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}
	if res.GetData().GetApp() == nil || res.GetData().GetApp().GetId() == 0 {
		t.Fatalf("CreateApp reply missing app: %+v", res)
	}
	testAppId = res.GetData().GetApp().GetId()
	log.Infof("CreateApp ok, id=%d name=%s", testAppId, req.GetName())
}

// TestUpdateAppV2 更新产品
func TestUpdateAppV2(t *testing.T) {
	if testAppId == 0 {
		t.Skip("依赖 TestCreateAppV2 先创建产品")
	}
	ins := initTestV2Env().systemSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	req := &pb.UpdateAppRequest{
		AppId:    testAppId,
		Name:     uniqGroupName("v2产品-改"),
		Code:     fmt.Sprintf("VAPP%d", testAppId),
		EntryUrl: "https://v2-test-updated.example.com",
		Icon:     "test-icon-2",
		Sort:     60,
		Type:     "link",
		Desc:     "v2单测产品-改",
		Detail:   `{"k":"v2"}`,
		Status:   0, // 0=不改变状态
	}
	if _, err := ins.UpdateApp(ctx, req); err != nil {
		t.Fatalf("UpdateApp failed: %v", err)
	}
	log.Infof("UpdateApp ok, id=%d", testAppId)
}

// TestListAppsV2 拿当前用户在指定企业内有权限的产品列表 (超管返回全部)
func TestListAppsV2(t *testing.T) {
	ins := initTestV2Env().systemSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	// 超管: 传任意存在的 groupId 均返回所有启用产品
	res, err := ins.ListApps(ctx, &pb.ListAppsRequest{GroupId: 1})
	if err != nil {
		t.Fatalf("ListApps failed: %v", err)
	}
	log.Infof("ListApps ok, len=%d, apps=%+v", len(res.GetData().GetApps()), res.GetData().GetApps())

	// 按 type=platform 过滤
	byType, err := ins.ListApps(ctx, &pb.ListAppsRequest{GroupId: 1, Type: "platform"})
	if err != nil {
		t.Fatalf("ListApps(type=platform) failed: %v", err)
	}
	for _, a := range byType.GetData().GetApps() {
		if a.GetType() != "platform" {
			t.Fatalf("ListApps(type=platform) returned non-platform app: %+v", a)
		}
	}
	log.Infof("ListApps(type=platform) ok, len=%d", len(byType.GetData().GetApps()))
}

// TestListAppsByPageV2 分页查询产品列表 (管理端, 含 name/code/status 过滤分支)
func TestListAppsByPageV2(t *testing.T) {
	ins := initTestV2Env().systemSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	// 1. 无过滤分页查询
	res, err := ins.ListAppsByPage(ctx, &pb.ListAppsByPageRequest{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListAppsByPage failed: %v", err)
	}
	log.Infof("ListAppsByPage(all) ok, total=%d len=%d, apps=%+v", res.GetData().GetTotal(), len(res.GetData().GetList()), res.GetData().GetList())

	// 2. 按 name 模糊过滤 (复用 TestCreateAppV2 创建的产品名前缀), 覆盖 cond.Name 分支
	byName, err := ins.ListAppsByPage(ctx, &pb.ListAppsByPageRequest{
		Page: 1, PageSize: 10, Name: "v2产品",
	})
	if err != nil {
		t.Fatalf("ListAppsByPage(name) failed: %v", err)
	}
	log.Infof("ListAppsByPage(name=v2产品) ok, total=%d len=%d, apps=%+v", byName.GetData().GetTotal(), len(byName.GetData().GetList()), byName.GetData().GetList())

	// 3. 按 status=1 过滤, 覆盖 cond.Status 分支
	byStatus, err := ins.ListAppsByPage(ctx, &pb.ListAppsByPageRequest{
		Page: 1, PageSize: 10, Status: 1,
	})
	if err != nil {
		t.Fatalf("ListAppsByPage(status) failed: %v", err)
	}
	log.Infof("ListAppsByPage(status=1) ok, total=%d len=%d, apps=%+v", byStatus.GetData().GetTotal(), len(byStatus.GetData().GetList()), byStatus.GetData().GetList())

	// 4. 按 type=platform 过滤, 覆盖 cond.Type 分支
	byType, err := ins.ListAppsByPage(ctx, &pb.ListAppsByPageRequest{
		Page: 1, PageSize: 10, Type: "platform",
	})
	if err != nil {
		t.Fatalf("ListAppsByPage(type) failed: %v", err)
	}
	for _, a := range byType.GetData().GetList() {
		if a.GetType() != "platform" {
			t.Fatalf("ListAppsByPage(type=platform) returned non-platform app: %+v", a)
		}
	}
	log.Infof("ListAppsByPage(type=platform) ok, total=%d len=%d", byType.GetData().GetTotal(), len(byType.GetData().GetList()))
}

// TestSetSystemConfigV2 批量设置配置 (upsert)
func TestSetSystemConfigV2(t *testing.T) {
	ins := initTestV2Env().systemSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	// 空 configs → 走早返回分支 (不调用 biz, 直接成功)
	if _, err := ins.SetSystemConfig(ctx, &pb.SetSystemConfigRequest{Configs: map[string]string{}}); err != nil {
		t.Fatalf("SetSystemConfig(empty) failed: %v", err)
	}

	key := fmt.Sprintf("v2_test_key_%d", time.Now().UnixNano())
	req := &pb.SetSystemConfigRequest{
		Configs: map[string]string{
			key:             "v2单测值",
			"v2_test_fixed": "fixed-value",
		},
	}
	if _, err := ins.SetSystemConfig(ctx, req); err != nil {
		t.Fatalf("SetSystemConfig failed: %v", err)
	}
	log.Infof("SetSystemConfig ok, keys=%v", req.GetConfigs())
}

// TestGetSystemConfigV2 获取配置 (key 为空则返回全部)
func TestGetSystemConfigV2(t *testing.T) {
	ins := initTestV2Env().systemSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	// 1. 不传 key, 返回全部
	allRes, err := ins.GetSystemConfig(ctx, &pb.GetSystemConfigRequest{})
	if err != nil {
		t.Fatalf("GetSystemConfig(all) failed: %v", err)
	}
	log.Infof("GetSystemConfig(all) ok, len=%d, configs=%+v", len(allRes.GetData().GetConfigs()), allRes.GetData().GetConfigs())

	// 2. 指定 key (依赖前一个 SetSystemConfig 写入的固定 key)
	keyRes, err := ins.GetSystemConfig(ctx, &pb.GetSystemConfigRequest{ConfigKey: "v2_test_fixed"})
	if err != nil {
		t.Fatalf("GetSystemConfig(key) failed: %v", err)
	}
	log.Infof("GetSystemConfig(key) ok, configs=%+v", keyRes.GetData().GetConfigs())
}
