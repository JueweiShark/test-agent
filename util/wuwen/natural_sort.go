package wuwen

import (
    "fmt"
    "strconv"
    "strings"
)

// NaturalSortConvert 将字符串进行转换，将数字部分的长度添加到字符串前面，末尾添加999并减去0的数量。
func NaturalSortConvert(str string) string {
    var sb strings.Builder
    runes := []rune(str)
    tmp := -1

    for i, r := range runes {
        if isDigit(r) {
            if tmp < 0 {
                tmp = i
            }
        } else {
            if tmp >= 0 {
                // 添加字符前的数字部分到比较单元
                o := string(runes[tmp:i])
                addNumberLength(&sb, o)
                tmp = -1
            }
            sb.WriteRune(r)
        }
    }

    // 如果最后一个是数字，将最后的数字添加到列表
    if isDigit(runes[len(runes)-1]) {
        tmp = tmpOrDefault(tmp, len(runes)-1)
        o := string(runes[tmp:])
        addNumberLength(&sb, o)
    }

    return sb.String()
}

// addNumberLength 在字符串包含数字时，在字符串前添加数字的长度，末尾添加999并减去0的数量。
func addNumberLength(sb *strings.Builder, originalStr string) {
    n := strconv.Itoa(parseNumber(originalStr))
    sb.WriteString(fillWithZero(strconv.Itoa(len(n)), 3))
    sb.WriteString(n)
    sb.WriteString(fillWithZero(strconv.Itoa(999-len(originalStr)+len(n)), 3))
}

// isDigit 检查字符是否是数字
func isDigit(c rune) bool {
    return c >= '0' && c <= '9'
}

// tmpOrDefault 返回 tmp 的值，如果为负数，则返回默认值
func tmpOrDefault(tmp, defaultValue int) int {
    if tmp < 0 {
        return defaultValue
    }
    return tmp
}

// parseNumber 将字符串解析为数字
func parseNumber(s string) int {
    n, _ := strconv.Atoi(s)
    return n
}

// fillWithZero 使用零填充字符串到指定的长度
func fillWithZero(s string, length int) string {
    return fmt.Sprintf("%0*s", length, s)
}
