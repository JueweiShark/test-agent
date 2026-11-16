package v2

import (
	"crypto/rand"
	"errors"
	"math/big"
	"unicode"
)

// 密码字符集 (排除易混淆字符 0/O/o/1/l/I)
const (
	pwdDigits   = "23456789"
	pwdUpper    = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	pwdLower    = "abcdefghijkmnpqrstuvwxyz"
	pwdSpecials = "!@#$%^&*()-_=+[]{}"
)

// 密码规则: 长度 8-32, 必须包含数字、大写字母、小写字母、至少一个特殊字符 (非数字非字母)
const (
	defaultPwdMinLen = 8
	defaultPwdMaxLen = 32
)

// ValidatePassword 校验密码规则:
// 长度 minLen~maxLen (默认 8-32), 必须包含数字、大写字母、小写字母、至少一个特殊字符.
func ValidatePassword(pwd string, minLen, maxLen int) error {
	if minLen <= 0 {
		minLen = defaultPwdMinLen
	}
	if maxLen <= 0 {
		maxLen = defaultPwdMaxLen
	}
	n := len(pwd)
	if n < minLen || n > maxLen {
		return errors.New("password length must be between 8 and 32")
	}
	var hasDigit, hasUpper, hasLower, hasSpecial bool
	for _, r := range pwd {
		switch {
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			hasSpecial = true
		}
	}
	if !hasDigit || !hasUpper || !hasLower || !hasSpecial {
		return errors.New("password must contain digit, upper, lower and special char")
	}
	return nil
}

// GenerateRandomPassword 按规则随机生成密码 (长度默认 12, 包含数字/大写/小写/特殊字符).
// 长度取 max(minLen, 8) 与 maxLen 的合法范围, 默认 12.
func GenerateRandomPassword(minLen, maxLen int) (string, error) {
	if minLen <= 0 {
		minLen = defaultPwdMinLen
	}
	if maxLen <= 0 {
		maxLen = defaultPwdMaxLen
	}
	// 长度至少要能容纳 4 类字符各一个
	if minLen < 4 {
		minLen = 4
	}
	// 默认生成 12 位, 但不超过 maxLen, 也不小于 minLen
	length := 12
	if length < minLen {
		length = minLen
	}
	if length > maxLen {
		length = maxLen
	}

	chars := pwdDigits + pwdUpper + pwdLower + pwdSpecials
	result := make([]byte, 0, length)

	// 先各放一个保证满足规则
	picks := []string{pwdDigits, pwdUpper, pwdLower, pwdSpecials}
	for _, set := range picks {
		if len(result) >= length {
			break
		}
		c, err := pickRandom(set)
		if err != nil {
			return "", err
		}
		result = append(result, c)
	}

	// 剩余位置从全字符集随机填充
	for len(result) < length {
		c, err := pickRandom(chars)
		if err != nil {
			return "", err
		}
		result = append(result, c)
	}

	// Fisher-Yates 打乱顺序, 避免前 4 位固定为各类首字符
	for i := len(result) - 1; i > 0; i-- {
		jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return "", err
		}
		j := int(jBig.Int64())
		result[i], result[j] = result[j], result[i]
	}

	return string(result), nil
}

func pickRandom(set string) (byte, error) {
	idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(set))))
	if err != nil {
		return 0, err
	}
	return set[idx.Int64()], nil
}
