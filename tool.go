package golony

import (
	"math"
	"unsafe"
)

const null = math.MaxUint16

func pAdd[T any](p *T, n uint16, s int) *T {
	return (*T)(unsafe.Add(unsafe.Pointer(p), int(n)*s))
}

func (m *Gololy[T]) updateSkip(pSkip *uint16, pe *element[T]) {
	m.freeGroupHead.size++
	m.totalSize++

	s := (*pSkip) - 1
	if s != 0 { // case 1. skip block node len > 1
		*pAdd(pSkip, s, 2) = s
		*pAdd(pSkip, 1, 2) = s
		m.freeGroupHead.freeListHead++
		if pe.next != null {
			pAdd(pe, pe.next, m.elemSize).prev = m.freeGroupHead.freeListHead
		}
		pAdd(pe, 1, m.elemSize).next = pe.next
	} else { // case 2. skip block 1 node, remove skip block
		m.freeGroupHead.freeListHead = pe.next
		if pe.next != null {
			pAdd(pe, pe.next, m.elemSize).prev = null
		} else {
			m.freeGroupHead = m.freeGroupHead.eraseNext
		}
	}
	*pSkip = 0
}
