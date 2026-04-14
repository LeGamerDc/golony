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

// Insert 插入一个元素，返回元素的索引。
// check 由调用方提供，必须保证不会与仍可能被持有的旧 Index 重复。
//
// 警告：不应以 check=0 插入元素。Index[T]{} 是 Go 的零值，其 check/group/offset
// 均为 0，若 slot(group=0, offset=0) 中存在 check=0 的存活元素，零值 Index 将
// 意外地通过 Get 的合法性校验并解析到该元素，导致未初始化的句柄误操作真实数据。
// 推荐使用从 1 开始的单调递增值作为 check。
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
	if fi, ok := m.Get(i); ok {
		return m.eraseAndClear(fi)
	}
	return false
}

// EraseFat 使用FatIndex删除一个元素，返回是否删除成功
func (m *Golony[T]) EraseFat(fi FatIndex[T]) bool {
	if !m.validFatIndex(fi) {
		return false
	}
	return m.eraseAndClear(fi)
}

// Get 获取一个元素，返回FatIndex已经判断Index是否失效
func (m *Golony[T]) Get(i Index[T]) (fi FatIndex[T], ok bool) {
	if int(i.group) >= len(m.groups) {
		return FatIndex[T]{}, false
	}
	if g := m.groups[i.group]; g != nil {
		// 添加边界检查，防止数组越界
		if i.offset >= g.capacity {
			return FatIndex[T]{}, false
		}
		if v := &g.elements[i.offset]; g.skips[i.offset] == 0 && v.check == i.check {
			return FatIndex[T]{
				index:   i,
				pointer: v,
			}, true
		}
	}
	return FatIndex[T]{}, false
}

// Iterate 遍历所有元素直到 process 返回 stop，途中 process 返回 erase 会导致当前元素被删除。
// 在回调中 Erase 非当前元素是允许的；删除当前迭代元素只能通过返回 erase=true，
// 而不应直接在回调中对当前元素的 Index 调用 Erase。
//
// Insert 语义（未定义行为）：在 process 回调中调用 Insert 不会损坏数据结构——
// skipfield 与 free list 的结构完整性在所有情形下均得到维持。
// 但新插入元素是否会在本次遍历中被访问到是未定义的：
//   - 若新元素被分配到当前 group 中当前迭代位置之后的空槽，它会被本次遍历访问到；
//   - 若被分配到当前位置之前的空槽，或触发了新 group 的分配，则不会被访问到。
//
// 具体行为取决于容器内部的碎片状态与分配路径，不应在代码中依赖此行为。
// 如需在遍历中产生新元素，推荐将待插入数据收集到临时切片，在 Iterate 返回后再批量 Insert。
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
					m.eraseAndClear(i)
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
// 用户可以用这个接口来分批次遍历。
// Insert 期间的语义约束与 Iterate 相同，详见 Iterate 的注释。
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
			m.eraseAndClear(i)
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

// Size 返回当前存活元素数量。
//
// 注意：在 32 位平台上，当容量接近最大值（65536 × 32768 = 2,147,483,648）时，
// int 转换结果可能溢出为负数，详见 Capacity 的说明。
func (m *Golony[T]) Size() int {
	return int(m.totalSize)
}

// Capacity 返回当前已分配的总槽位数（含存活与已删除）。
//
// 注意：Golony 的最大理论容量为 65536 × 32768 = 2,147,483,648，恰好超过
// math.MaxInt32。在 32 位平台上若容量达到上限，此处的 int 转换结果将溢出为负数。
// 若需在 32 位平台支持极大容量，请限制 group 数量使总容量不超过 math.MaxInt32，
// 或在使用返回值前自行做范围检查。
func (m *Golony[T]) Capacity() int {
	return int(m.totalCapacity)
}
