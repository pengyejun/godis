package list

import (
	"fmt"
	"github.com/hdt3213/godis/lib/cmem"
	"github.com/hdt3213/godis/lib/utils"
	"math"
	"reflect"
	"unsafe"
)

type ZLEntryPrevLen struct {
	prevRawLenSize uint32
	prevRawLen uint32
}

type ZLEntryMeta struct {
	lenSize uint32
	len uint32
	headerSize uint32
	encoding byte
}


type ZLEntry struct {
	ZLEntryPrevLen
	ZLEntryMeta
	p utils.Pointer
}

type ZLEntryValue interface {}


type ZipList struct {
	ptr utils.Pointer
}

const (
	ZipListHead = 0
	ZipListTail = -1
	// zlbytes(uint32) zltail(uint32) zllen(uint16)
	ZipListBytesSize = 4
	ZipListTailSize = 4
	ZipListLenSize = 2
	ZipListHeaderSize = ZipListBytesSize + ZipListTailSize + ZipListLenSize
	ZipListEndSize    = 1
	ZipEnd = 255
	ZipBigPrevLen = 254
	// Different encoding/length possibilities
	ZipStrMask = 0xc0
	ZipIntMask = 0x30
	ZipStr06B = 0 << 6
	ZipStr14B = 1 << 6
	ZipStr32B = 2 << 6
	ZipInt16B = 0xc0 | 0<<4
	ZipInt32B = 0xc0 | 1<<4
	ZipInt64B = 0xc0 | 2<<4
	ZipInt24B = 0xc0 | 3<<4
	ZipInt8B = 0xfe

	// 4 bit integer immediate encoding |1111xxxx| with xxxx between 0001 and 1101.
	ZipIntImmMask = 0x0f /* Mask to extract the 4 bits value. To add one is needed to reconstruct the value. */
	ZipIntImmMin  = 0xf1 /* 11110001 */
	ZipIntImmMax  = 0xfd /* 11111101 */
	Int24Max      = 0x7fffff
	Int24Min      = -Int24Max - 1

	ZipEncodingSizeInvalid = 0xff
)


func (ep *ZLEntryPrevLen) decodePrevLen(p utils.Pointer) {
	ep.decodePrevLenSize(p)
	if ep.prevRawLenSize == 1 {
		ep.prevRawLen = uint32(p.GetUint8())
	} else {
		ep.prevRawLen = p.Offset(1).GetUint32()
	}
}

func (ep *ZLEntryPrevLen) decodePrevLenSize(p utils.Pointer) {
	if p.GetUint8() < ZipBigPrevLen {
		ep.prevRawLenSize = 1
	} else {
		ep.prevRawLenSize = 5
	}
}


func (em *ZLEntryMeta) decodeEncoding(p utils.Pointer) {
	em.encoding = p.GetUint8()
	if em.encoding < ZipStrMask {
		em.encoding &= ZipStrMask
	}
}


func (em *ZLEntryMeta) decodeLen(p utils.Pointer) {
	if em.encoding < ZipStrMask {
		if em.encoding == ZipStr06B {
			em.lenSize = 1
			em.len = uint32(p.GetUint8()) & 0x3f
		} else if em.encoding == ZipStr14B {
			em.lenSize = 2
			em.len = ((uint32(p.GetUint8()) & 0x3f) << 8) | uint32(p.Offset(1).GetUint8())
		} else if em.encoding == ZipStr32B {
			em.lenSize = 5
			em.len = p.Offset(1).GetUint32()
		} else {
			// bad encoding
			em.lenSize = 0
			em.len = 0
		}
	} else {
		em.lenSize = 1
		if em.encoding == ZipInt8B {
			em.len = 1
		} else if em.encoding == ZipInt16B {
			em.len = 2
		} else if em.encoding == ZipInt24B {
			em.len = 3
		} else if em.encoding == ZipInt32B {
			em.len = 4
		} else if em.encoding == ZipInt64B {
			em.len = 8
		} else if em.encoding >= ZipIntImmMin && em.encoding <= ZipIntImmMax {
			em.len = 0 /* 4 bit immediate */
		} else {
			// bad encoding
			em.lenSize, em.len = 0, 0
		}
	}
}


func ZipIsStr(encoding byte) bool {
	return (encoding & ZipStrMask) < ZipStrMask
}

