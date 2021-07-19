package utils

import (
	"math"
)

// ToCmdLine convert strings to [][]byte
func ToCmdLine(cmd ...string) [][]byte {
	args := make([][]byte, len(cmd))
	for i, s := range cmd {
		args[i] = []byte(s)
	}
	return args
}

func ToCmdLine2(commandName string, args ...string) [][]byte {
	result := make([][]byte, len(args)+1)
	result[0] = []byte(commandName)
	for i, s := range args {
		result[i+1] = []byte(s)
	}
	return result
}

func ToCmdLine3(commandName string, args ...[]byte) [][]byte {
	result := make([][]byte, len(args)+1)
	result[0] = []byte(commandName)
	for i, s := range args {
		result[i+1] = s
	}
	return result
}

// Equals check whether the given value is equal
func Equals(a interface{}, b interface{}) bool {
	sliceA, okA := a.([]byte)
	sliceB, okB := b.([]byte)
	if okA && okB {
		return BytesEquals(sliceA, sliceB)
	}
	return a == b
}

// BytesEquals check whether the given bytes is equal
func BytesEquals(a []byte, b []byte) bool {
	if (a == nil && b != nil) || (a != nil && b == nil) {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	size := len(a)
	for i := 0; i < size; i++ {
		av := a[i]
		bv := b[i]
		if av != bv {
			return false
		}
	}
	return true
}


func String2int64(str string, strLen uint, value *int64) bool {

	var pLen uint = 0
	if strLen == pLen {
		return false
	}
	char := str[pLen]
	if strLen == 1 && char == '0' {
		if value != nil {
			*value = 0
		}
		return true
	}

	negative := false

	if char == '-' {
		negative = true
		pLen++
		if pLen == strLen {
			return false
		}
		char = str[pLen]
	}

	var tmpValue uint64 = 0

	if char >= '1' && char <= '9' {
		tmpValue = uint64(char - '0')
		pLen++
		char = str[pLen]
	} else {
		return false
	}

	for pLen < strLen && char >= '0' && char <= '9' {
		if tmpValue > (math.MaxUint64 / 10) {
			return false
		}
		tmpValue *= 10

		if tmpValue > (math.MaxUint64 - uint64(char - '0')) {
			return false
		}

		tmpValue += uint64(char - '0')
		pLen++
		if pLen < strLen {
			char = str[pLen]
		}
	}

	if pLen < strLen {
		return false
	}

	if negative {
		if tmpValue > (uint64(-(math.MinInt64 + 1)) + 1) {
			return false
		}
		if value != nil {
			*value = -int64(tmpValue)
		}
	} else {
		if tmpValue > math.MaxInt64 {
			return false
		}
		if value != nil {
			*value = int64(tmpValue)
		}
	}
	return true
}
