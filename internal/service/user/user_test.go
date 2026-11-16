package service

import (
	"context"

	"testing"

	"os"

	"github.com/go-kratos/kratos/v2/log"
	pb "wuwen.com/wwuc/api_gen/user/v1"

	"fmt"

	biz_common "wuwen.com/wwuc/internal/biz/common"
	user2 "wuwen.com/wwuc/internal/biz/user"
	"wuwen.com/wwuc/internal/data/common"
	"wuwen.com/wwuc/internal/data/user"
	"wuwen.com/wwuc/pkg/bootstrap"
)

func initTestTSConsumer() *UserService {

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
	tokenRepo := user.NewTokenRepo(data, logger)
	tokenUsecase := user2.NewTokenUsecase(logger, bootstrap, tokenRepo)
	refRolePermissionRepo := user.NewRolePermissionRepo(data, logger)
	permissionRepo := user.NewPermissionRepo(data, logger)
	refRolePermissionUsecase := user2.NewRefRolePermissionUsecase(logger, transaction, refRolePermissionRepo, permissionRepo)

	userUsecase := user2.NewUserUsecase(logger, transaction, userRepo, refUserRoleRepo, roleRepo, groupRepo, refUserGroupRepo, tokenUsecase, refRolePermissionUsecase, permissionRepo)

	return NewUserService(userUsecase, tokenUsecase)
}

func TestUserInfo(t *testing.T) {
	ins := initTestTSConsumer()
	ctx := context.Background()

	var request pb.GetUserRequest
	request.Id = 131
	ctx = biz_common.NewReqCtx(ctx, request.Id, "")
	res, err := ins.UserInfo(ctx, &request)
	if err != nil {
		t.Fatal(err)
	}
	log.Infof("UserInfo:%+v", res)
	log.Infof("role:%+v", res.Data.RoleInfo)
}

func TestCreateUser(t *testing.T) {
	ins := initTestTSConsumer()
	ctx := context.Background()
	uid := 131
	ctx = biz_common.NewReqCtx(ctx, int64(uid), "")
	var request pb.CreateUserRequest
	request.Username = "jushenE1@123.com"
	request.Nickname = "jushenE1"

	request.RoleIds = []int64{491}

	res, err := ins.CreateUser(ctx, &request)
	if err != nil {
		t.Fatal(err)
	}
	log.Infof("UserInfo:%+v", res)
}

func TestForbiddenUser(t *testing.T) {
	ins := initTestTSConsumer()
	ctx := context.Background()
	var request pb.ForbiddenUserRequest
	request.Id = 132
	request.Status = 0
	res, err := ins.ForbiddenUser(ctx, &request)
	if err != nil {
		t.Fatal(err)
	}
	log.Infof("UserInfo:%+v", res)
}

func TestLogin(t *testing.T) {
	ins := initTestTSConsumer()
	ctx := context.Background()

	var request pb.LoginRequest
	request.Username = "admin@123.com"
	request.Username = "admin@123.com"
	request.Password = "Wuwen0122!"
	request.AppId = 2

	res, err := ins.Login(ctx, &request)
	if err != nil {
		t.Fatal(err)
	}
	log.Infof("UserInfo:%+v", res)
}

func TestUserList(t *testing.T) {
	ins := initTestTSConsumer()
	ctx := context.Background()
	uid := 223
	ctx = biz_common.NewReqCtx(ctx, int64(uid), "")

	var request pb.ListUserRequest
	request.PageNo = 1
	request.PageSize = 10

	//status := int32(1);
	////nickName :="test"
	//request.Status = &status
	//request.Nickname = &nickName
	res, err := ins.ListUser(ctx, &request)
	if err != nil {
		t.Fatal(err)
	}
	log.Infof("UserInfo:%+v", res)
}

func TestUserGroupInfo(t *testing.T) {
	ins := initTestTSConsumer()
	ctx := context.Background()

	var request pb.GetUserRequest
	request.Id = 122

	res, err := ins.UserGroupInfo(ctx, &request)
	if err != nil {
		t.Fatal(err)
	}
	log.Infof("UserGroupInfo:%+v", res)
}

func TestUpdateuser(t *testing.T) {
	ins := initTestTSConsumer()
	ctx := context.Background()

	var request pb.UpdateUserRequest
	request.Id = 122
	request.Password = "123456"
	request.OldPassword = "123123"
	ctx = biz_common.NewReqCtx(ctx, request.Id, "")
	res, err := ins.UpdateUser(ctx, &request)
	if err != nil {
		t.Fatal(err)
	}
	log.Infof("UserInfo:%+v", res)
}

func TestListPermissionUser(t *testing.T) {
	ins := initTestTSConsumer()
	ctx := context.Background()

	uid := 253
	ctx = biz_common.NewReqCtx(ctx, int64(uid), "")
	var request pb.ListUserRequest
	request.PageNo = 1
	request.PageSize = 100

	res, err := ins.ListPermissionUser(ctx, &request)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range res.Data.List {
		fmt.Printf("=====v:%+v \n", v)
	}
}