func ZipTryEncoding(str string, len uint32, v *int64, encoding *byte) bool{
	var value int64
	if len >= 32 || len == 0 {
		return false
	}

	if utils.String2int64(str, uint(len), &value) {
		if value >= 0 && value <= 12 {
			*encoding = byte(ZipIntImmMin + value)
		} else if value >= math.MinInt8 && value <= math.MaxInt8 {
			*encoding = ZipInt8B
		} else if value >= math.MinInt16 && value <= math.MaxInt16 {
			*encoding = ZipInt16B
		} else if value >= Int24Min && value <= Int24Max {
			*encoding = ZipInt24B
		} else if value >= math.MinInt32 && value <= math.MaxInt32 {
			*encoding = ZipInt32B
		} else {
			*encoding = ZipInt64B
		}
		*v = value
		return true
	}
	return false
}

func zipLoadInteger(p utils.Pointer, encoding byte) int64{
	var i16 int16
	var i32 int32
	var i64, ret int64
	if encoding == ZipInt8B {
		ret = int64(p.GetInt8())
	} else if encoding == ZipInt16B {
		cmem.MemCpy(unsafe.Pointer(&i16), p.Value(), uint(unsafe.Sizeof(i16)))
		ret = int64(i16)
	} else if encoding == ZipInt24B {

		tmpPtr := utils.MakePointer(unsafe.Pointer(&i32))
		tmpPtr.Move(1)
		cmem.MemCpy(tmpPtr.Value(), p.Value(), uint(unsafe.Sizeof(i32) - 1))
		ret = int64(i32 >> 8)
	} else if encoding == ZipInt32B {
		cmem.MemCpy(unsafe.Pointer(&i32), p.Value(), uint(unsafe.Sizeof(i32)))
		ret = int64(i32)
	} else if encoding == ZipInt64B {
		cmem.MemCpy(unsafe.Pointer(&i64), p.Value(), uint(unsafe.Sizeof(i64)))
		ret = i64
	} else if encoding >= ZipIntImmMin && encoding <= ZipIntImmMax {
		ret = int64((encoding & ZipIntImmMask) - 1)
	} else {
		panic("error")
	}
	return ret
}

func zipIntSize(encoding byte) uint32{
	switch encoding {
	case ZipInt8B:
		return 1
	case ZipInt16B:
		return 2
	case ZipInt24B:
		return 3
	case ZipInt32B:
		return 4
	case ZipInt64B:
		return 8
	}

	if encoding >= ZipIntImmMin && encoding <= ZipIntImmMax {
		return 0
	}
	panic("error")
}

func zipPrevLenByteDiff(p utils.Pointer, length uint32) int32{
	ep := &ZLEntryPrevLen{}
	ep.decodePrevLenSize(p)
	return int32(zipStorePrevEntryLength(utils.EmptyPointer, length) - ep.prevRawLenSize)
}

func zipStoreEntryEncoding(p utils.Pointer, encoding byte, rawLen uint32) uint32{
	var length uint32 = 1
	var buf [5]byte

	if ZipIsStr(encoding) {
		if rawLen <= 0x3f {
			if p == utils.EmptyPointer {
				return length
			}
			buf[0] = byte(ZipStr06B | rawLen)
		} else if rawLen <= 0x3fff {
			length += 1
			if p == utils.EmptyPointer {
				return length
			}
			buf[0] = byte(ZipStr14B | ((rawLen >> 8) & 0x3f))
			buf[1] = byte(rawLen & 0xff)
		} else {
			length += 4
			if p == utils.EmptyPointer {
				return length
			}
			buf[0] = ZipStr32B
			tmpBuf := *(* [4]byte)(unsafe.Pointer(&rawLen))
			for i := 0; i < 4; i++ {
				buf[i + 1] = tmpBuf[i]
			}
		}
	} else {
		if p == utils.EmptyPointer {
			return length
		}
		buf[0] = encoding
	}
	cmem.MemCpy(p.Value(), unsafe.Pointer(&buf), uint(length))
	if rawLen == 26000 {
		fmt.Println("-----")
		fmt.Println(p.GetUint8())
		fmt.Println(p.Offset(1).GetUint32())
	}
	return length
}

