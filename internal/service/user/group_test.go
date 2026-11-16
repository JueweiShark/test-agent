package service

import (
	"context"

	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"os"
	pb "wuwen.com/wwuc/api_gen/user/v1"

	biz_common "wuwen.com/wwuc/internal/biz/common"
	user2 "wuwen.com/wwuc/internal/biz/user"
	"wuwen.com/wwuc/internal/data/common"
	"wuwen.com/wwuc/internal/data/user"
	"wuwen.com/wwuc/pkg/bootstrap"
)

func initTestGroupConsumer() *GroupService {

	bootstrap, _ := bootstrap.LoadConfig("../../../conf/wwuc.yaml")

	logger := log.NewStdLogger(os.Stdout)
	db := common.NewDB(bootstrap, logger)
	client := common.NewRedis(bootstrap, logger)
	data, _, err := common.NewData(db, client, logger)
	if err != nil {
		log.Errorf("common.NewData err:%+v", data)
	}
	transaction := common.NewTransaction(data)
	userRepo := user.NewUserRepo(data, logger)
	refUserRoleRepo := user.NewRefUserRoleRepo(data, logger)
	roleRepo := user.NewRoleRepo(data, logger)
	groupRepo := user.NewGroupRepo(data, logger)
	refUserGroupRepo := user.NewRefUserGroupRepo(data, logger)
	refRolePermissionRepo := user.NewRolePermissionRepo(data, logger)
	permissionRepo := user.NewPermissionRepo(data, logger)
	refRolePermissionUsecase := user2.NewRefRolePermissionUsecase(logger, transaction, refRolePermissionRepo, permissionRepo)
	
	
	tokenRepo := user.NewTokenRepo(data, logger)
	tokenUsecase := user2.NewTokenUsecase(logger, bootstrap, tokenRepo)
	groupUsecase := user2.NewGroupUsecase(logger, transaction, userRepo, groupRepo, roleRepo, refUserGroupRepo, refUserRoleRepo, refRolePermissionRepo, permissionRepo)
	userUsecase := user2.NewUserUsecase(logger, transaction, userRepo, refUserRoleRepo, roleRepo, groupRepo, refUserGroupRepo, tokenUsecase, refRolePermissionUsecase, permissionRepo)

	return NewGroupService(groupUsecase,userUsecase)
}

func TestCreateGroup(t *testing.T) {
	ins := initTestGroupConsumer()
	ctx := context.Background()
	uid := 131
	var request pb.CreateGroupRequest
	request.Name = "hutest4分组"
	request.Remarks = "aaaaaa"

	ctx = biz_common.NewReqCtx(ctx, int64(uid), "")
	res, err := ins.CreateGroup(ctx, &request)
	if err != nil {
		log.Errorf("err:%+v", res)
		t.Fatal(err)
	}
	log.Infof("res:%+v", res)
}


func TestUpdateGroup(t *testing.T) {
	ins := initTestGroupConsumer()
	ctx := context.Background()
	uid := 131
	var request pb.UpdateGroupRequest
	request.Id = 1848974093960425633
	request.Name = "hutest3分组 测试"
	
	ctx = biz_common.NewReqCtx(ctx, int64(uid), "")
	res, err := ins.UpdateGroup(ctx, &request)
	if err != nil {
		t.Fatal(err)
	}
	log.Infof("UserInfo:%+v", res)
}


func TestGetGroup(t *testing.T) {
	ins := initTestGroupConsumer()
	ctx := context.Background()
	uid := 122
	var request pb.GetGroupRequest
	request.Id = 1848974093960425635
	
	ctx = biz_common.NewReqCtx(ctx, int64(uid), "")
	res, err := ins.GetGroup(ctx, &request)
	if err != nil {
		t.Fatal(err)
	}
	log.Infof("UserInfo:%+v", res)
}


func TestListGroup(t *testing.T) {
	ins := initTestGroupConsumer()
	ctx := context.Background()
	uid := 125
	var request pb.ListGroupRequest
	name := "hutest4"
	request.Name = &name
	
	ctx = biz_common.NewReqCtx(ctx, int64(uid), "")
	res, err := ins.ListGroup(ctx, &request)
	if err != nil {
		t.Fatal(err)
	}
	for _,v:=range res.Data.List {
		log.Infof("===group Remarks:%+v", v.Remarks)
	}
}

func TestMemberList(t *testing.T) {
	ins := initTestGroupConsumer()
	ctx := context.Background()
	uid := 125
	var request pb.MemberListRequest
	request.GroupId = 1848974093960425645
	request.PageNo = 1
	request.PageSize = 20

	
	ctx = biz_common.NewReqCtx(ctx, int64(uid), "")
	res, err := ins.MemberList(ctx, &request)
	if err != nil {
		t.Fatal(err)
	}
	for _,v:=range res.Data.List {
		log.Infof("===v:%+v", v.RoleInfo)
	}
}
