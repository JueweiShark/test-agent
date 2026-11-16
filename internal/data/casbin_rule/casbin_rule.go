package casbin_rule

import (
    "github.com/casbin/casbin/v2"
    "github.com/casbin/casbin/v2/model"
    "github.com/go-kratos/kratos/v2/log"
    "gorm.io/gorm"

    gormadapter "github.com/casbin/gorm-adapter/v3"
)

var modelConfig = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && (keyMatch2(r.obj, p.obj) || keyMatch(r.obj, p.obj)) && (r.act == p.act || p.act == "*")
`

//m = g(r.sub, r.dom, p.sub) && r.obj == p.obj && r.act == p.act

func InitCasbin(db *gorm.DB, lgr log.Logger) (enforcer *casbin.SyncedEnforcer, err error) {
    var a *gormadapter.Adapter
    a, err = gormadapter.NewAdapterByDBUseTableName(db, "", "casbin_rule")
    if err != nil {
        return
    }

    // 创建模型
    casbinModel := model.NewModel()
    err = casbinModel.LoadModelFromText(modelConfig)
    if err != nil {
        log.NewHelper(lgr).Info("casbin model load fail", err)
        return
    }
    enforcer, err = casbin.NewSyncedEnforcer(casbinModel, a)
    if err != nil {
        log.NewHelper(lgr).Info("casbin synced Enforcer fail", err)
        return
    }

    // 加载策略
    err = enforcer.LoadPolicy()
    if err != nil {
        //global.Log.Error(ctx, "加载策略失败：%v", err)
        log.NewHelper(lgr).Info("casbin load policy fail", err)
        return
    }
    log.NewHelper(lgr).Info("casbin init success")
    return
}