func zipStorePrevEntryLength(p utils.Pointer, length uint32) uint32{
	if p.IsNil() {
		if length < ZipBigPrevLen {
			return 1
		} else {
			return 5
		}
	} else {
		if length < ZipBigPrevLen {
			p.SetUint8(uint8(length))
			return 1
		} else {
			return zipStorePrevEntryLengthLarge(p, length)
		}
	}
}

func zipStorePrevEntryLengthLarge(p utils.Pointer, length uint32) uint32{
	if !p.IsNil() {
		p.SetUint8(ZipBigPrevLen)
		fmt.Println(uint(unsafe.Sizeof(length)))
		p.Offset(1).SetUint32(length)
		fmt.Println(p.Offset(1).GetUint32())
	}
	return 5
}

func zipSaveInteger(p utils.Pointer, value int64, encoding byte) {
	var i16 int16
	var i32 int32
	var i64 int64
	if encoding == ZipInt8B {
		p.SetInt8(int8(value))
	} else if encoding == ZipInt16B {
		i16 = int16(value)
		cmem.MemCpy(p.Value(), unsafe.Pointer(&i16), uint(unsafe.Sizeof(i16)))
	} else if encoding == ZipInt24B {
		i32 = int32(value << 8)
		tmpPtr := utils.MakePointer(unsafe.Pointer(&i32))
		tmpPtr.Move(1)
		cmem.MemCpy(p.Value(), tmpPtr.Value(), uint(unsafe.Sizeof(i32) - 1))
	} else if encoding == ZipInt32B {
		i32 = int32(value)
		cmem.MemCpy(p.Value(), unsafe.Pointer(&i32), uint(unsafe.Sizeof(i32)))
	} else if encoding == ZipInt64B {
		i64 = value
		cmem.MemCpy(p.Value(), unsafe.Pointer(&i64), uint(unsafe.Sizeof(i64)))
	} else if encoding >= ZipIntImmMin && encoding <= ZipIntImmMax {
		// Nothing to do, the value is stored in the encoding itself
	} else {
		panic("error")
	}
}

func MakeEmptyZipList() ZipList {
	var size uint32 = ZipListHeaderSize + ZipListEndSize
	zl := MakeZipList(size)
	zl.setZlTail(ZipListHeaderSize)
	return zl
}


func MakeZipList(size uint32) ZipList{
	ptr := cmem.Malloc(uint(size))
	zl := ZipList{utils.MakePointer(ptr)}
	zl.setZlBytes(size)
	zl.setZlEnd()
	return zl
}

func entryEncodingLenSize(encoding byte) uint32 {
	if encoding == ZipInt16B || encoding == ZipInt32B || encoding == ZipInt24B ||
		encoding == ZipInt64B || encoding == ZipInt8B {
		return 1
	}
	if encoding >= ZipIntImmMin && encoding <= ZipIntImmMax {
		return 1
	}
	if encoding == ZipStr06B {
		return 1
	}
	if encoding == ZipStr14B {
		return 2
	}
	if encoding == ZipStr32B {
		return 5
	}
	return ZipEncodingSizeInvalid
}

func (list ZipList) assertValidEntry(p utils.Pointer) {
	e := &ZLEntry{}
	if !list.entrySafe(p, e, true) {
		panic("error")
	}
}

func (list ZipList) zlBytes() uint32 {
	return list.ptr.GetUint32()
}

func (list ZipList) setZlBytes(size uint32) {
	list.ptr.SetUint32(size)
}

func (list ZipList) zlTail() uint32 {
	ptr := list.ptr.Offset(ZipListBytesSize)
	return ptr.GetUint32()
}

func (list ZipList) setZlTail(size uint32) {
	ptr := list.ptr.Offset(ZipListBytesSize)
	ptr.SetUint32(size)
}

func (list ZipList) zlLen() uint16{
	ptr := list.ptr.Offset(ZipListBytesSize + ZipListTailSize)
	return ptr.GetUint16()
}

func (list ZipList) setZlLen(size uint16) {
	ptr := list.ptr.Offset(ZipListBytesSize + ZipListTailSize)
	ptr.SetUint16(size)
}

func (list ZipList) zlEnd() uint8 {
	ptr := list.entryEnd()
	return ptr.GetUint8()
}

func (list ZipList) setZlEnd() {
	ptr := list.entryEnd()
	ptr.SetUint8(ZipEnd)
}

