package wuwen

import (
    "image"
    "image/jpeg"
    _ "image/png"
    "os"

    "github.com/go-kratos/kratos/v2/log"
    "github.com/nfnt/resize"
)

type Image struct {
    Height int
    Width  int
}

func OpenImage(path string) Image {
    imageStruct := Image{
        Width:  0,
        Height: 0,
    }
    file, err := os.Open(path)
    if err != nil {
        log.Errorf("Failed to open file:%v", err)
        return imageStruct
    }
    defer func(file *os.File) {
        err := file.Close()
        if err != nil {
            log.Error(err)
        }
    }(file)
    // 解码PNG文件
    img, _, err := image.Decode(file)
    if err != nil {
        log.Errorf("Failed to decode PNG:%v", err)
        return imageStruct
    }
    imageStruct.Width = img.Bounds().Dx()
    imageStruct.Height = img.Bounds().Dy()

    return imageStruct
}

type Thumbnails struct {
    path       string
    width      int
    height     int
    targetFile string
}

func ThumbnailsOf(path string) *Thumbnails {

    return &Thumbnails{
        path: path,
    }
}
func (t *Thumbnails) Size(width, height int) *Thumbnails {
    t.width = width
    t.height = height
    return t
}
func (t *Thumbnails) To(path string) error {
    t.targetFile = path
    return t.generate()
}

func (t *Thumbnails) generate() error {
    file, err := os.Open(t.path)
    if err != nil {
        log.Error(t.path, err)
        return err
    }
    defer func(file *os.File) {
        err := file.Close()
        if err != nil {
            log.Error(t.path, err)
        }
    }(file)

    // 解码图片文件
    img, _, err := image.Decode(file)
    if err != nil {
        log.Error(t.path, err)
        return err
    }

    // 调整图片大小
    newWidth := t.width
    newHeight := t.height
    resizedImg := resize.Resize(uint(newWidth), uint(newHeight), img, resize.Lanczos3)

    // 创建新图片文件
    outputFile, err := os.Create(t.targetFile)
    if err != nil {
        log.Error(t.path, err)
        return err
    }
    defer outputFile.Close()

    // 保存调整大小后的图片到文件
    err = jpeg.Encode(outputFile, resizedImg, nil)
    if err != nil {
        log.Error(t.path, err)
        return err
    }

    log.Info("Image resized and saved.")
    return nil
}
