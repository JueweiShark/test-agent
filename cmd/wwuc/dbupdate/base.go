package dbupdate

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"wuwen.com/wwuc/internal/data/common"
)

type dbUpdateType string

const (
    dbUpdateTypeInit     dbUpdateType = "init"
    dbUpdateTypeUpdate   dbUpdateType = "update"
    dbUpdateTypeLoadData dbUpdateType = "load"
)

type InitMethod interface {
    MethodName() string
    Do(ctx context.Context, data *common.Data) bool
}

var MethodMap = map[string]InitMethod{
    loadPermissions.MethodName():        loadPermissions,
    loadSA.MethodName():                 loadSA,
    createAdmin.MethodName():            createAdmin,
}

func addOrUpdateColumn(db *gorm.DB, tableModel schema.Tabler, column, definition string) bool {
    migrator := db.Migrator()
    var sql string
    if migrator.HasColumn(tableModel, column) {
        sql = fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s  %s", tableModel.TableName(), column, definition)
    } else {
        sql = fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s  %s", tableModel.TableName(), column, definition)
    }
    err := db.Exec(sql).Error
    if err != nil {
        fmt.Println("sql exec fail：", sql, err)
        return false
    }
    return true
}
func addOrUpdateIndex(db *gorm.DB, tableModel schema.Tabler, index, definition string) bool {
    migrator := db.Migrator()
    var sql string
    if !migrator.HasIndex(tableModel, index) {
        sql = fmt.Sprintf("ALTER TABLE %s ADD INDEX %s  %s", tableModel.TableName(), index, definition)
    } else {
        fmt.Println("index is already exists ", index, definition)
        return false
    }
    err := db.Exec(sql).Error
    if err != nil {
        fmt.Println("sql exec fail：", sql, err)
        return false
    }
    return true
}
func createTable(db *gorm.DB, sql string) bool {
    // 删除注释行
    lines := strings.Split(sql, "\n")
    var filteredLines []string
    for _, line := range lines {
        if !strings.HasPrefix(strings.TrimSpace(line), "--") {
            filteredLines = append(filteredLines, line)
        }
    }
    filteredSQL := strings.Join(filteredLines, "\n")
    // 拆分 SQL 语句
    sqlStatements := strings.Split(filteredSQL, ";")
    // 执行 SQL 语句

    for _, statement := range sqlStatements {
        trimSapceSql := strings.TrimSpace(statement)
        if len(trimSapceSql) == 0 {
            continue
        }
        err := db.Debug().Exec(trimSapceSql).Error
        if err != nil {
            fmt.Println("sql exec fail：", statement, err)
        }
    }
    return true
}

type CreateTable struct {
    name string
    sql  string
}

func NewCreateTable(name, sql string) CreateTable {
    return CreateTable{
        name: name,
        sql:  sql,
    }
}
func (c CreateTable) Do(db *common.Data) bool {
    ctx, cancelFunc := context.WithCancel(context.Background())
    defer cancelFunc()
    return createTable(db.DB(ctx), c.sql)
}
func (c CreateTable) MethodName() string {
    return c.name
}

type UpdateMethod struct {
    methodName   string
    updateColumn map[string]string
    updateIndex  map[string]string
    tableModel   schema.Tabler
}

func NewUpdateMethod(tableModel schema.Tabler, methodName string, updateColumnInfo, updateIndexInfo map[string]string) *UpdateMethod {
    return &UpdateMethod{
        methodName:   methodName,
        updateColumn: updateColumnInfo,
        updateIndex:  updateIndexInfo,
        tableModel:   tableModel,
    }
}
func (m *UpdateMethod) MethodName() string {
    return m.methodName
}
func (m *UpdateMethod) Do(db *common.Data) bool {
    ctx, cancelFunc := context.WithCancel(context.Background())
    defer cancelFunc()
    if m.updateColumn != nil {
        for column, definition := range m.updateColumn {
            if !addOrUpdateColumn(db.DB(ctx), m.tableModel, column, definition) {
                fmt.Println(m.tableModel, ":addOrUpdateColumn fail")
                return false
            }
        }
    } else if m.updateIndex != nil {
        for s := range m.updateIndex {
            if !addOrUpdateIndex(db.DB(ctx), m.tableModel, s, m.updateIndex[s]) {
                fmt.Println(m.tableModel, ":addOrUpdateIndex fail")
            }
        }
    } else {
        fmt.Println(m.tableModel, ": all is empty")
    }
    return true
}