func (list ZipList) Len() int {
	zlLen := uint32(list.zlLen())
	if zlLen >= math.MaxUint16 {
		zlLen = 0
		headPtr := list.entryHead()
		for headPtr.GetUint8() != ZipEnd {
			headPtr.Move(int64(list.rawEntryLengthSafe(headPtr)))
			zlLen++
		}
		if zlLen < math.MaxUint16 {
			list.setZlLen(uint16(zlLen))
		}
	}

	return int(zlLen)
}


func (list ZipList) entryHead() utils.Pointer {
	return list.ptr.Offset(ZipListHeaderSize)
}

func (list ZipList) entryTail() utils.Pointer {
	tail := list.zlTail()
	return list.ptr.Offset(int64(tail))
}

func (list ZipList) entryEnd() utils.Pointer {
	zlBytes := list.zlBytes()
	ptr := list.ptr.Offset(int64(uintptr(zlBytes) - 1))
	return ptr
}

func (list ZipList) outOfRange(p utils.Pointer) bool {
	return p.Addr() < list.entryHead().Addr() || p.Addr() > list.entryEnd().Addr()
}

func (list ZipList) entrySafe(p utils.Pointer, e *ZLEntry, validatePrevLen bool) bool{
	firstPtr := list.entryHead()
	lastPtr := list.entryEnd()
	if p.Addr() > firstPtr.Addr() && p.Addr() + 10 < lastPtr.Addr() {
		e.decodePrevLen(p)
		tmpPtr := p.Offset(int64(e.prevRawLenSize))
		e.decodeEncoding(tmpPtr)
		e.decodeLen(tmpPtr)
		e.headerSize = e.prevRawLenSize + e.lenSize
		e.p = p

		if e.lenSize == 0 {
			return false
		}

		if list.outOfRange(p.Offset(int64(e.headerSize + e.len))) {
			return false
		}
		if validatePrevLen && list.outOfRange(p.Offset(-int64(e.prevRawLenSize))) {
			return false
		}
		return true
	}

	/* Make sure the pointer doesn't reach outside the edge of the zipList */
	if list.outOfRange(p) {
		return false
	}
	e.decodePrevLenSize(p)

	tmpPtr := p.Offset(int64(e.prevRawLenSize))
	if list.outOfRange(tmpPtr) {
		return false
	}
	e.decodeEncoding(tmpPtr)
	e.lenSize = entryEncodingLenSize(e.encoding)
	if e.lenSize == ZipEncodingSizeInvalid {
		return false
	}
	tmpPtr.Move(int64(e.lenSize))
	if list.outOfRange(tmpPtr) {
		return false
	}

	/* Decode the prevLen and entry len headers. */
	e.decodePrevLen(p)
	tmpPtr = p.Offset(int64(e.prevRawLenSize))
	e.decodeLen(tmpPtr)
	e.headerSize = e.prevRawLenSize + e.lenSize
	tmpPtr = p.Offset(int64(e.headerSize + e.len))
	/* Make sure the entry doesn't reach outside the edge of the zipList */
	if list.outOfRange(tmpPtr) {
		return false
	}
	tmpPtr = p.Offset(-int64(e.prevRawLen))
	if validatePrevLen && list.outOfRange(tmpPtr) {
		return false
	}
	e.p = p
	return true
}

func (e *ZLEntry) decode(p utils.Pointer) {
	e.decodePrevLen(p)
	tmpPtr := p.Offset(int64(e.prevRawLenSize))
	e.decodeEncoding(tmpPtr)
	e.decodeLen(tmpPtr)
	if e.lenSize == 0 {
		panic("error")
	}
	e.headerSize = e.prevRawLenSize + e.lenSize
	e.p = p
}

func (e *ZLEntry) reset() {
	e.prevRawLenSize = 0
	e.prevRawLen = 0
	e.lenSize = 0
	e.len = 0
	e.headerSize = 0
	e.encoding = 0
	e.p = utils.EmptyPointer

}


func (list ZipList) rawEntryLengthSafe(p utils.Pointer) uint32 {
	e := &ZLEntry{}
	if !list.entrySafe(p, e, false) {
		panic("error")
	}
	return e.headerSize + e.len
}

