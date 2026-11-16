package common

import (
    "testing"

    "github.com/go-kratos/kratos/v2/log"
)

func TestWriteData2File(t *testing.T) {

    strings := make([][]string, 0)
    strings = append(strings, []string{"word1", "word2", "word3", "word4", "word5"})
    strings = append(strings, []string{"word1", "word2", "word3", "word4", "word5"})
    _, _, err := WriteData2Excel("test", "2/3/2/2", strings, ExcelConfig{
        ColWidth: []float64{10, 10, 20, 30},
    })
    log.Error(err)
    for i := 0; i < 500; i++ {
        column := convertToExcelColumn(i + 1)
        log.Info(column)
    }

}
