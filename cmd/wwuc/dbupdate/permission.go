package dbupdate

import (
    "context"
    "encoding/json"
    "fmt"

    "gorm.io/gorm/clause"

    "wuwen.com/wwuc/internal/data/common"
    "wuwen.com/wwuc/internal/data/schema"
)

var (
    loadPermissions = NewPermission(dbUpdateTypeLoadData)
)

// Permission 初始化 wdc_dataset
type Permission struct {
    t dbUpdateType
}

func NewPermission(t dbUpdateType) *Permission {
    return &Permission{
        t: t,
    }
}

func (m *Permission) MethodName() string {
    return "load_permission"
}
func (m *Permission) Do(ctx context.Context, data *common.Data) bool {
    switch m.t {
    case dbUpdateTypeInit:
        return false
    case dbUpdateTypeUpdate:
        return false
    case dbUpdateTypeLoadData:
        return LoadPermission(ctx, data)
    default:
        return false
    }
}

var permissionJson = `{
    "id": 1,
    "name": "功能权限",
    "children":
    [
        {
            "id": 11,
            "parentId": 1,
            "name": "资源管理",
            "route": "resource",
            "path": "be的接口路径",
            "operateMethod": "GET",
            "children":
            [
                {
                    "id": 110,
                    "parentId": 11,
                    "name": "主车库",
                    "route": "resource/mainGarage",
                    "path": "/materials/ego",
                    "operateMethod": "GET",
                    "children":
                    [
                        {
                            "id": 1101,
                            "parentId": 110,
                            "name": "新建"
                        },
                        {
                            "id": 1102,
                            "parentId": 110,
                            "name": "编辑"
                        },
                        {
                            "id": 1103,
                            "parentId": 110,
                            "name": "删除"
                        },
                        {
                            "id": 1104,
                            "parentId": 110,
                            "name": "导出"
                        }
                    ]
                },
                {
                    "id": 111,
                    "parentId": 11,
                    "name": "元素库",
                    "route": "resource/elementLibrary",
                    "path": "materials/element,materials/element",
                    "operateMethod": "GET",
                    "children":
                    [
                        {
                            "id": 1111,
                            "parentId": 111,
                            "name": "新建"
                        },
                        {
                            "id": 1112,
                            "parentId": 111,
                            "name": "编辑"
                        },
                        {
                            "id": 1113,
                            "parentId": 111,
                            "name": "删除"
                        },
                        {
                            "id": 1114,
                            "parentId": 111,
                            "name": "导出"
                        }
                    ]
                },
                {
                    "id": 112,
                    "parentId": 11,
                    "name": "地图库",
                    "route": "resource/mapLibrary",
                    "path": "materials/map",
                    "operateMethod": "GET",
                    "children":
                    [
                        {
                            "id": 1121,
                            "parentId": 112,
                            "name": "新建"
                        },
                        {
                            "id": 1122,
                            "parentId": 112,
                            "name": "编辑"
                        },
                        {
                            "id": 1123,
                            "parentId": 112,
                            "name": "删除"
                        },
                        {
                            "id": 1124,
                            "parentId": 112,
                            "name": "导出"
                        }
                    ]
                },
                {
                    "id": 113,
                    "parentId": 11,
                    "name": "被测系统库",
                    "route": "resource/testedSystemLibrary",
                    "path": "materials/testSystem",
                    "operateMethod": "GET",
                    "children":
                    []
                },
                {
                    "id": 114,
                    "parentId": 11,
                    "name": "bridge库",
                    "route": "resource/bridgeLibrary",
                    "path": "materials/bridge",
                    "operateMethod": "GET",
                    "children":
                    [
                        {
                            "id": 1141,
                            "parentId": 114,
                            "name": "新建"
                        },
                        {
                            "id": 1142,
                            "parentId": 114,
                            "name": "编辑"
                        },
                        {
                            "id": 1143,
                            "parentId": 114,
                            "name": "删除"
                        },
                        {
                            "id": 1144,
                            "parentId": 114,
                            "name": "导出"
                        }
                    ]
                }
            ]
        },
        {
            "id": 12,
            "parentId": 1,
            "name": "任务管理",
            "route": "task",
            "path": "be的接口路径",
            "operateMethod": "GET",
            "children":
            [
                {
                    "id": 120,
                    "parentId": 12,
                    "name": "任务列表",
                    "route": "task/taskList",
                    "path": "task",
                    "operateMethod": "GET",
                    "children":
                    [
                        {
                            "id": 1201,
                            "parentId": 120,
                            "name": "详情"
                        },
                        {
                            "id": 1202,
                            "parentId": 120,
                            "name": "报告"
                        },
                        {
                            "id": 1203,
                            "parentId": 120,
                            "name": "删除"
                        }
                    ]
                }
            ]
        },
        {
            "id": 13,
            "parentId": 1,
            "name": "用例管理",
            "route": "usecasemanagement",
            "path": "be的接口路径",
            "operateMethod": "GET",
            "children":
            [
                {
                    "id": 130,
                    "parentId": 13,
                    "name": "worldsim用例库",
                    "route": "usecasemanagement/worldsimCase",
                    "path": "usecase",
                    "operateMethod": "GET",
                    "children":
                    [
                        {
                            "id": 1301,
                            "parentId": 130,
                            "name": "新建"
                        },
                        {
                            "id": 1302,
                            "parentId": 130,
                            "name": "编辑"
                        },
                        {
                            "id": 1303,
                            "parentId": 130,
                            "name": "删除"
                        },
                        {
                            "id": 1304,
                            "parentId": 130,
                            "name": "导出"
                        },
                        {
                            "id": 1305,
                            "parentId": 130,
                            "name": "上传"
                        },
                        {
                            "id": 1306,
                            "parentId": 130,
                            "name": "移动"
                        }
                    ]
                },
                {
                    "id": 131,
                    "parentId": 13,
                    "name": "指标管理",
                    "route": "usecasemanagement/IndicatorManagement",
                    "path": "metric",
                    "operateMethod": "GET",
                    "children":
                    [
                        {
                            "id": 1311,
                            "parentId": 131,
                            "name": "新建"
                        },
                        {
                            "id": 1312,
                            "parentId": 131,
                            "name": "编辑"
                        },
                        {
                            "id": 1313,
                            "parentId": 131,
                            "name": "删除"
                        }
                    ]
                },
                {
                    "id": 132,
                    "parentId": 13,
                    "name": "场景标签管理",
                    "route": "usecasemanagement/sceneTagManagement",
                    "path": "sceneTag",
                    "operateMethod": "GET",
                    "children":
                    []
                },
                {
                    "id": 133,
                    "parentId": 13,
                    "name": "场景统计看板",
                    "route": "usecasemanagement/sceneStatisticsDashboard",
                    "path": "setStatistics",
                    "operateMethod": "GET",
                    "children":
                    []
                }
            ]
        },
        {
            "id": 14,
            "parentId": 1,
            "name": "系统管理",
            "route": "system",
            "path": "be的接口路径",
            "operateMethod": "GET",
            "children":
            [
                {
                    "id": 140,
                    "parentId": 14,
                    "name": "用户管理",
                    "route": "system/user",
                    "path": "user",
                    "operateMethod": "GET",
                    "children":
                    [
                        {
                            "id": 1401,
                            "parentId": 140,
                            "name": "新建"
                        },
                        {
                            "id": 1402,
                            "parentId": 140,
                            "name": "编辑"
                        },
                        {
                            "id": 1403,
                            "parentId": 140,
                            "name": "删除"
                        }
                    ]
                },
                {
                    "id": 141,
                    "parentId": 14,
                    "name": "组织管理",
                    "route": "system/organize",
                    "path": "group",
                    "operateMethod": "GET",
                    "children":
                    [
                        {
                            "id": 1411,
                            "parentId": 141,
                            "name": "新建"
                        },
                        {
                            "id": 1412,
                            "parentId": 141,
                            "name": "编辑"
                        },
                        {
                            "id": 1413,
                            "parentId": 141,
                            "name": "删除"
                        },
                        {
                            "id": 1413,
                            "parentId": 141,
                            "name": "管理"
                        }
                    ]
                }
            ]
        }
    ]
}`

