package common

import (
    "context"
    "errors"
    "fmt"
    "strings"
    "time"

    "github.com/go-kratos/kratos/v2/log"
    "gorm.io/gorm"
    "gorm.io/gorm/clause"
)

func Create[M, T any](ctx context.Context, data *Data, m M, t T) error {
    if data == nil {
        return errors.New("data repo is nil")
    }
    err := data.DB(ctx).Model(m).Create(t).Error
    if err != nil {
        log.Context(ctx).Errorf("create %T error %+v", m, err)
        return err
    }
    return nil
}
func BatchCreate[M, T any](ctx context.Context, data *Data, m M, t T) error {
    if data == nil {
        return errors.New("data repo is nil")
    }
    err := data.DB(ctx).Model(m).Create(t).Error
    if err != nil {
        log.Context(ctx).Errorf("create %T error %+v", m, err)
        return err
    }
    return nil
}
func BatchCreateOnIgnore[M, T any](ctx context.Context, data *Data, m M, t T, columns ...string) error {
    if data == nil {
        return errors.New("data repo is nil")
    }
    sqlColumns := make([]clause.Column, len(columns))
    for i, col := range columns {

        sqlColumns[i] = clause.Column{Name: col}
    }
    err := data.DB(ctx).Model(m).Clauses(clause.OnConflict{
        Columns: sqlColumns,
        //DoUpdates: clause.Assignments(map[string]interface{}{
        //    "is_deleted": 0,
        //}),
        DoNothing: true,
    }).Create(t).Error
    if err != nil {
        log.Context(ctx).Errorf("Batch Create On Ignore %T error %+v", m, err)
        return err
    }
    return nil
}
func Update[M, T any](ctx context.Context, data *Data, m M, t T) error {
    if data == nil {
        return errors.New("data repo is nil")
    }
    err := data.DB(ctx).Debug().Model(m).Updates(t).Error
    if err != nil {
        log.Context(ctx).Errorf("update %T error %+v", m, err)
        return err
    }
    return nil
}
func FindOne[M any](ctx context.Context, data *Data, m M, id int64) (*M, error) {
    po := new(M)
    err := data.DB(ctx).Model(m).First(po, "id = ?", id).Error
    if err != nil {
        log.Context(ctx).Error("find file error", err)
        return nil, err
    }
    return po, nil
}
func BatchDelete[M any](ctx context.Context, data *Data, userId int64, m M, ids []int64) error {
    if data == nil {
        return errors.New("data repo is nil")
    }
    err := data.DB(ctx).Model(m).Debug().
        Where("deleted_unique_id = ?", 0).
        Where("id IN (?)", ids).
        Updates(map[string]interface{}{
            "deleted_unique_id": gorm.Expr("id"),
            "deleted_at":        time.Now(),
            "updated_by":        userId,
        }).Error
    if err != nil {
        log.Context(ctx).Errorf("create %T error %+v", m, err)
        return err
    }
    return nil
}

// QueryWithEscapedLike Global query function to handle LIKE with escaped characters
func QueryWithEscapedLike(db *gorm.DB, fieldName string, searchTerm string) *gorm.DB {
    escapedSearchTerm := escapeLikeSpecialChars(searchTerm)
    query := fmt.Sprintf("%s LIKE ? ESCAPE '\\\\'", fieldName)
    return db.Where(query, "%"+escapedSearchTerm+"%")
}

// escapeLikeSpecialChars 转义 SQL LIKE 模式中的特殊字符
func escapeLikeSpecialChars(str string) string {
    str = strings.ReplaceAll(str, "\\", "\\\\") // 转义反斜杠
    str = strings.ReplaceAll(str, "%", "\\%")   // 转义百分号
    str = strings.ReplaceAll(str, "_", "\\_")   // 转义下划线
    return str
}
