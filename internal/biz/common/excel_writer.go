package common

import (
    "errors"
    "os"
    "path/filepath"

    "github.com/go-kratos/kratos/v2/log"
    "github.com/xuri/excelize/v2"
)

type ExcelConfig struct {
    ColWidth  []float64
    RowHeight []float64
}

func WriteData2Excel(fileName, localFileDir string, dataSlice [][]string, config ...ExcelConfig) (int64, string, error) {

    excelConfig := getConfig(config...)

    f := excelize.NewFile()
    setExcelRowColSize(excelConfig, f)

    defer func() {
        if err := f.Close(); err != nil {
            log.Error(err)
        }
    }()
    for index, data := range dataSlice {
        cell, err := excelize.CoordinatesToCellName(1, index+1)
        if err != nil {
            log.Error("read sheet cell fail :", err)
            return 0, "", errors.New("read sheet cell fail ")
        }
        err = f.SetSheetRow("Sheet1", cell, &data)
        if err != nil {
            log.Error("write data to cell fail:", err)
            return 0, "", errors.New("write data to cell fail")
        }
    }

    outPutFolder := localFileDir
    err := os.MkdirAll(outPutFolder, os.ModePerm)
    if err != nil {
        log.Error("create file parent fail ", err)
        return 0, "", err
    }
    outPutFile := filepath.Join(outPutFolder, fileName)
    if err := f.SaveAs(outPutFile); err != nil {
        log.Error("save excel file fail ", err)
        return 0, "", errors.New("save excel file fail ")
    }
    log.Info("outPutFile save success")
    statInfo, err := os.Stat(outPutFile)
    if err != nil {
        log.Error("get file  stat info fail ", err)
        return 0, "", errors.New("get file  stat info fail")
    }
    return statInfo.Size(), outPutFile, nil
}

func setExcelRowColSize(excelConfig *ExcelConfig, f *excelize.File) {
    if len(excelConfig.ColWidth) != 0 {
        for i := range excelConfig.ColWidth {
            err := f.SetColWidth("Sheet1", convertToExcelColumn(i+1), convertToExcelColumn(i+1), excelConfig.ColWidth[i])
            if err != nil {
                log.Error("SetColWidth fail ", err)
            }
        }

    }
    if len(excelConfig.RowHeight) != 0 {

        for i := range excelConfig.RowHeight {
            err := f.SetRowHeight("Sheet1", i+1, excelConfig.RowHeight[i])
            if err != nil {
                log.Error("SetRowHeight fail ", err)
            }
        }

    }
}

func getConfig(config ...ExcelConfig) *ExcelConfig {
    options := &ExcelConfig{}
    for _, opt := range config {
        options = &opt
    }
    return options
}
func convertToExcelColumn(num int) string {
    column := ""
    for num > 0 {
        remainder := (num - 1) % 26
        column = string(rune('A'+remainder)) + column
        num = (num - 1) / 26
    }
    return column
}
