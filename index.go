package index

import "unsafe"

type Index[T any] struct {
	_        [0]*T // prevent wrong conversions
	group, x uint16
	check    uint32
}

func (i Index[T]) Value() uint64 {
	return *(*uint64)(unsafe.Pointer(&i))
}

func (i Index[T]) Check() uint32 {
	return i.check
}

type FatIndex[T any] struct {
	index   Index[T]
	pointer *T
}

func (f FatIndex[T]) Index() Index[T] {
	return f.index
}

type element[T any] struct {
	prev, next uint16
	check      uint32
	v          T
}

type group[T any] struct {
	skips                []uint16
	elements             []T
	prev, next           *group[T]
	erasePrev, eraseNext *group[T]
	size, capacity       uint16
	freeListHead         uint16
}

type Map[T any] struct {
	groups        []*group[T]
	freeGroupHead *group[T]
	recycle       *group[T]
	totalSize     uint32
	totalCapacity uint32
	zero          T
}
