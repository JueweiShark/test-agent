package v2

import (
	"os"
	"sync"

	"github.com/go-kratos/kratos/v2/log"

	bizUser "wuwen.com/wwuc/internal/biz/user"
	bizV2 "wuwen.com/wwuc/internal/biz/v2"
	"wuwen.com/wwuc/internal/data/common"
	dataUser "wuwen.com/wwuc/internal/data/user"
	dataV2 "wuwen.com/wwuc/internal/data/v2"
	"wuwen.com/wwuc/internal/pkg/oss"
	"wuwen.com/wwuc/pkg/bootstrap"
)

// testV2Env 持有全部 v2 service 实例, 供各 _test.go 文件复用.
// 参考 internal/service/user/group_test.go 的 initTestGroupConsumer 写法,
// 这里把共享的 DB/Redis/事务/仓储/用例装配收敛到一处, 避免每个测试文件重复初始化.
type testV2Env struct {
	authSvc   *AuthV2Service
	groupSvc  *GroupV2Service
	roleSvc   *RoleV2Service
	menuSvc   *MenuV2Service
	userSvc   *UserV2Service
	systemSvc *SystemV2Service
}

var (
	testV2Once sync.Once
	testV2Ins  *testV2Env
)

// initTestV2Env 初始化测试环境 (加载配置、建 DB/Redis、装配 v2 全部分层依赖).
// 配置路径相对当前包: ../../../conf/wwuc.yaml
func initTestV2Env() *testV2Env {
	testV2Once.Do(func() {
		bootstrap, err := bootstrap.LoadConfig("../../../conf/wwuc.yaml")
		if err != nil {
			log.Errorf("load config err:%+v", err)
		}

		logger := log.NewStdLogger(os.Stdout)
		db := common.NewDB(bootstrap, logger)
		client := common.NewRedis(bootstrap, logger)
		data, _, err := common.NewData(db, client, logger)
		if err != nil {
			log.Errorf("common.NewData err:%+v", err)
		}
		transaction := common.NewTransaction(data)

		// v1 Token (v2 AuthUsecase 依赖 v1 的 TokenUsecase 签发 token)
		tokenRepo := dataUser.NewTokenRepo(data, logger)
		tokenUsecase := bizUser.NewTokenUsecase(logger, bootstrap, tokenRepo)

		// v2 仓储层
		userRepo := dataV2.NewUserRepo(data, logger)
		groupRepo := dataV2.NewGroupRepo(data, logger)
		roleRepo := dataV2.NewRoleRepo(data, logger)
		refUserRoleRepo := dataV2.NewRefUserRoleRepo(data, logger)
		refRolePermissionRepo := dataV2.NewRefRolePermissionRepo(data, logger)
		permissionRepo := dataV2.NewPermissionRepo(data, logger)
		refGroupPermissionRepo := dataV2.NewRefGroupPermissionRepo(data, logger)
		refUserGroupRepo := dataV2.NewRefUserGroupRepo(data, logger)
		appRepo := dataV2.NewAppRepo(data, logger)
		systemConfigRepo := dataV2.NewSystemConfigRepo(data, logger)

		// v2 业务层
		ossUC := oss.NewOssUsecase(bootstrap, logger)
		userUC := bizV2.NewUserUsecase(logger, transaction, userRepo, bootstrap.Password)
		authUC := bizV2.NewAuthUsecase(logger, userRepo, groupRepo, roleRepo, refUserRoleRepo,
			permissionRepo, appRepo, tokenUsecase, userUC, ossUC)
		groupUC := bizV2.NewGroupUsecase(logger, transaction, groupRepo, refGroupPermissionRepo, permissionRepo, appRepo)
		roleUC := bizV2.NewRoleUsecase(logger, transaction, roleRepo, refUserRoleRepo,
			refRolePermissionRepo, permissionRepo, userRepo, groupRepo, refGroupPermissionRepo, refUserGroupRepo)
		menuUC := bizV2.NewMenuUsecase(logger, permissionRepo)
		systemUC := bizV2.NewSystemUsecase(logger, transaction, systemConfigRepo, appRepo, ossUC)

		// v2 服务层
		testV2Ins = &testV2Env{
			authSvc:   NewAuthV2Service(authUC, logger),
			groupSvc:  NewGroupV2Service(groupUC, logger),
			roleSvc:   NewRoleV2Service(roleUC, logger),
			menuSvc:   NewMenuV2Service(menuUC, logger),
			userSvc:   NewUserV2Service(userUC, logger),
			systemSvc: NewSystemV2Service(systemUC, authUC, logger),
		}
	})
	return testV2Ins
}

// 可配置测试参数: 以下 ID/账号需根据本地数据库实际情况调整.
// create 类测试用时间戳保证可重复运行; get/update/delete 类测试在文件内由 create 产物填充.
var (
	testOperatorId int64 = 1 // 超管操作人 (需为真实存在的超管用户)
)
