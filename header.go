package golony

import "unsafe"

type Index[T any] struct {
	_             [0]*T // prevent wrong conversions
	check         uint32
	offset, group uint16
}

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

type EraseIndex struct {
	check         uint32
	offset, group uint16
}

type FatIndex[T any] struct {
	index   Index[T]
	pointer *element[T]
}

func (f FatIndex[T]) Index() Index[T] {
	return f.index
}

func (f FatIndex[T]) Pointer() *T {
	return &f.pointer.v
}

type element[T any] struct {
	prev, next uint16
	check      uint32
	v          T
}

type group[T any] struct {
	skips              []uint16
	elements           []element[T]
	prev, next         *group[T]
	freePrev, freeNext *group[T]
	size, capacity     uint16
	freeListHead       uint16
	groupIndex         uint16
}

type Golony[T any] struct {
	groups        []*group[T]
	freeGroupHead *group[T]
	recycle       *group[T]
	totalSize     uint32
	totalCapacity uint32
	groupSize     uint16
	zero          T
}
