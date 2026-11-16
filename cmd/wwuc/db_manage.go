package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/log"

	"wuwen.com/wwuc/cmd/wwuc/dbupdate"
	"wuwen.com/wwuc/internal/conf"
	"wuwen.com/wwuc/internal/data/common"
)

func updateDB(conf *conf.Bootstrap, logger log.Logger, methodStr string) {
    db := common.NewDB(conf, logger)
    rdb := common.NewRedis(conf, logger)
    dataData, cleanup, err := common.NewData(db, rdb, logger)
    if err != nil {
        log.Errorf("new data err:%v", err)
    }
    defer cleanup()
    ctx, cancelFunc := context.WithCancel(context.Background())
    defer cancelFunc()

    methods := strings.Split(methodStr, ",")
    fmt.Println("methods", methods)
    for _, method := range methods {
        if method == "createAdmin" {
            if !dbupdate.CreateAdminInstance().SetParams(flagParams) {
                log.Errorf("set createAdmin params failed")
                return
            }
        }
        if dbManager, exist := dbupdate.MethodMap[method]; exist {
            result := dbManager.Do(ctx, dataData)
            log.Infof("method: %s exec end,result %t", method, result)
        } else {
            log.Warnf("method:%s not exist", method)
        }
    }
    log.Info("update db process end")
}
