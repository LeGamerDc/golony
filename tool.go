package golony

import (
	"math"
	"unsafe"
)

const null = math.MaxUint16

func pAdd[T any](p *T, n uint16) *T {
	var (
		zero T
		s    = int(unsafe.Sizeof(zero))
	)
	return (*T)(unsafe.Add(unsafe.Pointer(p), int(n)*s))
}

func (m *Golony[T]) updateSkip(pSkip *uint16, pe *element[T]) {
	m.freeGroupHead.size++
	m.totalSize++

	s := (*pSkip) - 1
	if s != 0 { // case 1. skip block node len > 1
		*pAdd(pSkip, s) = s
		*pAdd(pSkip, 1) = s
		m.freeGroupHead.freeListHead++
		if pe.next != null {
			pAdd(pe, pe.next).prev = m.freeGroupHead.freeListHead
		}
		pAdd(pe, 1).next = pe.next
	} else { // case 2. skip block 1 node, remove skip block
		m.freeGroupHead.freeListHead = pe.next
		if pe.next != null {
			pAdd(pe, pe.next).prev = null
		} else {
			m.freeGroupHead = m.freeGroupHead.freeNext
		}
	}
	*pSkip = 0
}

func (g *group[T]) reset(zero bool) {
	g.size = 0
	g.prev, g.next = nil, nil
	g.freePrev, g.freeNext = nil, nil
	if zero {
		clear(g.elements)
		clear(g.skips)
	}
	g.freeListHead = 0
	g.skips[0] = g.capacity
	g.skips[g.capacity-1] = g.capacity
	g.elements[0].prev = null
	g.elements[0].next = null
}

func (m *Golony[T]) newGroup() {
	var (
		g, prev, next *group[T]
		idx           = null
	)
	g, m.recycle = m.recycle, g
	if g == nil {
		g = &group[T]{
			skips:        make([]uint16, m.groupSize+1),
			elements:     make([]element[T], m.groupSize),
			size:         0,
			capacity:     m.groupSize,
			freeListHead: 0,
		}
		g.reset(false)
	}

	for i, cg := range m.groups {
		if cg != nil {
			prev = cg
			continue
		}
		idx = i
		break
	}
	// step 1. place group
	if idx == -1 {
		idx = len(m.groups)
		m.groups = append(m.groups, nil)
	}
	m.groups[idx] = g
	g.groupIndex = uint16(idx)
	// step 2. chain group in list
	if prev != nil {
		next = prev.next
		g.prev = prev
		prev.next = g
	}
	if next != nil {
		g.next = next
		next.prev = g
	}
	// step 3. chain group in free list
	m.freeGroupHead = g
	// step 4. update g
	m.totalCapacity += uint32(m.groupSize)
}

func (m *Golony[T]) erase(c FatIndex[T]) (n FatIndex[T], ok bool) {
	if c.index.check != c.pointer.check {
		return
	}
	g := m.groups[c.index.group]
	if g == nil {
		return
	}
	m.totalSize--
	g.size--
	if g.size != 0 { // not empty after erase
		before := c.index.offset > 0 && g.skips[c.index.offset-1] != 0
		after := g.skips[c.index.offset+1] != 0 // no boundary check due to skips array has 1 extra position
		if !(before || after) {                 // case 1. no need merge skip blocks
			g.skips[c.index.offset] = 1
			if g.freeListHead != null {
				g.elements[g.freeListHead].prev = c.index.offset
			} else {
				g.freeNext = m.freeGroupHead
				if m.freeGroupHead != nil {
					m.freeGroupHead.freePrev = g
				}
				m.freeGroupHead = g
			}
			g.elements[c.index.offset].next = g.freeListHead
			g.freeListHead = c.index.offset
		} else if before && !after { // case 2. merge skip block with prev
			v := g.skips[c.index.offset-1] + 1
			g.skips[c.index.offset] = v
			g.skips[c.index.offset-v+1] = v
		} else if !before { // case 3. merge skip block with next
			v := g.skips[c.index.offset+1] + 1
			g.skips[c.index.offset] = v
			g.skips[c.index.offset+v-1] = v
			pe, ne := g.elements[c.index.offset], g.elements[c.index.offset+1]
			pe.prev = ne.prev
			pe.next = ne.next
			if ne.next != null {
				g.elements[ne.next].prev = c.index.offset
			}
			if ne.prev != null {
				g.elements[ne.prev].next = c.index.offset
			} else {
				g.freeListHead = c.index.offset
			}
		} else { // case 4. merge skip block with prev and next
			g.skips[c.index.offset] = 1 // ensure all skip for erased element is > 0
			// TODO
		}
	}
}
