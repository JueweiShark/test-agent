package wuwen

import (
    "bufio"
    "errors"
    "fmt"
    "io"
    "io/ioutil"
    "net/http"
    "os"
    "path/filepath"
    "strings"

    "github.com/gabriel-vasile/mimetype"
    "github.com/go-kratos/kratos/v2/log"
)

var extensionMap map[string]string = map[string]string{
    ".jpg":  "image/jpeg",
    ".jpe":  "image/jpeg",
    ".fpx":  "image/vnd.fpx",
    ".pbm":  "image/x-portable-bitmap",
    ".pnm":  "image/x-portable-anymap",
    ".ras":  "image/x-cmu-rast",
    ".svgz": "image/svg+xml",
    ".png":  "image/png",
    ".jfif": "image/jpeg",
    ".rgb":  "image/x-rgb",
    ".gif":  "image/gif",
    ".pgm":  "image/x-portable-graymap",
    ".xwd":  "image/x-xwindowdump",
    ".jpeg": "image/jpeg",
    ".fpix": "image/vnd.fpx",
    ".bmp":  "image/bmp",
    ".svg":  "image/svg+xml",
    ".ppm":  "image/x-portable-pixmap",
    ".xbm":  "image/x-xbitmap",
    ".ief":  "image/ief",
    ".tiff": "image/tiff",
    ".tif":  "image/tif",
}

// ListFiles 获取目录下的所有文件
func ListFiles(path string) ([]os.DirEntry, error) {
    stat, err := os.Stat(path)
    if err != nil {
        log.Error(err)
        return nil, err
    }
    if !stat.IsDir() {
        err := fmt.Errorf("path :%s is not a dir", path)
        log.Error(err)
        return nil, err
    }
    fileInfos, err := os.ReadDir(path)
    if err != nil {
        log.Error(err)
        return nil, err
    }
    return fileInfos, nil
}

// IsEmpty 判断文件或目录是否为空
func FileIsEmpty(path string) (bool, error) {
    fileInfo, err := os.Stat(path)
    if err != nil {
        log.Error(err)
        return false, err
    }

    if fileInfo.IsDir() {
        // 如果是目录
        files, err := ioutil.ReadDir(path)
        if err != nil {
            log.Error(err)
            return false, err
        }
        return len(files) == 0, nil
    } else if fileInfo.Mode().IsRegular() {
        // 如果是文件
        return fileInfo.Size() == 0, nil
    }

    return false, nil
}

func getUserHomePath() string {
    home, err := os.UserHomeDir()
    if err != nil {
        log.Error(err)
        // 处理错误，例如无法获取用户家目录的情况
        return ""
    }
    return home
}

func normalize(path string) string {
    if path == "" {
        return ""
    }

    // 移除 "classpath:" 和 "file:" 前缀
    pathToUse := strings.TrimPrefix(path, "classpath:")
    pathToUse = strings.TrimPrefix(pathToUse, "file:")

    // 处理波浪号 "~"
    if strings.HasPrefix(pathToUse, "~") {
        pathToUse = filepath.Join(getUserHomePath(), pathToUse[1:])
    }

    // 替换路径分隔符
    pathToUse = filepath.ToSlash(pathToUse)

    // 处理 Windows 共享路径
    if strings.HasPrefix(pathToUse, `\\`) {
        pathToUse = "\\" + pathToUse
    }

    // 获取路径前缀
    prefix := ""
    prefixIndex := strings.Index(pathToUse, ":")
    if prefixIndex > -1 {
        prefix = pathToUse[:prefixIndex+1]
        if strings.HasPrefix(prefix, "/") {
            prefix = prefix[1:]
        }

        if !strings.Contains(prefix, "/") {
            pathToUse = pathToUse[prefixIndex+1:]
        } else {
            prefix = ""
        }
    }

    // 处理路径
    if strings.HasPrefix(pathToUse, "/") {
        prefix = prefix + "/"
        pathToUse = pathToUse[1:]
    }

    // 拆分路径元素
    pathList := strings.Split(pathToUse, "/")
    pathElements := make([]string, 0)
    tops := 0

    for i := len(pathList) - 1; i >= 0; i-- {
        element := pathList[i]
        if element != "." {
            if element == ".." {
                tops++
            } else if tops > 0 {
                tops--
            } else {
                pathElements = append(pathElements, element)
            }
        }
    }

    // 处理 ".." 元素
    if tops > 0 && prefix == "" {
        for tops > 0 {
            pathElements = append([]string{".."}, pathElements...)
            tops--
        }
    }

    // 组装路径
    return filepath.Join(prefix, filepath.Join(pathElements...))
}