func (list ZipList) rawEntryLength(p utils.Pointer) uint32 {
	e := &ZLEntry{}
	e.decode(p)
	return e.headerSize + e.len
}

func (list ZipList) incrZlLen()  {
	length := list.zlLen()
	if length < math.MaxUint16 {
		list.setZlLen(length + 1)
	}
}

func (list ZipList) decrZlLen(delta uint32) {
	length := list.zlLen()
	if length < math.MaxUint16 && uint32(length) >= delta{
		list.setZlLen(length - uint16(delta))
	}
}

func (list ZipList) next(p utils.Pointer) utils.Pointer{
	if p.GetUint8() == ZipEnd {
		return utils.EmptyPointer
	}
	p.Move(int64(list.rawEntryLength(p)))

	if p.GetUint8() == ZipEnd {
		return utils.EmptyPointer
	}
	list.assertValidEntry(p)
	return p
}

func (list ZipList) prev(p utils.Pointer) utils.Pointer{
	if p.GetUint8()== ZipEnd {
		p = list.entryTail()
		if p.GetUint8() == ZipEnd {
			return utils.EmptyPointer
		}
		return p
	} else if p.Addr() == list.entryHead().Addr() {
		return utils.EmptyPointer
	} else {
		e := &ZLEntry{}
		e.decodePrevLen(p)
		if e.prevRawLen <= 0 {
			panic("error")
		}
		p.Move(-int64(e.prevRawLen))
		list.assertValidEntry(p)
		return p
	}
}

func getValue(p utils.Pointer) ZLEntryValue{
	e := &ZLEntry{}
	e.decode(p)
	if ZipIsStr(e.encoding) {
		var i uint32
		buffer := make([]byte, e.len)
		data := p.Offset(int64(e.headerSize))
		for i = 0; i < e.len; i++ {
			char := data.Offset(int64(i)).GetUint8()
			buffer[i] = char
		}
		return *(*string)(unsafe.Pointer(&buffer))
	} else {
		return zipLoadInteger(p.Offset(int64(e.headerSize)), e.encoding)
	}
}


func (list ZipList) cascadeUpdate(p utils.Pointer) ZipList{
	curEntry := &ZLEntry{}
	entryPrevlen := &ZLEntryPrevLen{}
	var prevOffset, offset uintptr
	var firstEntryLen uint32
	var extra, cnt uint32
	var rawLen uint32
	var delta uint32 = 4
	curLen := list.zlBytes()

	tailPtr := list.entryTail()

	if p.GetUint8() == ZipEnd {
		return list
	}

	curEntry.decode(p)

	firstEntryLen = curEntry.headerSize + curEntry.len
	entryPrevlen.prevRawLen = firstEntryLen
	entryPrevlen.prevRawLenSize = zipStorePrevEntryLength(utils.EmptyPointer, entryPrevlen.prevRawLen)
	prevOffset = p.Addr() - list.ptr.Addr()
	p.Move(int64(entryPrevlen.prevRawLen))

	for p.GetUint8() != ZipEnd {
		if !list.entrySafe(p, curEntry, false) {
			panic("error")
		}

		if curEntry.prevRawLen == entryPrevlen.prevRawLen {
			break
		}

		if curEntry.prevRawLenSize >= entryPrevlen.prevRawLenSize {
			if curEntry.prevRawLenSize == entryPrevlen.prevRawLenSize {
				zipStorePrevEntryLength(p, entryPrevlen.prevRawLen)
				curEntry.decode(p)
			} else {
				zipStorePrevEntryLengthLarge(p, entryPrevlen.prevRawLen)
			}
			break
		}

		if !(curEntry.prevRawLen == 0 || curEntry.prevRawLen + delta == entryPrevlen.prevRawLen) {
			panic("error")
		}

		rawLen = curEntry.headerSize + curEntry.len
		entryPrevlen.prevRawLen = rawLen + delta
		entryPrevlen.prevRawLenSize = zipStorePrevEntryLength(utils.EmptyPointer, entryPrevlen.prevRawLen)
		prevOffset = p.Addr() - list.ptr.Addr()
		p.Move(int64(rawLen))
		extra += delta
		cnt++
	}

	if extra == 0 {
		return list
	}

	if tailPtr.Addr() == (list.ptr.Addr() + prevOffset) {
		if extra - delta != 0 {
			list.setZlTail(list.zlTail() + extra - delta)
		}
	} else {
		list.setZlTail(list.zlTail() + extra)
	}

	offset = p.Addr() - list.ptr.Addr()
	zl := list.resize(curLen + extra)
	p = zl.ptr.Offset(int64(offset))
	cmem.MemMove(p.Offset(int64(extra)).Value(), p.Value(), uint(curLen-uint32(offset)-1))
	p.Move(int64(extra))

	for cnt > 0 {
		curEntry.decode(zl.ptr.Offset(int64(prevOffset)))
		rawLen = curEntry.headerSize + curEntry.len
		cmem.MemMove(p.Offset(-int64(rawLen - curEntry.prevRawLenSize)).Value(),
			zl.ptr.Offset(int64(uint32(prevOffset)+curEntry.prevRawLenSize)).Value(),
			uint(rawLen-curEntry.prevRawLenSize))
		p.Move(-int64(rawLen + delta))
		if curEntry.prevRawLen == 0 {
			zipStorePrevEntryLength(p, firstEntryLen)
		} else {
			zipStorePrevEntryLength(p, curEntry.prevRawLen + delta)
		}

		prevOffset -= uintptr(curEntry.prevRawLen)
		cnt--
	}
	return zl
}

