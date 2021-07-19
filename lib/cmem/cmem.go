package cmem
import "C"

/*
#include <stdlib.h>
#include <string.h>
*/
import "C"
import (
	"unsafe"
)


func Malloc(size uint) unsafe.Pointer {
	return C.malloc(C.size_t(size))
}

func Free(ptr unsafe.Pointer) {
	C.free(ptr)
}

func ReAlloc(ptr unsafe.Pointer, size uint) unsafe.Pointer {
	if size == 0 {
		if ptr != nil {
			Free(ptr)
		}
		return nil
	}

	if ptr == nil {
		return Malloc(size)
	} else {
		return C.realloc(ptr, C.size_t(size))
	}
}

func MemMove(p, q unsafe.Pointer, size uint) {
	C.memmove(p, q, C.size_t(size))
}

func MemCpy(p, q unsafe.Pointer, size uint) {
	C.memcpy(p, q, C.size_t(size))
}

func MemCmp(p, q unsafe.Pointer, size uint) int32 {
	return int32(C.memcmp(p, q, C.size_t(size)))
}
