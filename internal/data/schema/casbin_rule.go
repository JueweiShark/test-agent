package schema

type CasbinRule struct {
    PType  string `json:"ptype" gorm:"column:ptype" description:"策略类型"`
    RoleId string `json:"role_id" gorm:"column:v0" description:"角色ID"`
    Path   string `json:"path" gorm:"column:v1" description:"api路径"`
    Method string `json:"method" gorm:"column:v2" description:"访问方法"`
}

func (model *CasbinRule) TableName() string {
    return "casbin_rule"
}
