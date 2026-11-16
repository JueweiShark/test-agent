package wuwen

import (
    "archive/tar"
    "archive/zip"
    "compress/bzip2"
    "compress/gzip"
    "errors"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "os"
    "path"
    "path/filepath"
    "strings"
    "time"

    "github.com/go-kratos/kratos/v2/log"
)

const (
    TAR     = ".TAR"
    TAR_GZ  = ".TAR.GZ"
    TAR_BZ2 = ".TAR.BZ2"
    ZIP     = ".ZIP"
)

func RemoveUrlParameter(fileUrl string) string {

    if strings.Contains(fileUrl, "?") {
        fileUrl = fileUrl[0:strings.Index(fileUrl, "?")]
    }
    return fileUrl
}

func ValidateURL(urlStr string) bool {
    boo := validateURLS(urlStr)
    if !boo {
        encodedURLStr := url.QueryEscape(urlStr)
        return validateURLS(encodedURLStr)
    }
    return true
}

func validateURLS(urlStr string) bool {
    client := http.Client{
        Timeout: time.Second * 5,
    }

    resp, err := client.Get(urlStr)
    if err != nil {
        log.Info("URL error:", err)
        return false
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusOK {
        return true
    }

    return false
}
func Decompress(filePath, decompressPath string) error {
    if EndWithIgnoreCase(filePath, TAR, true) {
        // tar
        return UnTarTo(filePath, decompressPath)
    } else if EndWithIgnoreCase(filePath, TAR_GZ, true) {
        // tar.gz
        return UnTarGZTo(filePath, decompressPath)

    } else if EndWithIgnoreCase(filePath, TAR_BZ2, true) {
        // tar_bz2
        return UnBzip2To(filePath, decompressPath)
    } else if EndWithIgnoreCase(filePath, ZIP, true) {
        // zip
        return UnZipTo(filePath, decompressPath)
    }

    return errors.New("not support compress type ")
}

func UnBzip2To(source string, destination string) error {
    file, err := os.Open(source)
    if err != nil {
        return fmt.Errorf("无法打开源文件 %s: %v", source, err)
    }
    defer file.Close()

    // 解压缩 tar.bz2 文件
    bz2Reader := bzip2.NewReader(file)
    tarReader := tar.NewReader(bz2Reader)

    for {
        header, err := tarReader.Next()
        if err == io.EOF {
            break // 处理完所有条目
        } else if err != nil {
            return fmt.Errorf("无法读取 Tar 文件中的条目: %v", err)
        }

        // 构造目标路径
        targetPath := destination + "/" + header.Name

        switch header.Typeflag {
        case tar.TypeDir: // 如果是目录类型，则创建对应的目录
            err = os.MkdirAll(targetPath, 0755)
            if err != nil {
                return fmt.Errorf("无法创建目录 %s: %v", targetPath, err)
            }
        default: // 否则将文件写入目标位置
            writer, err := os.Create(targetPath)
            if err != nil {
                return fmt.Errorf("无法创建文件 %s: %v", targetPath, err)
            }
            _, err = io.Copy(writer, tarReader)
            if err != nil {
                return fmt.Errorf("无法复制文件内容到 %s: %v", targetPath, err)
            }
            writer.Close()
        }
    }
    return nil
}

func UnTarTo(source string, destination string) error {
    fileReader, err := os.OpenFile(source, os.O_RDONLY, 0644) // 打开源文件
    if err != nil {
        log.Error(err)
        return fmt.Errorf("无法打开源文件 %s: %v", source, err)
    }
    defer func(fileReader *os.File) {
        err := fileReader.Close()
        if err != nil {
            log.Error(err)
        }
    }(fileReader)

    tarReader := tar.NewReader(fileReader) // 创建Tar读取器

    for {
        header, err := tarReader.Next() // 获取下一个条目头部信息
        if err == io.EOF {
            break // 已经处理完所有条目
        } else if err != nil {
            log.Error(err)
            return fmt.Errorf("无法读取Tar文件中的条目: %v", err)
        }

        // 构造目标路径
        targetPath := destination + "/" + header.Name

        switch header.Typeflag {
        case tar.TypeDir: // 如果是目录类型，则创建对应的目录
            err = os.MkdirAll(targetPath, 0755)
            if err != nil {
                return fmt.Errorf("无法创建目录 %s: %v", targetPath, err)
            }
        default: // 否则将文件写入目标位置
            writer, err := os.Create(targetPath)
            if err != nil {
                return fmt.Errorf("无法创建文件 %s: %v", targetPath, err)
            }
            _, err = io.Copy(writer, tarReader)
            if err != nil {
                return fmt.Errorf("无法复制文件内容到 %s: %v", targetPath, err)
            }
            writer.Close()
        }
    }

    return nil
}
func UnZipTo(src, dst string) error {
    archive, err := zip.OpenReader(src)
    if err != nil {
        log.Error(err)
        return err
    }
    defer func(archive *zip.ReadCloser) {
        err := archive.Close()
        if err != nil {
            log.Error(err)
            return
        }
    }(archive)

    for _, item := range archive.File {
        if strings.Contains(item.Name, "__MACOSX") {
            continue
        }
        filePath := filepath.Join(dst, item.Name)
        if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
            log.Error(err)
            return err
        }

        if item.FileInfo().IsDir() {
            err := os.MkdirAll(filePath, os.ModePerm)
            if err != nil {
                log.Error(err)
                return err
            }
            continue
        }

        file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, item.Mode())
        if err != nil {
            log.Error(err)
            return err
        }

        fileInArchive, err := item.Open()
        if err != nil {
            log.Error(err)
            return err
        }

        if _, err := io.Copy(file, fileInArchive); err != nil {
            log.Error(err)
            return err
        }
    }
    return nil
}