func isAbsolutePath(path string) bool {
    if path == "" {
        return false
    }

    // 使用 filepath 包的 IsAbs 函数判断是否为绝对路径
    return filepath.IsAbs(path)
}

func WriteUtf8ToFile(filePath, content string) error {
    // 以追加写入的方式打开文件，如果文件不存在则创建
    file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
    if errors.Is(err, os.ErrNotExist) {
        if file, err = CreateFile(filePath); err != nil {
            log.Error(err)
            return err
        }
    } else if err != nil {
        log.Error(err)
        return err
    }

    defer file.Close()

    // 使用 bufio.NewWriter 封装文件写入
    writer := bufio.NewWriter(file)

    // 将字符串写入文件
    _, err = writer.WriteString(content)
    if err != nil {
        log.Error(err)
        return err
    }

    // 将缓冲区的内容刷新到文件
    err = writer.Flush()
    if err != nil {
        log.Error(err)
        return err
    }

    return nil
}

func GetFileName(path string) string {
    if len(path) == 0 {
        return ""
    }
    length := len(path)
    if isFileSeparator(rune(path[length-1])) {
        length--
    }

    begin := 0
    for i := length - 1; i >= 0; i-- {
        if isFileSeparator(rune(path[i])) {
            begin = i + 1
            break
        }
    }
    return path[begin:length]
}

func isFileSeparator(char rune) bool {
    return '/' == char || '\\' == char
}

func GetMimeType(path string) string {
    mimeType, err := getURLMimeType(path)

    if len(mimeType) == 0 {
        if EndWithIgnoreCase(path, ".css", true) {
            mimeType = "text/css"
        } else if EndWithIgnoreCase(path, ".js", true) {
            mimeType = "application/x-javascript"
        } else if EndWithIgnoreCase(path, ".rar", true) {
            mimeType = "application/x-rar-compressed"
        } else if EndWithIgnoreCase(path, ".7z", true) {
            mimeType = "application/x-7z-compressed"
        }
    }
    if len(mimeType) != 0 {
        return mimeType
    }

    mimeType, err = getFileMimeType(path)

    if len(mimeType) == 0 {
        mimeType = findByFileName(path)
    }
    if err != nil && len(mimeType) == 0 {
        log.Errorf("getFileMimeType ,%v", err)
    }
    return mimeType
}

func findByFileName(path string) string {
    ext := ""
    i := strings.LastIndex(path, "#")

    if i > 0 {
        path = path[:i-1]
    }

    i = strings.LastIndex(path, ".")
    i = maxInt(i, strings.LastIndex(path, "/"))
    i = maxInt(i, strings.LastIndex(path, "?"))

    if i != -1 && path[i] == '.' {
        ext = strings.ToLower(path[i:])
    }
    return extensionMap[ext]
}

func maxInt(a, b int) int {
    if a > b {
        return a
    }
    return b
}

func GetUrlFileContent(url string) (*[]byte, error) {
    // 发起HTTP GET请求
    resp, err := http.Get(url)
    if err != nil {
        return nil, fmt.Errorf("HTTP GET request failed: %w", err)
    }
    defer resp.Body.Close()
    // 检查HTTP响应状态码
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("HTTP GET request returned non-200 status code: %d", resp.StatusCode)
    }
    // 读取响应内容
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read HTTP response body: %w", err)
    }
    return &body, nil
}

func getURLMimeType(url string) (string, error) {
    resp, err := http.Get(url)
    if err != nil {
        log.Info(url, ",getURLMimeType err:", err)
        return "", err
    }
    defer resp.Body.Close()
    buffer := make([]byte, 512)

    _, err = io.ReadFull(resp.Body, buffer)
    detect := mimetype.Detect(buffer)
    return detect.String(), err
}

func getFileMimeType(filePath string) (string, error) {
    file, err := os.Open(filePath)
    if err != nil {
        log.Info(filePath, ",getFileMimeType err:", err)
        return "", err
    }
    defer file.Close()

    buffer := make([]byte, 512)
    _, err = file.Read(buffer)
    if err != nil {
        log.Info(filePath, ",getFileMimeType err:", err)
        return "", err
    }

    mimeType := http.DetectContentType(buffer)
    return mimeType, nil
}

