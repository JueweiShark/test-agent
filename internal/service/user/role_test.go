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

func initTestRoleConsumer() *RoleService {

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
	userUsecase := user2.NewUserUsecase(logger, transaction, userRepo, refUserRoleRepo, roleRepo, groupRepo, refUserGroupRepo, tokenUsecase, refRolePermissionUsecase, permissionRepo)
	roleUsecase := user2.NewRoleUsecase(logger, transaction, userRepo, roleRepo, permissionRepo, refUserRoleRepo, refRolePermissionRepo)
	
	return NewRoleService(roleUsecase,userUsecase)
}

func TestCreateRole(t *testing.T) {
	ins := initTestRoleConsumer()
	ctx := context.Background()
	uid := 131
	var request pb.CreateRoleRequest
	request.Name = "具身角色e"
	request.Remarks = "测试角色"
	//request.GroupId	= 1848974093960425630
	per := []int64{2032,2031,2033}
	request.PermissionCodes =per

	ctx = biz_common.NewReqCtx(ctx, int64(uid), "")
	res, err := ins.CreateRole(ctx, &request)
	if err != nil {
		t.Fatal(err)
	}
	log.Infof("UserInfo:%+v", res)
}




func TestListRole(t *testing.T) {
	ins := initTestRoleConsumer()
	ctx := context.Background()
	uid := 131
	var request pb.ListRoleRequest

	request.GroupId	= 1848974093960425630
	request.Name= ""
	ctx = biz_common.NewReqCtx(ctx, int64(uid), "")
	res, err := ins.ListRole(ctx, &request)
	if err != nil {
		t.Fatal(err)
	}
	for _,v:=range res.Data{
	log.Infof("===v:%+v", v.Remarks)
	}
}