// 将文件解压到指定目录
func UnTarGZTo(src, dst string) error {
    archive, err := os.Open(src)

    if err != nil {
        log.Error(err)
        return err
    }
    defer func(archive *os.File) {
        err := archive.Close()
        if err != nil {
            log.Error(err)
            return
        }
    }(archive)

    gr, err := gzip.NewReader(archive)
    if err != nil {
        log.Error(err)
        return err
    }

    defer func(gr *gzip.Reader) {
        err := gr.Close()
        if err != nil {
            log.Error(err)
            return
        }
    }(gr)

    tr := tar.NewReader(gr)
    for {
        f, err := tr.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Error(err)
            return err
        }
        filePath := path.Join(dst, f.Name)
        if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
            log.Error(err)
            return err
        }

        if f.FileInfo().IsDir() {
            err := os.MkdirAll(filePath, os.ModePerm)
            if err != nil {
                log.Error(err)
                return err
            }
            continue
        } else {
            fileName := path.Join(dst, f.Name)

            fw, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(f.Mode))
            if err != nil {
                log.Error(err)
                return err
            }
            defer func(fw *os.File) {
                err := fw.Close()
                if err != nil {
                    log.Error(err)
                    return
                }
            }(fw)

            if _, err := io.Copy(fw, tr); err != nil {
                log.Error(err)
                return err
            }
        }

    }

    return nil
}

// CompressFolder 压缩文件夹
func CompressFolder(sourceFolder, destinationFile string) error {
    // 检查源文件夹是否存在
    sourceInfo, err := os.Stat(sourceFolder)
    if err != nil {
        return err
    }
    if !sourceInfo.IsDir() {
        return errors.New("source is not a directory")
    }

    // 检查目标文件夹是否存在，如果不存在则创建
    destDir := filepath.Dir(destinationFile)
    if _, err := os.Stat(destDir); os.IsNotExist(err) {
        err := os.MkdirAll(destDir, os.ModePerm)
        if err != nil {
            return err
        }
    }

    // 创建目标文件
    destFile, err := os.Create(destinationFile)
    if err != nil {
        return err
    }
    defer func(destFile *os.File) {
        err := destFile.Close()
        if err != nil {
            log.Error("CompressFolder destFile.Close()", err)
        }
    }(destFile)

    // 创建 zip 写入器
    zipWriter := zip.NewWriter(destFile)
    defer func(zipWriter *zip.Writer) {
        err := zipWriter.Close()
        if err != nil {
            log.Error("CompressFolder zipWriter.Close()", err)
        }
    }(zipWriter)

    // 准备遍历文件夹
    err = filepath.Walk(sourceFolder, func(fp string, info os.FileInfo, err error) error {
        if filepath.Base(fp) == "." {
            return nil
        }
        if filepath.Base(fp) == ".." {
            return nil
        }
        if err != nil {
            return err
        }

        // 创建文件头
        header, err := zip.FileInfoHeader(info)
        if err != nil {
            return err
        }

        header.Name, err = filepath.Rel(sourceFolder, fp)
        if err != nil {
            return err
        }

        // 写入文件头
        writer, err := zipWriter.CreateHeader(header)
        if err != nil {
            return err
        }

        // 如果是文件，复制文件内容到 zip 文件中
        if !info.IsDir() {
            file, err := os.Open(fp)
            if err != nil {
                return err
            }
            defer file.Close()

            _, err = io.Copy(writer, file)
            if err != nil {
                return err
            }
        }

        return nil
    })
    if err != nil {
        return err
    }

    return nil
}
