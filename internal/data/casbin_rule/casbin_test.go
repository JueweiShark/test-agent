package casbin_rule

import (
    "fmt"
    "testing"

    "github.com/go-kratos/kratos/v2/log"

    "wuwen.com/wwuc/internal/conf"
    "wuwen.com/wwuc/internal/data/common"
)

func TestLoadCasbin(t *testing.T) {
    db := common.NewDB(&conf.Bootstrap{
        Data: &conf.Data{
            Database: &conf.Data_Database{
                Driver:         "mysql",
                Source:         "root:ImOxO8Lz@tcp(192.168.2.49:8191)/wwuc?charset=utf8mb4&parseTime=True&loc=Local",
                MaxConnThreads: 100,
                MaxLifeMinute:  30,
            },
            Redis: nil,
        },
    }, log.DefaultLogger)
    db = db.Debug()
    enforcer, err := InitCasbin(db, log.DefaultLogger)
    if err != nil {
        log.Error("init casbin", err)
        return
    }
    enforcer.EnableLog(true)
    //enforcer.ClearPolicy()

    // Step 1: 定义权限规则 (Permission -> Resource/Action)
    _, _ = enforcer.AddPolicy("permission:1", "/api/resource/*", "write")
    _, _ = enforcer.AddPolicy("permission:2", "/api/resource/*", "read")
    _, _ = enforcer.AddPolicy("permission:3", "/user/*", "GET")

    // 添加角色与权限的映射关系
    _, _ = enforcer.AddGroupingPolicy("role:1", "permission:1")
    _, _ = enforcer.AddGroupingPolicy("role:2", "permission:2")
    _, _ = enforcer.AddGroupingPolicy("role:3", "permission:3")

    // 添加用户与角色的映射关系
    _, _ = enforcer.AddGroupingPolicy("user:1", "role:1")
    _, _ = enforcer.AddGroupingPolicy("user:2", "role:2")
    _, _ = enforcer.AddGroupingPolicy("user:2", "role:3")

    // 保存策略到数据库
    //err = enforcer.SavePolicy()
    //if err != nil {
    //    panic(err)
    //}
    fmt.Println("策略已保存！")
    policies, _ := enforcer.GetPolicy()
    fmt.Printf("Policies: %+v\n", policies)

    groupingPolicies, _ := enforcer.GetGroupingPolicy()
    fmt.Printf("Grouping Policies: %+v\n", groupingPolicies)

    // 获取用户的角色
    roles := enforcer.GetRolesForUserInDomain("user:1", "group1")
    rolesForUser, _ := enforcer.GetImplicitRolesForUser("user:1")
    fmt.Printf("roles: %+v\n", roles)
    fmt.Printf("rolesForUser: %+v\n", rolesForUser)

    rolesDO := enforcer.GetRolesForUserInDomain("user:1", "group1")
    fmt.Printf("Roles for user:1 in group1: %+v\n", rolesDO)

    permissions1, _ := enforcer.GetImplicitPermissionsForUser("user:1")
    fmt.Printf("Permissions for user:1: %+v\n", permissions1)
    // 遍历用户的角色并获取权限
    finalPermissions := []string{}
    for _, role := range roles {
        // 查询角色对应的权限
        //permissions, err := cas.GetNamedGroupingPolicy("g")
        permissions, err := enforcer.GetImplicitPermissionsForUser(role)
        if err != nil {
            panic(err)
        }
        fmt.Printf("permissions for role %s: %+v\n", role, permissions)
        for _, p := range permissions {
            finalPermissions = append(finalPermissions, fmt.Sprintf("%s:%s", p[1], p[2]))
        }
    }
    fmt.Printf("finalPermissions: %+v\n", finalPermissions)

    // Step 4: 验证权限
    ok, err := enforcer.Enforce("user:1", "/api/resource/123", "write") // 检查 user:1 是否能写入 /api/resource/123
    if err != nil {
        panic(err)
    }
    if ok {
        fmt.Println("允许访问")
    } else {
        fmt.Println("禁止访问")
    }
}
