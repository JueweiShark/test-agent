package v2

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/golang/protobuf/proto"

	pb "wuwen.com/wwuc/api_gen/user/v2"

	biz_common "wuwen.com/wwuc/internal/biz/common"
)

// testMenuApp 菜单所属平台 (可按本地数据库调整)
var testMenuApp int64 = 1

// testMenuId 由 TestCreateMenuV2 创建后回填, 供同文件后续 update/list/tree/delete 测试复用.
var testMenuId int64

// TestCreateMenuV2 建菜单节点
func TestCreateMenuV2(t *testing.T) {
	ins := initTestV2Env().menuSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	req := &pb.CreateMenuRequest{
		Name:              uniqGroupName("v2菜单"),
		Type:              1, // 1=前端
		MenuType:          2, // 2=菜单
		App:               testMenuApp,
		RoutePath:         "/v2/test/menu",
		Icon:              "test-icon",
		Sort:              100,
		Path:              "/api/v2/test",
		Act:               "GET",
		DependenceParents: 0, // 顶级节点
		Domain:            "wwuc",
	}
	res, err := ins.CreateMenu(ctx, req)
	if err != nil {
		t.Fatalf("CreateMenu failed: %v", err)
	}
	if res.GetData().GetMenu() == nil || res.GetData().GetMenu().GetId() == 0 {
		t.Fatalf("CreateMenu reply missing menu: %+v", res)
	}
	testMenuId = res.GetData().GetMenu().GetId()
	log.Infof("CreateMenu ok, menuId=%d name=%s app=%d", testMenuId, req.GetName(), req.GetApp())
}

// TestUpdateMenuV2 编辑菜单
func TestUpdateMenuV2(t *testing.T) {
	if testMenuId == 0 {
		t.Skip("依赖 TestCreateMenuV2 先创建菜单")
	}
	ins := initTestV2Env().menuSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	req := &pb.UpdateMenuRequest{
		Id:          testMenuId,
		Name:        uniqGroupName("v2菜单-改"),
		Sort:        200,
		Hidden:      proto.Int32(0),                       // 改为隐藏
		Description: proto.String("备注-改"),              // 改为备注
	}
	if _, err := ins.UpdateMenu(ctx, req); err != nil {
		t.Fatalf("UpdateMenu failed: %v", err)
	}
	log.Infof("UpdateMenu ok, menuId=%d", testMenuId)
}

// TestListMenusV2 按 app/父节点/类型查菜单列表
func TestListMenusV2(t *testing.T) {
	ins := initTestV2Env().menuSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	res, err := ins.ListMenus(ctx, &pb.ListMenusRequest{
		App:      testMenuApp,
		MenuType: 0, // 0=不限类型
	})
	if err != nil {
		t.Fatalf("ListMenus failed: %v", err)
	}
	log.Infof("ListMenus ok, app=%d len=%d, menus=%+v", testMenuApp, len(res.GetData().GetMenus()), res.GetData().GetMenus())
}

// TestGetMenuTreeV2 按 app 拿完整菜单树
func TestGetMenuTreeV2(t *testing.T) {
	ins := initTestV2Env().menuSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	res, err := ins.GetMenuTree(ctx, &pb.GetMenuTreeRequest{App: testMenuApp})
	if err != nil {
		t.Fatalf("GetMenuTree failed: %v", err)
	}
	// 输出完整菜单树 (proto text 格式, 直观查看层级结构)
	log.Infof("GetMenuTree ok, app=%d roots=%d, tree=%+v", testMenuApp, len(res.GetData().GetTree()), res.GetData().GetTree())
}

// TestDeleteMenuV2 删除菜单 (有子节点则拒绝; 放在文件末尾清理)
func TestDeleteMenuV2(t *testing.T) {
	if testMenuId == 0 {
		t.Skip("依赖 TestCreateMenuV2 先创建菜单")
	}
	ins := initTestV2Env().menuSvc
	ctx := context.Background()
	ctx = biz_common.NewReqCtx(ctx, testOperatorId, "")

	if _, err := ins.DeleteMenu(ctx, &pb.DeleteMenuRequest{Id: testMenuId}); err != nil {
		t.Fatalf("DeleteMenu failed: %v", err)
	}
	log.Infof("DeleteMenu ok, menuId=%d", testMenuId)
	testMenuId = 0
}
