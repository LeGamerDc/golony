package index

import (
	"math"
	"unsafe"
)

const null = math.MaxUint16

func pAdd(p *uint16, n uint16) *uint16 {
	return (*uint16)(unsafe.Add(unsafe.Pointer(p), 2*n))
}

func (m *Map[T]) updateSkip(pSkip *uint16, pe *element[T]) {
	s := (*pSkip) - 1
	if s != 0 {
		// update start and end of skip block
		*pAdd(pSkip, s) = s
		*pAdd(pSkip, 1) = s
		m.freeGroupHead.freeListHead++
		if pe.next != null {

		}
	}
}