func EndWithIgnoreCase(str, suffix string, ignoreCase bool) bool {
    if !ignoreCase {
        return strings.HasSuffix(str, suffix)
    }

    return strings.HasSuffix(strings.ToLower(str), strings.ToLower(suffix))
}

type DownloadProgress func(step, progress, totalsize int64)

type progressReader struct {
    Total    int64
    Progress int64
    Step     int64
    listener DownloadProgress
}

func NewProgressReader(total int64, listener DownloadProgress) *progressReader {
    return &progressReader{
        Total:    total,
        Progress: 0,
        Step:     0,
        listener: listener,
    }
}

func (p *progressReader) Write(b []byte) (n int, err error) {
    n = len(b)
    p.Progress += int64(n)
    p.Step++
    p.listener(p.Step, p.Progress, p.Total)
    return n, nil
}

func (p *progressReader) Finish() {
    p.listener(p.Step, p.Progress, p.Total)
}

func DownloadFile(fileURL, filePath string, progressOpts ...DownloadProgress) error {
    var progress *DownloadProgress
    saveFile, err := CreateFile(filePath)
    if err != nil {
        log.Errorf("create file fail %v", err)
        return errors.New("create file fail")
    }

    if len(progressOpts) != 0 {
        for _, opt := range progressOpts {
            progress = &opt
        }
        return DownloadFileFromUrl(fileURL, saveFile, *progress)
    } else {
        return DownloadFileFromUrl(fileURL, saveFile, func(step, progress, totalsize int64) {
            if step%5000 == 0 {
                log.Infof("%s  total %d ;progress:%d ", fileURL, totalsize, progress)
            }
        })
    }
}

func DownloadFileFromUrl(fileURL string, file *os.File, progress DownloadProgress) error {
    // 发起 HTTP GET 请求
    response, err := http.Get(fileURL)
    if err != nil || response == nil || response.StatusCode != 200 {
        log.Errorf("Failed to download file:%v,%v", err, response)
        return errors.New("url connect fail")
    }
    defer response.Body.Close()

    pr := NewProgressReader(response.ContentLength, progress)
    buf := make([]byte, 4*1024*1024) // 4MB 缓冲区
    for {
        n, err := response.Body.Read(buf)
        if n > 0 {
            _, writeErr := io.MultiWriter(file, pr).Write(buf[:n])
            if writeErr != nil {
                log.Error("file download fail>>", err)
                return errors.New("content write file fail")
            }
        }
        if err != nil {
            if err == io.EOF {
                break
            } else {
                log.Error("file download fail>>", err)
                return errors.New("content write file fail")
            }
        }
    }
    pr.Finish()
    return nil
}

func CreateFile(filePath string) (*os.File, error) {
    file, err := os.Create(filePath)
    if err != nil {
        // 检查错误类型
        log.Warn(err)
        if os.IsNotExist(err) {
            // 如果路径不存在，则创建路径
            err = os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
            if err != nil {
                log.Error("Failed to create directory:", err)
                return nil, err
            }

            // 创建文件
            file, err = os.Create(filePath)
            if err != nil {
                log.Error("Failed to create file:", err)
                return nil, err
            }
        } else {
            log.Error("Failed to create file:", err)
            return nil, err
        }
    }
    return file, nil
}

func CleanDir(dir string) bool {
    log.Warn("will clean dir ", dir)
    err := os.RemoveAll(dir)
    if err != nil {
        log.Errorf("dir:%s, clean fail err %v", dir, err)
        return false
    }
    return true
}

// WalkFun 每个文件都会回调这个方法
type WalkFun func(dir, name string, isDir bool) error

func WalkDepth(name string, maxDepth int, fn WalkFun) error {
    return walkDirs([]string{name}, 1, maxDepth, fn)
}

// 遍历每个dirs下的文件，并且返回所有dirs下是目录的文件
func walkDirs(dirs []string, dep, maxDepth int, fn WalkFun) error {
    var subDirs []string
    for _, d := range dirs {
        ff, err := os.ReadDir(d)
        if err != nil {
            log.Error(err)
            return err
        }
        for _, f := range ff {
            err = fn(d, f.Name(), f.IsDir())
            if err != nil {
                log.Error(err)
                return err
            }
            if f.IsDir() { // 如果是目录
                subDirs = append(subDirs, filepath.Join(d, f.Name()))
            }
        }
    }
    if len(subDirs) > 0 && dep < maxDepth {
        return walkDirs(subDirs, dep+1, maxDepth, fn)
    }
    return nil
}