func (list ZipList) find(str string, skip int32) utils.Pointer{
	var skipCnt int32 = 0
	var vEncoding byte = 0
	var v int64 = 0
	strLen := uint32(len(str))
	strPtr := utils.MakePointer(unsafe.Pointer((* reflect.StringHeader)(unsafe.Pointer(&str)).Data))
	p := list.entryHead()
	e := &ZLEntry{}
	var q utils.Pointer
	for p.GetUint8() != ZipEnd {
		e.reset()
		if !list.entrySafe(p, e, true) {
			panic("error")
		}
		q = p.Offset(int64(e.prevRawLenSize + e.lenSize))

		if skipCnt == 0 {
			if ZipIsStr(e.encoding) {
				if e.len == strLen && cmem.MemCmp(q.Value(), strPtr.Value(), uint(strLen)) == 0 {
					return p
				}
			} else {
				if vEncoding == 0 {
					if !ZipTryEncoding(str, strLen, &v, &vEncoding) {
						vEncoding = math.MaxUint8
					}

					if vEncoding != 0 {
						panic("error")
					}
				}

				if vEncoding != math.MaxUint8 {
					ll := zipLoadInteger(q, e.encoding)
					if ll == v {
						return p
					}
				}
			}
			skipCnt = skip
		} else {
			skipCnt--
		}
		p = q.Offset(int64(e.len))
	}
	return utils.EmptyPointer
}

func (list ZipList) index(index int) utils.Pointer{
	var p utils.Pointer
	if index == ZipListHead {
		p = list.entryHead()
		return p
	}
	if index == ZipListTail {
		p = list.entryEnd()
		return p
	}
	e := &ZLEntry{}
	if index < 0 {
		index = (-index) - 1
		p = list.entryTail()
		p.GetUint8()
		if p.GetUint8() != ZipEnd {
			e.decodePrevLen(p)
			for e.prevRawLen > 0 && index > 0 {
				p.Move(-int64(e.prevRawLen))
				if list.outOfRange(p) {
					panic("error")
				}
				e.decodePrevLen(p)
				index--
			}
		}
	} else {
		p = list.entryHead()
		for index > 0 {
			p.Move(int64(list.rawEntryLengthSafe(p)))
			if p.GetUint8() == ZipEnd {
				break
			}
			index--
		}
	}

	if p.GetUint8() == ZipEnd || index > 0 {
		return utils.EmptyPointer
	}
	list.assertValidEntry(p)
	return p
}

