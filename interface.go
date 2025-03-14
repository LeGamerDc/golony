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
	return false
}

func (m *Golony[T]) Get(i Index[T]) (fi FatIndex[T], ok bool) {
	return FatIndex[T]{}, false
}
