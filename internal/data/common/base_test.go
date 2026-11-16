package common

import (
    "context"
    "errors"
    "testing"

    "github.com/go-kratos/kratos/v2/log"
    r "github.com/go-redis/redis/v8"
    "github.com/golang/protobuf/ptypes/duration"

    "wuwen.com/wwuc/internal/conf"
)

func TestResis(t *testing.T) {

    redis := NewRedis(&conf.Bootstrap{
        Data: &conf.Data{
            Redis: &conf.Data_Redis{
                Addr: "10.74.7.107:6379",
                Pwd:  "83sCl5lPh",
                Db:   9,
                DialTimeout: &duration.Duration{
                    Seconds: 10,
                },
                WriteTimeout: &duration.Duration{
                    Seconds: 10,
                },
                ReadTimeout: &duration.Duration{
                    Seconds: 10,
                },
            },
        },
    }, log.DefaultLogger)
    ctx := context.Background()
    count1, err := redis.Get(ctx, "test#12").Int64()
    log.Infof("get result1 %d err %+v", count1, err != nil && !errors.Is(err, r.Nil))
    err = redis.Incr(ctx, "test#12").Err()

    log.Infof("set result2 err %+v", err != nil && !errors.Is(err, r.Nil))

    count, err := redis.Get(ctx, "test#12").Int64()

    log.Infof("get result3 %d err %+v", count, err != nil && !errors.Is(err, r.Nil))
}