func (list ZipList) insert(p utils.Pointer, str string) ZipList{
	var reqLen, newLen uint32
	strLen := uint32(len(str))
	var value int64 = 123456789
	e := &ZLEntry{}
	if p.GetUint8() != ZipEnd {
		e.decodePrevLen(p)
	} else {
		tailPtr := list.entryTail()
		if tailPtr.GetUint8() != ZipEnd {
			e.prevRawLen = list.rawEntryLengthSafe(tailPtr)
		}
	}

	if ZipTryEncoding(str, strLen, &value, &e.encoding) {
		reqLen = zipIntSize(e.encoding)
	} else {
		reqLen = strLen
	}

	reqLen += zipStorePrevEntryLength(utils.EmptyPointer, e.prevRawLen)
	reqLen += zipStoreEntryEncoding(utils.EmptyPointer, e.encoding, strLen)

	forceLarge := false


	var nextDiff int32 = 0
	if p.GetUint8() != ZipEnd {
		nextDiff = zipPrevLenByteDiff(p, reqLen)
	}

	if nextDiff == -4 && reqLen < 4 {
		nextDiff = 0
		forceLarge = true
	}

	var offset = p.Addr() - list.ptr.Addr()
	curLen := list.zlBytes()
	newLen = curLen + reqLen + uint32(nextDiff)

	zl := list.resize(newLen)
	p = zl.ptr.Offset(int64(offset))

	if p.GetUint8() != ZipEnd {
		tmpPtr := p.Offset(int64(reqLen))
		cmem.MemMove(tmpPtr.Value(), p.Offset(-int64(nextDiff)).Value(), uint(int32(curLen)-int32(offset)-1+nextDiff))

		if forceLarge {
			zipStorePrevEntryLengthLarge(tmpPtr, reqLen)
		} else {
			zipStorePrevEntryLength(tmpPtr, reqLen)
		}
		zl.setZlTail(zl.zlTail() + reqLen)
		tailEntry := &ZLEntry{}
		if !zl.entrySafe(tmpPtr, tailEntry, true) {
			panic("error")
		}
		if tmpPtr.Offset(int64(tailEntry.headerSize + tailEntry.len)).GetUint8() != ZipEnd {
			zl.setZlTail(zl.zlTail() + uint32(nextDiff))
		}
	} else {
		zl.setZlTail(uint32(p.Addr() - zl.ptr.Addr()))
	}

	if nextDiff != 0 {
		offset = p.Addr() - zl.ptr.Addr()
		zl = zl.cascadeUpdate(p.Offset(int64(reqLen)))
		p = zl.ptr.Offset(int64(offset))
	}
	p.Move(int64(zipStorePrevEntryLength(p, e.prevRawLen)))
	p.Move(int64(zipStoreEntryEncoding(p, e.encoding, strLen)))
	if ZipIsStr(e.encoding) {
		cmem.MemCpy(p.Value(), unsafe.Pointer((*reflect.StringHeader)(unsafe.Pointer(&str)).Data), uint(strLen))
		fmt.Println(p.GetUint8())
		fmt.Println(p.Offset(2).GetUint8())
		tmp := zl.ptr.Offset(int64(offset))
		e.decode(tmp)
		fmt.Println(getValue(tmp))
		fmt.Println(*e)
	} else {
		zipSaveInteger(p, value, e.encoding)
	}
	zl.incrZlLen()
	return zl
}

func (list ZipList) resize(newLen uint32) ZipList{
	ptr := cmem.ReAlloc(list.ptr.Value(), uint(newLen))
	zl := ZipList{ptr:utils.MakePointer(ptr)}
	zl.setZlBytes(newLen)
	zl.setZlEnd()
	return zl
}

func (list ZipList) Insert(index int, value string) ZipList{
	p := list.index(index)
	if p == utils.EmptyPointer {
		p = list.entryEnd()
	}
	return list.insert(p, value)
}

func (list ZipList) Add(value string) ZipList {
	return list.Insert(ZipListTail, value)
}


func (list ZipList) Get(index int) (val interface{}) {
	panic("implement me")
}

func (list ZipList) Set(index int, val interface{}) {
	panic("implement me")
}