type permissionBO struct {
    Id       int64      `json:"id"`
    Type     string     `json:"type"`
    Domain   string     `json:"domain"`
    Children []children `json:"children"`
}
type children struct {
    Id                 int64      `json:"id"`
    ParentId           int        `json:"parentId"`
    Name               string     `json:"name"`
    Route              string     `json:"route"`
    Path               string     `json:"path"`
    OperateMethod      string     `json:"operateMethod"`
    Description        string     `json:"description"`
    Children           []children `json:"children,omitempty"`
    DependenceChildren []int64    `json:"dependenceChildren,omitempty"`
    DependenceParents  []int64    `json:"dependenceParents,omitempty"`
}

func LoadPermission(ctx context.Context, data *common.Data) bool {
    if data == nil {
        fmt.Println("data is nil")
        return false
    }
    permission := new(permissionBO)
    err := json.Unmarshal([]byte(permissionJson), permission)
    if err != nil {
        fmt.Println("permission json Unmarshal fail", err)
        return false
    }
    childMap := map[string]schema.Permission{}
    getAllPermission(permission.Id, permission.Children, childMap)
    fmt.Println(childMap)
    //existPermissions := getCurrentPermission()
    //fmt.Println(existPermissions)
    updatePermission(ctx, permission.Domain, data, childMap)
    return true
}

