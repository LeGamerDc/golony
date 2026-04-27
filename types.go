package golony

import "unsafe"

type (
	// Index 引用，可以用于在Golony中查找元素，check用来验证引用是否失效
	// Index 可以被用户用于在其他任何地方保存元素的引用。
	// check 由调用方维护，必须保证不会和仍可能被持有的旧引用重复。
	//
	// 零值陷阱：Index[T]{} 的零值为 (check=0, group=0, offset=0)。
	// 若 slot (group=0, offset=0) 中恰好存有一个 check=0 的存活元素，
	// Get(Index[T]{}) 会成功返回该元素，造成意外别名访问。
	// 强烈建议调用方永远不以 check=0 调用 Insert，从而使零值 Index 始终表示无效引用。
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
	// FatIndex 是Index的扩展，包含Index和一个指针，一般是用于Get元素后局部操作。
	// 不要长期保存 FatIndex，也不要将 Pointer() 返回的 *T 存储到 FatIndex 的使用作用域之外，
	// 因为 Golony 不追踪原始指针的生命周期，长期持有会绕过 check/generation 校验机制。
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
		groupSize     uint16 // group index is uint16, so at most 65536 groups can exist
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

func FromU64[T any](id uint64) Index[T] {
	return *(*Index[T])(unsafe.Pointer(&id))
}

func (f FatIndex[T]) Index() Index[T] {
	return f.index
}

// Pointer 返回指向元素值的原始指针。
//
// 指针稳定性：Golony 的元素在容器生命周期内不会被移动（无 reallocation），
// 因此只要该元素未被 Erase，此指针始终指向有效内存。
//
// ABA 危险：此指针完全绕过 check/generation 校验机制。
// 若持有者在元素被 Erase 后（该 slot 可能已被新元素复用）仍通过此指针写入，
// 则会静默修改新元素的值，且 Index 的 check 机制无法感知此类错误。
// 不应将此指针持有超出当前元素的生命周期，亦不应在调用 Erase 之后继续使用。
func (f FatIndex[T]) Pointer() *T {
	return &f.pointer.v
}