func (list ZipList) remove(p utils.Pointer, num uint32) ZipList{
	var i, deleteTotalLen, deleted uint32
	var offset uint
	var nextDiff int32 = 0
	first, tail := &ZLEntry{}, &ZLEntry{}
	zlBytes := list.zlBytes()
	first.decode(p)
	for i = 0; p.GetUint8() != ZipEnd && i < num; i++ {
		p.Move(int64(list.rawEntryLengthSafe(p)))
		deleted++
	}

	if p.Addr() < first.p.Addr() {
		panic("error")
	}

	deleteTotalLen = uint32(p.Addr() - first.p.Addr())
	if deleteTotalLen == 0 {
		return list
	}
	var setTail uint32
	if p.GetUint8() != ZipEnd {
		nextDiff = zipPrevLenByteDiff(p, first.prevRawLen)
		p.Move(-int64(nextDiff))
		if !(p.Addr() >= first.p.Addr() && p.Addr() < list.entryEnd().Addr()) {
			panic("error")
		}
		zipStorePrevEntryLength(p, first.prevRawLen)
		setTail = list.zlTail() - deleteTotalLen
		if !list.entrySafe(p, tail, true) {
			panic("error")
		}
		if p.Offset(int64(tail.headerSize+tail.len)).GetUint8() != ZipEnd {
			setTail += uint32(nextDiff)
		}

		var bytesToMove = uint(zlBytes - uint32(p.Addr()-list.ptr.Addr()) - 1)
		cmem.MemMove(first.p.Value(), p.Value(), bytesToMove)
	} else {
		setTail = uint32(first.p.Addr() - list.ptr.Addr()) - first.prevRawLen
	}

	offset = uint(first.p.Addr() - list.ptr.Addr())
	zlBytes -= deleteTotalLen - uint32(nextDiff)
	zl := list.resize(zlBytes)
	p = zl.ptr.Offset(int64(offset))
	zl.decrZlLen(deleted)
	if setTail > zlBytes - ZipListEndSize {
		panic("error")
	}
	zl.setZlTail(setTail)

	if nextDiff != 0 {
		zl = zl.cascadeUpdate(p)
	}
	return zl
}

func (list ZipList) Remove(p *utils.Pointer) ZipList {
	var offset = uint((*p).Addr() - list.ptr.Addr())
	zl := list.remove(*p, 1)
	*p = zl.ptr.Offset(int64(offset))
	return zl
}

func (list ZipList) Replace(p utils.Pointer, str string) ZipList{
	entry := &ZLEntry{}
	entry.decode(p)
	var reqLen uint32
	var encoding byte = 0
	var value int64 = 123456789
	strLen := uint32(len(str))
	if ZipTryEncoding(str, strLen, &value, &encoding) {
		reqLen = zipIntSize(encoding)
	} else {
		reqLen = strLen
	}

	reqLen += zipStoreEntryEncoding(utils.EmptyPointer, encoding, strLen)

	if reqLen == entry.lenSize + entry.len {
		p.Move(int64(entry.prevRawLenSize))
		p.Move(int64(zipStoreEntryEncoding(p, encoding, strLen)))

		if ZipIsStr(encoding) {
			cmem.MemCpy(p.Value(), unsafe.Pointer((*reflect.StringHeader)(unsafe.Pointer(&str)).Data), uint(strLen))
		} else {
			zipSaveInteger(p, value, encoding)
		}
		return list
	} else {
		zl := list.Remove(&p)
		zl = zl.insert(p, str)
		return zl
	}
}

func (list ZipList) Compare(p utils.Pointer, str string) bool{
	entry := &ZLEntry{}
	var sEncoding byte
	var zVal, sVal int64
	var strLen = uint32(len(str))

	if p.GetUint8() == ZipEnd {
		return false
	}

	entry.decode(p)

	if ZipIsStr(entry.encoding) {
		if entry.len == strLen {
			return cmem.MemCmp(p.Offset(int64(entry.headerSize)).Value(), unsafe.Pointer((* reflect.StringHeader)(unsafe.Pointer(&str)).Data), uint(strLen)) == 0
		}
	} else {
		if ZipTryEncoding(str, strLen, &sVal, &sEncoding) {
			zVal = zipLoadInteger(p.Offset(int64(entry.headerSize)), entry.encoding)
			return zVal == sVal
		}
	}
	return false
}

func (list ZipList) RemoveLast() (val interface{}) {
	panic("implement me")
}

func (list ZipList) RemoveAllByVal(val interface{}) int {
	panic("implement me")
}

func (list ZipList) RemoveByVal(val interface{}, count int) int {
	panic("implement me")
}

func (list ZipList) ReverseRemoveByVal(val interface{}, count int) int {
	panic("implement me")
}

func (list ZipList) ForEach(consumer func(int, interface{}) bool) {
	panic("implement me")
}

func (list ZipList) Contains(val interface{}) bool {
	panic("implement me")
}

func (list ZipList) Range(start int, stop int) []interface{} {
	panic("implement me")
}
