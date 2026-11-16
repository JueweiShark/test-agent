package common

import (
    "context"
    "reflect"
    "time"

    "github.com/go-kratos/kratos/v2/log"
    "github.com/go-redis/redis/v8"
    "go.uber.org/zap"
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    gormLogger "gorm.io/gorm/logger"
    "gorm.io/plugin/opentelemetry/tracing"

    "wuwen.com/wwuc/internal/biz/common"
    "wuwen.com/wwuc/internal/conf"
    "wuwen.com/wwuc/util/wuwen/mapper"
)

// Data .
type Data struct {
    db  *gorm.DB
    rdb *redis.Client
    log *log.Helper
}
type contextTxKey struct{}

func (d *Data) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
    return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        ctx = context.WithValue(ctx, contextTxKey{}, tx)
        return fn(ctx)
    })
}

func (d *Data) RDB() *redis.Client {
    return d.rdb
}
func (d *Data) DB(ctx context.Context) *gorm.DB {
    tx, ok := ctx.Value(contextTxKey{}).(*gorm.DB)
    if ok {
        return tx
    }
    // 非事务查询也要挂上请求 ctx，否则 SQL span 会脱离当前 trace 变成独立 root trace
    return d.db.WithContext(ctx)
}

// NewTransaction .
func NewTransaction(d *Data) common.Transaction {
    return d
}

// NewData .
func NewData(db *gorm.DB, rdb *redis.Client, logger log.Logger) (*Data, func(), error) {
    cleanup := func() {
        log.NewHelper(logger).Info("closing the data resources")
    }
    db.Logger = db.Logger.LogMode(gormLogger.Info)
    log.Info(" create new data resources")
    l := log.NewHelper(log.With(logger, "module", "transaction/data"))
    d := &Data{
        db:  db,
        rdb: rdb,
        log: l,
    }
    //c.Database.Driver
    return d, cleanup, nil
}

// NewDB gorm Connecting to a Database
func NewDB(conf *conf.Bootstrap, logger log.Logger) *gorm.DB {
    log := log.NewHelper(log.With(logger, "module", "wsi/data/gorm"))

    db, err := gorm.Open(mysql.Open(conf.Data.Database.Source), &gorm.Config{
        TranslateError: true,
        NowFunc: func() time.Time {
            return time.Now().Local()
        },
    })
    if err != nil {
        log.Fatalf("failed opening connection to mysql: %v", err)
    }
    // 链路追踪：每条 SQL 作为 client span 挂到当前请求 trace 下（trace 未开启时 otel 为 noop，无额外开销）
    if err := db.Use(tracing.NewPlugin(tracing.WithoutMetrics())); err != nil {
        panic(err)
    }
    sqlDB, err := db.DB()
    if err != nil {
        log.Fatal(err)
    }
    sqlDB.SetConnMaxLifetime(time.Duration(conf.Data.Database.MaxLifeMinute) * time.Minute)
    sqlDB.SetMaxOpenConns(int(conf.Data.Database.MaxConnThreads))
    sqlDB.SetMaxIdleConns(int(conf.Data.Database.MaxConnThreads))
    return db
}
func NewRedis(conf *conf.Bootstrap, logger log.Logger) *redis.Client {
    l := log.NewHelper(log.With(logger, "module", "wsi/data/redis"))
    rdb := redis.NewClient(&redis.Options{
        Addr:         conf.Data.Redis.Addr,
        Password:     conf.Data.Redis.Pwd,
        DB:           int(conf.Data.Redis.Db),
        DialTimeout:  conf.Data.Redis.DialTimeout.AsDuration(),
        WriteTimeout: conf.Data.Redis.WriteTimeout.AsDuration(),
        ReadTimeout:  conf.Data.Redis.ReadTimeout.AsDuration(),
    })
    l.Info("redis connect success")
    return rdb

}
func ConvertData[T, W any](ctx context.Context, t *T) *W {
    var schemaObj W
    wType := reflect.TypeOf(&schemaObj).Elem()
    newInstance := reflect.New(wType).Interface().(*W)
    err := mapper.AutoMapper(t, newInstance)
    if err != nil {
        log.Context(ctx).Errorf("biz mapper to schema error, %+v,%+v", t, zap.Error(err))
        return nil
    }
    return newInstance
}
