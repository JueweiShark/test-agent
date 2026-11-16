package wuwen

import (
    "encoding/json"
    "path/filepath"
    "testing"

    "github.com/go-kratos/kratos/v2/log"
)

func TestDecompress(t *testing.T) {

    outDIR := "/Users/wangxmao/Downloads/decompressTest"
    err := Decompress("/Users/wangxmao/Downloads/point_cloud.zip", outDIR)
    if err != nil {
        return
    }
    _ = Decompress("/Users/wangxmao/workspace/wuwen/wwwdi.2024_01_17_wangnan.tar.gz", outDIR)
    _ = Decompress("/Users/wangxmao/workspace/godemo/go-fiber.tar", outDIR)
    _ = Decompress("/Users/wangxmao/workspace/godemo/rmq.tar.bz2", outDIR)
    _ = Decompress("/Users/wangxmao/workspace/godemo/rmq/README.md", outDIR)
}

type StructA struct {
    Obs []StructB
}
type StructB struct {
    Index int
}

var jsons = "[{\"Index\":2},{}]"

func TestContains(t *testing.T) {

    abs, err := filepath.Abs("./bean.go")
    dir := filepath.Dir(abs)
    log.Info(abs)
    log.Info(err)
    log.Info(dir)

    strut := StructA{
        Obs: []StructB{},
    }
    inner(&strut)
    i := new([]StructB)
    json.Unmarshal([]byte(jsons), i)

    log.Infof("test end %v", i)
}
func inner(a *StructA) {
    for i := 0; i < 10; i++ {
        a.Obs = append(a.Obs, StructB{
            Index: i,
        })
    }

}

func TestCompressFolder(t *testing.T) {
    source := "/Users/wangxmao/workspace/tetete"
    dest := "/Users/wangxmao/workspace/tetete.zip"
    err := CompressFolder(source, dest)
    if err != nil {
        log.Error(err)
        return
    }
}
