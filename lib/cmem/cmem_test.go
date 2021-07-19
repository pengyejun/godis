package cmem

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"
)

func TestMalloc(t *testing.T) {
	ptr := Malloc(10)
	*(* byte)(ptr) = 97

	if *(* byte)(ptr) != 97 {
		t.Error("Malloc Error")
	}

	p := reflect.StringHeader{Data:uintptr(ptr), Len:2}
	fmt.Println(*(* string)(unsafe.Pointer(&p)))

	*(* uint32)(ptr) = 258
	if *(* byte)(ptr) != 2 {
		t.Error("Malloc Error")
	}
	if *(* uint32)(ptr) != 258 {
		t.Error("Malloc Error")
	}

	Free(ptr)
}

func TestFree(t *testing.T) {
	ptr := Malloc(1000)
	Free(ptr)
}

func TestReAlloc(t *testing.T) {
	ptr := Malloc(1024 * 1024 * 2)
	*(* byte)(ptr) = 127
	* (* uint32)(unsafe.Pointer(uintptr(ptr) + 1)) = 657
	ptr = ReAlloc(ptr, 5)
	if *(* byte)(ptr) != 127 {
		t.Error("ReAlloc Error")
	}

	if * (* uint32)(unsafe.Pointer(uintptr(ptr) + 1)) != 657 {
		t.Error("ReAlloc Error")
	}

	ptr = ReAlloc(ptr, 0)
	if ptr != nil {
		t.Error("ReAlloc Error")
	}
}

func TestMemMove(t *testing.T) {
	ptr := Malloc(1024 * 1024 * 2)
	*(* byte)(ptr) = 127
	*(* byte)(unsafe.Pointer(uintptr(ptr) + 1)) = 125
	MemMove(unsafe.Pointer(uintptr(ptr) + 1), ptr, 2)
	if *(* byte)(unsafe.Pointer(uintptr(ptr) + 1)) != 127 {
		t.Error("MemMove Error")
	}
	if *(* byte)(unsafe.Pointer(uintptr(ptr) + 2))  != 125 {
		t.Error("MemMove Error")
	}

}

func TestMemCopy(t *testing.T) {
	ptr := Malloc(1024 * 1024 * 2)
	ptr1 := Malloc(20)
	*(* byte)(ptr) = 127
	MemCpy(ptr1, ptr, 1)
	if *(* byte)(ptr) != 127 {
		t.Error("MemCpy Error")
	}
	if *(* byte)(ptr1) != 127 {
		t.Error("MemCpy Error")
	}
}

func TestMemCmp(t *testing.T) {
	ptr := Malloc(1024 * 1024 * 2)
	cmpLen := 20
	ptr1 := Malloc(uint(cmpLen))

	for i := 0; i < cmpLen; i++ {
		*(* byte)(unsafe.Pointer(uintptr(ptr) + uintptr(i))) = byte(i)
		*(* byte)(unsafe.Pointer(uintptr(ptr1) + uintptr(i))) = byte(i)
	}

	if MemCmp(ptr, ptr1, uint(cmpLen)) != 0 {
		t.Error("MemCmp Error")
	}

	var i int
	for i = 0; i < cmpLen; i++ {
		*(* byte)(unsafe.Pointer(uintptr(ptr) + uintptr(i))) = byte(i)
		*(* byte)(unsafe.Pointer(uintptr(ptr1) + uintptr(i))) = byte(i)
	}

	*(* byte)(unsafe.Pointer(uintptr(ptr1) + uintptr(i-1))) = 10

	if MemCmp(ptr, ptr1, uint(cmpLen)) == 0 {
		t.Error("MemCmp Error")
	}

	if MemCmp(ptr, ptr1, uint(cmpLen-1)) != 0 {
		t.Error("MemCmp Error")
	}
}

