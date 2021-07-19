package utils

import (
	"unsafe"
)


type Pointer struct {
	ptr unsafe.Pointer
}

func MakePointer(ptr unsafe.Pointer) Pointer {
	return Pointer{ptr:ptr}
}

var EmptyPointer = Pointer{nil}

func (p Pointer) Addr() uintptr{
	return uintptr(p.ptr)
}

func (p Pointer) Value() unsafe.Pointer{
	return p.ptr
}

func (p Pointer) IsNil() bool {
	return p.ptr == nil
}

func (p Pointer) Offset(offset int64) Pointer{
	if offset == 0 {
		return p
	}
	var ptr unsafe.Pointer
	if offset > 0 {
		ptr = unsafe.Pointer(uintptr(p.ptr) + uintptr(offset))
	} else {
		ptr = unsafe.Pointer(uintptr(p.ptr) - uintptr(-offset))
	}
	return Pointer{ptr:ptr}
}

func (p *Pointer) Move(offset int64) {
	p.ptr = p.Offset(offset).ptr
}

func (p Pointer) GetUint32() uint32{
	return * (* uint32)(p.ptr)
}

func (p Pointer) SetUint32(size uint32) {
	* (* uint32)(p.ptr) = size
}

func (p Pointer) GetUint16() uint16{
	return * (* uint16)(p.ptr)
}

func (p Pointer) SetUint16(size uint16) {
	* (* uint16)(p.ptr) = size
}

func (p Pointer) GetUint8() uint8{
	return * (* uint8)(p.ptr)
}

func (p Pointer) SetUint8(size uint8) {
	* (* uint8)(p.ptr) = size
}

func (p Pointer) GetInt8() int8 {
	return * (* int8)(p.ptr)
}

func (p Pointer) SetInt8(size int8) {
	* (* int8)(p.ptr) = size
}
