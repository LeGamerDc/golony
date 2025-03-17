package golony

func New[T any](groupSize uint16) *Golony[T] {
	return &Golony[T]{
		groups:    make([]*group[T], 16),
		groupSize: groupSize,
	}
}

func (m *Golony[T]) Insert(check uint32) (fi FatIndex[T]) {
	if m.freeGroupHead == nil { // no free group, create one
		m.newGroup()
	}
	pe := &m.freeGroupHead.elements[m.freeGroupHead.freeListHead]
	fi = FatIndex[T]{
		index: Index[T]{
			check:  check,
			offset: m.freeGroupHead.freeListHead,
			group:  m.freeGroupHead.groupIndex,
		},
		pointer: pe,
	}
	m.updateSkip(&m.freeGroupHead.skips[m.freeGroupHead.freeListHead], pe)
	pe.check = check
	return
}

func (m *Golony[T]) Erase(i Index[T]) bool {
	if fi, ok := m.Get(i); ok && m.erase(fi) {
		fi.pointer.check = 0
		fi.pointer.v = m.zero
		return true
	}
	return false
}

func (m *Golony[T]) EraseFat(fi FatIndex[T]) bool {
	if m.erase(fi) {
		fi.pointer.check = 0
		fi.pointer.v = m.zero
		return true
	}
	return false
}

func (m *Golony[T]) Get(i Index[T]) (fi FatIndex[T], ok bool) {
	if g := m.groups[i.group]; g != nil {
		if v := &g.elements[i.offset]; v.check == i.check {
			return FatIndex[T]{
				index:   i,
				pointer: v,
			}, true
		}
	}
	return FatIndex[T]{}, false
}
