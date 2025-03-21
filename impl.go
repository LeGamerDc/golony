package golony

import (
	"math"
)

const null = math.MaxUint16

func (m *Golony[T]) updateSkip(g *group[T], offset uint16) {
	pSkip := &g.skips[offset]
	pe := &g.elements[offset]
	g.size++
	m.totalSize++

	s := (*pSkip) - 1
	if s != 0 { // case 1. skip block node len > 1
		g.skips[offset+s] = s
		g.skips[offset+1] = s
		g.freeListHead++
		if pe.next != null {
			g.elements[pe.next].prev = g.freeListHead
		}
		pn := &g.elements[offset+1]
		pn.prev, pn.next = null, pe.next
	} else { // case 2. skip block 1 node, remove skip block
		g.freeListHead = pe.next
		if pe.next != null {
			g.elements[pe.next].prev = null
		} else {
			m.freeGroupHead = g.freeNext
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
	for i := uint16(1); i < g.capacity-1; i++ {
		g.skips[i] = 1
	}
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
	if idx == null {
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

func (m *Golony[T]) begin(g *group[T], gi uint16) (fi FatIndex[T], ok bool) {
	if idx := g.skips[0]; idx < g.capacity {
		return FatIndex[T]{
			index:   Index[T]{group: gi, offset: idx, check: g.elements[idx].check},
			pointer: &g.elements[idx],
		}, true
	}
	return
}

func (m *Golony[T]) advance(g *group[T], c FatIndex[T]) (fi FatIndex[T], ok bool) {
	idx := c.index.offset + 1
	if idx = idx + g.skips[idx]; idx < g.capacity {
		return FatIndex[T]{
			index:   Index[T]{group: c.index.group, offset: idx, check: g.elements[idx].check},
			pointer: &g.elements[idx],
		}, true
	}
	return
}

func (m *Golony[T]) erase(c FatIndex[T]) (ok bool) {
	if c.index.check != c.pointer.check {
		return
	}
	g := m.groups[c.index.group]
	if g == nil || g.skips[c.index.offset] != 0 {
		return
	}
	m.totalSize--
	g.size--
	// if g.size != 0 { // not empty after erase
	before := c.index.offset > 0 && g.skips[c.index.offset-1] != 0
	after := g.skips[c.index.offset+1] != 0 // no boundary check due to skips array has 1 extra position
	if !(before || after) {                 // case 1. no need merge skip blocks
		g.skips[c.index.offset] = 1
		if g.freeListHead != null {
			g.elements[g.freeListHead].prev = c.index.offset
		} else { // first slot
			g.freeNext = m.freeGroupHead
			if m.freeGroupHead != nil {
				m.freeGroupHead.freePrev = g
			}
			m.freeGroupHead = g
		}
		g.elements[c.index.offset].next = g.freeListHead
		g.elements[c.index.offset].prev = null
		g.freeListHead = c.index.offset
	} else if before && !after { // case 2. merge skip block with prev
		v := g.skips[c.index.offset-1] + 1
		g.skips[c.index.offset] = v
		g.skips[c.index.offset-v+1] = v
	} else if !before { // case 3. merge skip block with next
		v := g.skips[c.index.offset+1] + 1
		g.skips[c.index.offset] = v
		g.skips[c.index.offset+v-1] = v
		pe, ne := &g.elements[c.index.offset], &g.elements[c.index.offset+1]
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
		pv := g.skips[c.index.offset-1]
		nv := g.skips[c.index.offset+1]
		g.skips[c.index.offset-pv] = pv + nv + 1
		g.skips[c.index.offset+nv] = pv + nv + 1
		ne := &g.elements[c.index.offset+1]
		if ne.next != null {
			g.elements[ne.next].prev = ne.prev
		}
		if ne.prev != null {
			g.elements[ne.prev].next = ne.next
		} else {
			g.freeListHead = ne.next
		}
	}
	// }
	return true
}
