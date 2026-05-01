package utils

import (
	"bytes"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unsafe"
)

// IsEmpty 是否是空字符串
func IsEmpty(s string) bool {
	if s == "" {
		return true
	}

	return strings.TrimSpace(s) == ""
}

// ConcatString 连接字符串
// NOTE: 性能比fmt.Sprintf和+号要好
func ConcatString(s ...string) string {
	if len(s) == 0 {
		return ""
	}
	var buffer bytes.Buffer
	for _, i := range s {
		buffer.WriteString(i)
	}
	return buffer.String()
}

// StringToUint64 字符串转uint64
func StringToUint64(str string) (uint64, error) {
	if str == "" {
		return 0, nil
	}
	valInt, err := strconv.Atoi(str)
	if err != nil {
		return 0, err
	}

	return uint64(valInt), nil
}

// StringToInt64 字符串转int64
func StringToInt64(str string) (int64, error) {
	if str == "" {
		return 0, nil
	}
	valInt, err := strconv.Atoi(str)
	if err != nil {
		return 0, err
	}

	return int64(valInt), nil
}

// StringToInt 字符串转int
func StringToInt(str string) (int, error) {
	if str == "" {
		return 0, nil
	}
	valInt, err := strconv.Atoi(str)
	if err != nil {
		return 0, err
	}

	return valInt, nil
}

// --------- 字节切片和字符串转换 ----------
// 性能很高, 原因在于底层无新的内存申请与拷贝

// BytesToString 字节切片转字符串
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// StringToBytes convert string to byte
func StringToBytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

// GetTotalWords 返回输入的字符串的字数，汉字和中文标点算 1 个字数，英文和其他字符 2 个算 1 个字数，不足 1 个算 1个
func GetTotalWords(str string) int {
	var total float64

	reg := regexp.MustCompile("/·|，|。|《|》|‘|’|”|“|；|：|【|】|？|（|）|、/")

	for _, r := range str {
		if unicode.Is(unicode.Scripts["Han"], r) || reg.Match([]byte(string(r))) {
			total = total + 1
		} else {
			total = total + 0.5
		}
	}

	return int(math.Ceil(total))
}
