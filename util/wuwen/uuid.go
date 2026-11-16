package wuwen

import (
    "crypto/md5"
    "encoding/hex"
    "math/big"
    "sync"

    "github.com/go-kratos/kratos/v2/log"
    "github.com/google/uuid"
)

var (
    once sync.Once
    snow *SnowFlake
)

func GenerateSonyNum() int64 {
    once.Do(func() {
        snow = &SnowFlake{}
        if err := snow.Init(1, 1); err != nil {
            log.Error()
        }
    })
    if id, err := snow.NextId(); err != nil {
        log.Error(err)
        return 0
    } else {
        return id
    }
}
func UUID() string {
    newUUID, err := uuid.NewRandom()
    if err != nil {
        log.Warn("uuid.NewUUID() base time and mac  new fail ...")
        newUUID = uuid.New()
    }
    return newUUID.String()
}
func StrMd5ToInt64(str string) int64 {
    hash := md5.Sum([]byte(str))
    hashString := hex.EncodeToString(hash[:])

    // 将十六进制字符串解释为 big.Int 类型
    hashInt := new(big.Int)
    hashInt.SetString(hashString, 16)

    // 将 big.Int 类型转换为 int64
    return hashInt.Int64()
}
