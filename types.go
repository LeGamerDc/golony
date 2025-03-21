package golony

import "unsafe"

type (
	// Index 引用，可以用于在Golony中查找元素，check用来验证引用是否失效
	// Index 可以被用户用于在其他任何地方保存元素的引用。
	Index[T any] struct {
		_             [0]*T // prevent wrong conversions
		check         uint32
		offset, group uint16
	}
	// EraseIndex 是去除了类型的引用，适用于需要在无类型存储的地方
	EraseIndex struct {
		check         uint32
		offset, group uint16
	}
	// FatIndex 是Index的扩展，包含Index和一个指针，一般是用于Get元素后局部操作
	// 不要长期保存 FatIndex，因为FatIndex中包含的pointer会随着操作而失效
	FatIndex[T any] struct {
		index   Index[T]
		pointer *element[T]
	}
	Golony[T any] struct {
		groups        []*group[T]
		freeGroupHead *group[T]
		recycle       *group[T]
		totalSize     uint32
		totalCapacity uint32
		groupSize     uint16
		zero          T
	}
	ProcessFunc[T any] func(index FatIndex[T]) (erase, stop bool)

	element[T any] struct {
		prev, next uint16
		check      uint32
		v          T
	}
	group[T any] struct {
		skips              []uint16
		elements           []element[T]
		prev, next         *group[T]
		freePrev, freeNext *group[T]
		size, capacity     uint16
		freeListHead       uint16
		groupIndex         uint16
	}
)

func (i Index[T]) Id() uint64 {
	return *(*uint64)(unsafe.Pointer(&i))
}

func (i Index[T]) Check() uint32 {
	return i.check
}

func (i Index[T]) Erase() EraseIndex {
	return EraseIndex{check: i.check, offset: i.offset, group: i.group}
}

func (i Index[T]) Eq(other Index[T]) bool {
	return i.Id() == other.Id()
}

func From[T any](i EraseIndex) Index[T] {
	return Index[T]{check: i.check, offset: i.offset, group: i.group}
}

func (f FatIndex[T]) Index() Index[T] {
	return f.index
}

func (f FatIndex[T]) Pointer() *T {
	return &f.pointer.v
}
