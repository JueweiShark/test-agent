package wuwen

import "strings"

// RemoveAll 函数实现在字符串中删除指定字符
func RemoveAll(str string, chars ...rune) string {
    if str == "" || len(chars) == 0 {
        return str
    }

    var builder strings.Builder
    builder.Grow(len(str))

    for _, c := range str {
        if !Contains(chars, c) {
            builder.WriteRune(c)
        }
    }

    return builder.String()
}

// removeAllLineBreaks 函数实现删除字符串中的所有换行符
func RemoveAllLineBreaks(str string) string {
    return RemoveAll(str, '\r', '\n')
}

// contains 函数用于检查切片中是否包含某个元素
func Contains(chars []rune, char rune) bool {
    for _, c := range chars {
        if c == char {
            return true
        }
    }
    return false
}
