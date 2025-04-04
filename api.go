package golony

const (
	maxGroupSize uint16 = 1 << 15
	minGroupSize        = 8
)

func New[T any](groupSize uint16) *Golony[T] {
	groupSize = min(maxGroupSize, max(minGroupSize, groupSize))
	return &Golony[T]{
		groups:    make([]*group[T], 0, 16),
		groupSize: groupSize,
	}
}

// Insert 插入一个元素，返回元素的索引
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
	m.updateSkip(m.freeGroupHead, m.freeGroupHead.freeListHead)
	pe.check = check
	return
}

// Erase 使用Index删除一个元素，返回是否删除成功
func (m *Golony[T]) Erase(i Index[T]) bool {
	if fi, ok := m.Get(i); ok && m.erase(fi) {
		fi.pointer.check = 0
		fi.pointer.v = m.zero
		return true
	}
	return false
}

// EraseFat 使用FatIndex删除一个元素，返回是否删除成功
func (m *Golony[T]) EraseFat(fi FatIndex[T]) bool {
	if m.erase(fi) {
		fi.pointer.check = 0
		fi.pointer.v = m.zero
		return true
	}
	return false
}

// Get 获取一个元素，返回FatIndex已经判断Index是否失效
func (m *Golony[T]) Get(i Index[T]) (fi FatIndex[T], ok bool) {
	if i.group >= uint16(len(m.groups)) {
		return FatIndex[T]{}, false
	}
	if g := m.groups[i.group]; g != nil {
		// TODO skips test can be avoided
		if v := &g.elements[i.offset]; g.skips[i.offset] == 0 && v.check == i.check {
			return FatIndex[T]{
				index:   i,
				pointer: v,
			}, true
		}
	}
	return FatIndex[T]{}, false
}

// Iterate 遍历所有元素直到process返回stop，途中process返回erase会导致当前元素被删除。
// 用户可以在process中调用 Erase 删除其他元素，但删除当前迭代元素只能通过返回 erase。
func (m *Golony[T]) Iterate(process ProcessFunc[T]) {
	var (
		i, ni FatIndex[T]
		ok    bool
	)
	for gi, g := range m.groups {
		if g != nil {
			for i, ok = m.begin(g, uint16(gi)); ok; {
				e, s := process(i)
				ni, ok = m.advance(g, i)
				if e {
					m.erase(i)
				}
				if s {
					return
				}
				i = ni
			}
		}
	}
}

// IterateGroup 遍历一个group中的所有元素直到process返回stop，途中process返回erase会导致当前元素被删除。
// 用户可以用这个接口来分批次遍历
func (m *Golony[T]) IterateGroup(idx int, process ProcessFunc[T]) {
	if idx < 0 || idx >= len(m.groups) || m.groups[idx] == nil {
		return
	}
	var (
		i, ni FatIndex[T]
		g     = m.groups[idx]
		ok    bool
	)

	for i, ok = m.begin(g, uint16(idx)); ok; {
		e, s := process(i)
		ni, ok = m.advance(g, i)
		if e {
			m.erase(i)
		}
		if s {
			return
		}
		i = ni
	}
}

func (m *Golony[T]) GroupNum() int {
	return len(m.groups)
}

func (m *Golony[T]) Size() int {
	return int(m.totalSize)
}

func (m *Golony[T]) Capacity() int {
	return int(m.totalCapacity)
}