func updatePermission(ctx context.Context, domain string, data *common.Data, permissions map[string]schema.Permission) {

    for _, permission := range permissions {
        err := data.DB(ctx).Clauses(clause.OnConflict{
            Columns: []clause.Column{{Name: "code"}, {Name: "type"}},
            DoUpdates: clause.Assignments(map[string]interface{}{
                "name":                permission.Name,
                "domain":              domain,
                "path":                permission.Path,
                "type":                permission.Type,
                "description":         permission.Description,
                "dependence_children": permission.DependenceChildren,
                "dependence_parents":  permission.DependenceParents,
            }),
        }).Create(&permission).Error
        if err != nil {
            fmt.Println("create or update fail ", err)
        }

    }

}
func getAllPermission(parentId int64, childrenSlice []children, childMap map[string]schema.Permission) {
    for _, child := range childrenSlice {
        childrenIds := make([]int64, 0, len(child.Children))
        for _, c := range child.Children {
            childrenIds = append(childrenIds, c.Id)
        }
        child.DependenceChildren = childrenIds
        if len(child.Path) > 0 {
            permission := schema.Permission{
                Code:               child.Id,
                Name:               child.Name,
                Path:               child.Path,
                Type:               0,
                Description:        &child.Description,
                DependenceChildren: child.DependenceChildren,
                DependenceParents:  parentId,
            }
            key := fmt.Sprintf("%d#%d", child.Id, 0)
            (childMap)[key] = permission
        }
        if len(child.Route) > 0 {
            permission := schema.Permission{
                Code:               child.Id,
                Name:               child.Name,
                Path:               child.Route,
                Type:               1,
                Description:        &child.Description,
                DependenceChildren: child.DependenceChildren,
                DependenceParents:  parentId,
            }
            key := fmt.Sprintf("%d#%d", child.Id, 1)
            (childMap)[key] = permission
        }
        if len(child.Route) == 0 && len(child.Path) == 0 {
            permission := schema.Permission{
                Code:               child.Id,
                Name:               child.Name,
                Type:               2,
                Description:        &child.Description,
                DependenceChildren: child.DependenceChildren,
                DependenceParents:  parentId,
            }
            key := fmt.Sprintf("%d#%d", child.Id, 2)
            (childMap)[key] = permission
        }

        if child.Children != nil {
            getAllPermission(child.Id, child.Children, childMap)
        }
    }
}