func MoveFile(sourcePath, destinationPath string, override bool) {
    if !override && FileExist(destinationPath) {
        log.Infof("current model override :%t ,destinationPath :%s already exist ", override, destinationPath)
        return
    }
    err := os.MkdirAll(filepath.Dir(destinationPath), os.ModePerm)
    if err != nil {
        log.Infof("current model override :%t ,destinationPath :%s already exist ", override, destinationPath)
    }
    err = os.Rename(sourcePath, destinationPath)
    if err != nil {
        log.Errorf("移动文件失败:%v", err)
    }
}

func Copy(sourcePath, destinationPath string, overwrite bool) error {
    sourceInfo, err := os.Stat(sourcePath)
    if err != nil {
        log.Error(err)
        return err
    }

    if sourceInfo.IsDir() {
        return CopyDir(sourcePath, destinationPath, overwrite)
    }

    return CopyFile(sourcePath, destinationPath, overwrite)
}

func CopyFile(sourcePath, destinationPath string, overwrite bool) error {
    // 检查目标文件是否存在
    if !overwrite {
        _, err := os.Stat(destinationPath)
        if err == nil {
            return fmt.Errorf("destination file already exists")
        }
        if !os.IsNotExist(err) {
            log.Error(err)
            return err
        }
    }
    destinationDir := filepath.Dir(destinationPath)

    err := os.MkdirAll(destinationDir, os.ModePerm)
    if err != nil {
        log.Error(err)
        return err
    }
    sourceFile, err := os.Open(sourcePath)
    if err != nil {
        log.Error(err)
        return err
    }
    defer sourceFile.Close()

    destinationFile, err := os.Create(destinationPath)
    if err != nil {
        log.Error(err)
        return err
    }
    defer destinationFile.Close()

    _, err = io.Copy(destinationFile, sourceFile)
    if err != nil {
        log.Error(err)
        return err
    }

    return nil
}

func CopyDir(sourceDir, destinationDir string, overwrite bool) error {
    // 检查目标文件夹是否存在
    if !overwrite {
        _, err := os.Stat(destinationDir)
        if err == nil {
            return fmt.Errorf("destination directory already exists")
        }
        if !os.IsNotExist(err) {
            log.Error(err)
            return err
        }
    }
    err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            log.Error(err)
            return err
        }
        // 构建目标路径
        destinationPath := filepath.Join(destinationDir, path[len(sourceDir):])
        if info.IsDir() {
            // 创建目录
            err = os.MkdirAll(destinationPath, info.Mode())
            if err != nil {
                log.Error(err)
                return err
            }
        } else {
            // 复制文件
            err = CopyFile(path, destinationPath, overwrite)
            if err != nil {
                log.Error(err)
                return err
            }
        }
        return nil
    })
    if err != nil {
        log.Error(err)
        return err
    }
    return nil
}

func FileExist(path string) bool {
    _, err := os.Stat(path)
    if err == nil {
        return true
    } else if os.IsNotExist(err) {
        return false
    } else {
        log.Error(err)
    }
    return false
}

// WalkDir recursively walks the directory tree rooted at root and returns the relative paths of all files.
func WalkDir(root string) (map[string]struct{}, map[string]string, error) {
    files := make(map[string]struct{})
    emptyDirs := make(map[string]struct{})
    parentMap := make(map[string]string)

    err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if info.Name() == ".DS_Store" {
            return nil
        }
        // Return error if accessing the path failed
        if err != nil {
            return err
        }

        // Get the relative path
        relPath, err := filepath.Rel(root, path)
        if err != nil {
            return err
        }

        // Get the parent directory name
        parentDir := filepath.Dir(relPath)
        if parentDir == "." {
            parentDir = "" // root level
        }

        if filepath.Base(relPath) != "." {
            parentMap[relPath] = parentDir
        }

        if info.IsDir() {
            // Check if the directory is empty
            entries, err := os.ReadDir(path)
            if err != nil {
                return err
            }
            if len(entries) == 0 {
                // Add the empty directory to the list
                emptyDirs[relPath] = struct{}{}
            }
        } else {
            // Add the file to the list
            files[relPath] = struct{}{}
        }
        return nil
    })
    if err != nil {
        log.Error("WalkDir err:", err)
    }

    return files, parentMap, err
}
