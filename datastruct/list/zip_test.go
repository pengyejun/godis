package list

import (
	"fmt"
	"github.com/hdt3213/godis/lib/utils"
	"math"
	"runtime"
	"strconv"
	"strings"
	"testing"
)


func TestAll(t *testing.T) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("%d Kb\n",m.Alloc/1024)
	zl := MakeEmptyZipList()
	fmt.Println(zl.zlBytes())
	for i := 0; i < 600; i++ {
		zl = zl.Insert(ZipListTail, fmt.Sprintf("ab%d", i))
	}
	p := zl.index(501)
	e := &ZLEntry{}
	e.decode(p)
	fmt.Println(*e)
	entryValue := getValue(p)
	fmt.Println(entryValue)

	zl = zl.Insert(500, strings.Repeat("abcdefghijklmnopqrstuvwxyz", 10))
	fmt.Println("insert after-----")
	p = zl.index(499)
	e.reset()
	e.decode(p)
	fmt.Println(*e)
	entryValue = getValue(p)
	fmt.Println(entryValue)
	p = zl.index(500)
	e.reset()
	e.decode(p)
	fmt.Println(*e)
	entryValue = getValue(p)
	fmt.Println(entryValue)
	p = zl.index(501)
	e.reset()
	e.decode(p)
	fmt.Println(*e)
	entryValue = getValue(p)
	fmt.Println(entryValue)
}

type testValue struct {
	ZLEntryPrevLen
	ZLEntryMeta
	value string
	isNumber bool
}

func compareValue(testvalue testValue, p utils.Pointer) (bool, string){
	e := &ZLEntry{}
	e.decode(p)
	if e.prevRawLen != testvalue.prevRawLen {
		return false, fmt.Sprintf("Not Equal PrevRawLen, %d %d", e.prevRawLen, testvalue.prevRawLen)
	}

	if e.prevRawLenSize != testvalue.prevRawLenSize {
		return false, fmt.Sprintf("Not Equal prevRawLenSize, %d %d", e.prevRawLenSize, testvalue.prevRawLenSize)
	}

	if e.lenSize!= testvalue.lenSize {
		return false, fmt.Sprintf("Not Equal lenSize, %d %d", e.lenSize, testvalue.lenSize)
	}

	if e.len != testvalue.len {
		return false, fmt.Sprintf("Not Equal len, %d %d", e.len, testvalue.len)
	}

	if e.headerSize != testvalue.headerSize {
		return false, fmt.Sprintf("Not Equal headerSize, %d %d", e.headerSize, testvalue.headerSize)
	}


	pValue := getValue(p)

	switch pValue.(type) {
	case string:
		if testvalue.isNumber {
			return false, "Not Equal string Type"
		}
		pValue = pValue.(string)
		if pValue != testvalue.value {
			return false, fmt.Sprintf("Not Equal str value, %s, %s", pValue, testvalue.value)
		}
	case int64:
		if !testvalue.isNumber {
			return false, "Not Equal int64 Type"
		}
		pValue = pValue.(int64)
		var value int64
		if !utils.String2int64(testvalue.value, uint(len(testvalue.value)), &value) {
			return false, "value is not convert int64"
		}
		if pValue != value {
			return false, fmt.Sprintf("Not Equal int64 value, %d, %d", pValue, value)
		}
	}

	return true, ""
}

func getAddTestValues() []testValue{
	v1 := testValue{ZLEntryPrevLen{prevRawLen:0, prevRawLenSize:1}, ZLEntryMeta{lenSize:1, len:8, headerSize:2}, "abcdefgh", false}
	v2 := testValue{ZLEntryPrevLen{prevRawLen:10, prevRawLenSize:1}, ZLEntryMeta{lenSize:1, len:0, headerSize:2}, "12", true}
	v3 := testValue{ZLEntryPrevLen{prevRawLen:2, prevRawLenSize:1}, ZLEntryMeta{lenSize:1, len:1, headerSize:2}, "123", true}
	v4 := testValue{ZLEntryPrevLen{prevRawLen:3, prevRawLenSize:1}, ZLEntryMeta{lenSize:2, len:260, headerSize:3},
		strings.Repeat("abcdefghijklmnopqrstuvwxyz", 10), false}
	v5 := testValue{ZLEntryPrevLen{prevRawLen:263, prevRawLenSize:5}, ZLEntryMeta{lenSize:1, len:3, headerSize:6}, "65535", true}
	v6 := testValue{ZLEntryPrevLen{prevRawLen:9, prevRawLenSize:1}, ZLEntryMeta{lenSize:1, len:2, headerSize:2}, "-32768", true}
	v7 := testValue{ZLEntryPrevLen{prevRawLen:4, prevRawLenSize:1}, ZLEntryMeta{lenSize:1, len:4, headerSize:2},
		strconv.Itoa(int(math.Pow(2, 31)) - 1), true}
	v8 := testValue{ZLEntryPrevLen{prevRawLen:6, prevRawLenSize:1}, ZLEntryMeta{lenSize:5, len:26000, headerSize:6}, strings.Repeat("abcdefghijklmnopqrstuvwxyz", 1000), false}
	v9 := testValue{ZLEntryPrevLen{prevRawLen:26006, prevRawLenSize:5}, ZLEntryMeta{lenSize:1, len:8, headerSize:6}, strconv.Itoa(-int(math.Pow(2, 63))), true}
	return []testValue{v1, v2, v3, v4, v5, v6, v7, v8, v9}
}


func addAllValue(zl ZipList, testValues []testValue) ZipList{
	for _, v := range testValues {
		zl = zl.Add(v.value)
	}
	return zl
}


func TestZipList_Add(t *testing.T) {
	zl := MakeEmptyZipList()
	testValues := getAddTestValues()
	zl = addAllValue(zl, testValues)

	for i, v := range testValues {
		p := zl.index(i)
		ok, str := compareValue(v, p)
		if !ok {
			t.Error(str)
		}
	}
}

func TestZipList_Insert(t *testing.T) {
	zl := MakeEmptyZipList()

	testValues := getAddTestValues()
	zl = addAllValue(zl, testValues)

	var tmpValue, nextValue testValue
	tmpValue = testValue{ZLEntryPrevLen{prevRawLenSize:1, prevRawLen:10}, ZLEntryMeta{lenSize:1, len: 10, headerSize:2}, "defghijklk", false}
	nextValue = testValue{ZLEntryPrevLen{prevRawLenSize:1, prevRawLen:12}, ZLEntryMeta{lenSize:1, len:0, headerSize:2}, "12", true}
	zl = zl.Insert(1, tmpValue.value)
	var ok bool
	var str string

	ok, str = compareValue(tmpValue, zl.index(1))
	if !ok {
		t.Error(str)
	}

	ok, str = compareValue(nextValue, zl.index(2))
	if !ok {
		t.Error(str)
	}

	tmpValue =
}

func TestZipList_Remove(t *testing.T) {
	zl := MakeEmptyZipList()

	testValues := getAddTestValues()
	zl = addAllValue(zl, testValues)
	fmt.Println(zl)
	//zl.Remove()

}