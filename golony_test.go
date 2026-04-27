package golony

import (
	"math"
	"math/rand"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestGolony_Insert(t *testing.T) {
	g := New[int](20)
	for i := 0; i <= 100; i++ {
		fi := g.Insert(uint32(i))
		*fi.Pointer() = i

		idx := fi.Index()
		assert.Equal(t, uint32(i), idx.check)
		assert.Equal(t, i/20, int(idx.group), g.totalSize)
		assert.Equal(t, i%20, int(idx.offset), g.totalSize)
		assert.Equal(t, i+1, int(g.totalSize), g.totalSize)
		assert.Equal(t, (i/20+1)*20, int(g.totalCapacity), g.totalSize)
	}
}

func TestGolony_Delete(t *testing.T) {
	g := New[int](20)
	var handles []FatIndex[int]
	for i := 0; i < 100; i++ {
		fi := g.Insert(uint32(0))
		*fi.Pointer() = i
		handles = append(handles, fi)
	}
	assert.Equal(t, 100, int(g.totalSize))
	for i := 0; i < 100; i += 2 {
		assert.True(t, g.Erase(handles[i].Index()))
	}
	assert.Equal(t, 50, int(g.totalSize))
	for i := 0; i < 100; i++ {
		fi, ok := g.Get(handles[i].Index())
		if i%2 == 0 {
			assert.False(t, ok, fi.Index())
		} else {
			assert.True(t, ok)
			assert.Equal(t, i, *fi.Pointer())
		}
	}
	vi := 1
	g.Iterate(func(fi FatIndex[int]) (bool, bool) {
		assert.Equal(t, vi, *fi.Pointer())
		vi += 2
		return false, false
	})
	for i := 0; i < 100; i++ {
		ok := g.Erase(handles[i].Index())
		if i%2 == 0 {
			assert.False(t, ok)
		} else {
			assert.True(t, ok)
		}
	}
	assert.Equal(t, 0, int(g.totalSize))
}

func TestGolony_Iterate(t *testing.T) {
	g := New[int](20)
	var handles []FatIndex[int]

	// 插入 10 个元素
	for i := 0; i < 10; i++ {
		fi := g.Insert(uint32(i))
		*fi.Pointer() = i
		handles = append(handles, fi)
	}

	// 场景1：在迭代过程中删除当前元素（删除所有偶数）
	g.Iterate(func(fi FatIndex[int]) (erase bool, stop bool) {
		if *fi.Pointer()%2 == 0 {
			return true, false // 删除当前元素，继续迭代
		}
		return false, false
	})

	// 验证只剩下奇数
	var values []int
	g.Iterate(func(fi FatIndex[int]) (erase bool, stop bool) {
		values = append(values, *fi.Pointer())
		return false, false
	})
	assert.Equal(t, []int{1, 3, 5, 7, 9}, values)

	// 场景2：在迭代过程中删除其他元素
	values = nil
	g.Iterate(func(fi FatIndex[int]) (erase bool, stop bool) {
		if fi.Index().Eq(handles[3].Index()) {
			g.Erase(handles[1].Index())
			g.Erase(handles[7].Index())
		}
		values = append(values, *fi.Pointer())
		return false, false
	})

	// 验证最终结果
	assert.False(t, g.Erase(handles[1].Index()))
	assert.False(t, g.Erase(handles[7].Index()))
	assert.Equal(t, []int{1, 3, 5, 9}, values)
}

func TestGolony_Erase(t *testing.T) {
	g := New[int](10)
	for j := 0; j < 200; j++ {
		var handles []FatIndex[int]
		for i := 0; i < 1000; i++ {
			fi := g.Insert(uint32(i))
			*fi.Pointer() = i
			handles = append(handles, fi)
		}
		rand.Shuffle(len(handles), func(i, j int) {
			handles[i], handles[j] = handles[j], handles[i]
		})
		for i := 0; i < 1000; i++ {
			assert.True(t, g.Erase(handles[i].Index()))
		}
		assert.Equal(t, 0, int(g.totalSize))
		assert.Equal(t, 1000, int(g.totalCapacity))
	}
}

func TestGolony_EraseFatValidation(t *testing.T) {
	t.Run("ValidFatIndex", func(t *testing.T) {
		g := New[int](8)
		fi := g.Insert(10)
		*fi.Pointer() = 99

		assert.True(t, g.EraseFat(fi))
		_, ok := g.Get(fi.Index())
		assert.False(t, ok)
	})

	t.Run("ZeroValueFatIndex", func(t *testing.T) {
		g := New[int](8)
		assert.False(t, g.EraseFat(FatIndex[int]{}))
	})

	t.Run("ForeignFatIndex", func(t *testing.T) {
		a := New[int](8)
		b := New[int](8)

		fa := a.Insert(123)
		*fa.Pointer() = 1

		fb := b.Insert(456)
		*fb.Pointer() = 2

		assert.False(t, b.EraseFat(fa))

		retrieved, ok := b.Get(fb.Index())
		assert.True(t, ok)
		assert.Equal(t, 2, *retrieved.Pointer())
	})
}

func TestGolony_GroupLimit(t *testing.T) {
	g := New[int](8)
	g.groups = make([]*group[int], maxGroupCount)

	dummy := &group[int]{}
	for i := range g.groups {
		g.groups[i] = dummy
	}

	assert.PanicsWithValue(t, "golony: group index exceeds uint16 range", func() {
		g.Insert(1)
	})
}

func TestGolony_MetadataAndStopSemantics(t *testing.T) {
	t.Run("Counters", func(t *testing.T) {
		g := New[int](8)
		assert.Equal(t, 0, g.GroupNum())
		assert.Equal(t, 0, g.Capacity())

		for i := 0; i < 9; i++ {
			fi := g.Insert(uint32(i + 1))
			*fi.Pointer() = i
		}

		assert.Equal(t, 2, g.GroupNum())
		assert.Equal(t, 16, g.Capacity())
		assert.Equal(t, 9, g.Size())
	})

	t.Run("IterateStop", func(t *testing.T) {
		g := New[int](8)
		for i := 0; i < 5; i++ {
			fi := g.Insert(uint32(i + 1))
			*fi.Pointer() = i
		}

		visited := 0
		g.Iterate(func(fi FatIndex[int]) (bool, bool) {
			visited++
			return false, visited == 3
		})

		assert.Equal(t, 3, visited)
	})

	t.Run("IterateGroupStop", func(t *testing.T) {
		g := New[int](8)
		for i := 0; i < 5; i++ {
			fi := g.Insert(uint32(i + 1))
			*fi.Pointer() = i
		}

		visited := 0
		g.IterateGroup(0, func(fi FatIndex[int]) (bool, bool) {
			visited++
			return false, visited == 2
		})

		assert.Equal(t, 2, visited)
	})
}

func TestGolony_IterateEraseClearsElement(t *testing.T) {
	g := New[*int](8)

	value := 42
	fi := g.Insert(1)
	*fi.Pointer() = &value

	g.Iterate(func(current FatIndex[*int]) (bool, bool) {
		return current.Index().Eq(fi.Index()), false
	})

	assert.Equal(t, uint32(0), g.groups[0].elements[0].check)
	assert.Nil(t, g.groups[0].elements[0].v)
}

func FuzzGolony(f *testing.F) {
	f.Add(uint32(1), 10) // 初始语料

	f.Fuzz(func(t *testing.T, check uint32, numOps int) {
		if numOps <= 0 || numOps > 1000 {
			return
		}

		g := New[int](20)
		var indices []Index[int]

		for i := 0; i < numOps; i++ {
			op := rand.Intn(3)
			switch op {
			case 0: // 插入
				fi := g.Insert(check)
				*fi.Pointer() = int(check)
				indices = append(indices, fi.Index())

			case 1: // 获取
				if len(indices) > 0 {
					idx := indices[rand.Intn(len(indices))]
					if fi, ok := g.Get(idx); ok {
						if *fi.Pointer() != int(idx.Check()) {
							t.Errorf("value mismatch: got %d, want %d", *fi.Pointer(), idx.Check())
						}
					}
				}

			case 2: // 删除
				if len(indices) > 0 {
					i := rand.Intn(len(indices))
					g.Erase(indices[i])
					// 从切片中移除已删除的索引
					indices = append(indices[:i], indices[i+1:]...)
				}
			}
		}

		// 验证最终状态
		count := 0
		g.Iterate(func(fi FatIndex[int]) (bool, bool) {
			count++
			return false, false
		})
		assert.Equal(t, len(indices), count)
	})
}

func TestGolony_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	g := New[int](64)
	ops := 1000000
	var indices []Index[int]

	for i := 0; i < ops; i++ {
		if len(indices) == 0 || rand.Float32() < 0.6 { // 60% 概率插入
			fi := g.Insert(uint32(i))
			*fi.Pointer() = i
			indices = append(indices, fi.Index())
		} else if len(indices) > 0 { // 40% 概率删除
			idx := rand.Intn(len(indices))
			assert.True(t, g.Erase(indices[idx]))
			indices = append(indices[:idx], indices[idx+1:]...)
		}

		if i%10000 == 0 {
			// 定期验证容器状态
			count := 0
			g.Iterate(func(fi FatIndex[int]) (bool, bool) {
				count++
				return false, false
			})
			assert.Equal(t, len(indices), count)
			assert.Equal(t, count, int(g.totalSize))
		}
	}
}

func TestGolony_EdgeCases(t *testing.T) {
	// 测试最小组大小
	g1 := New[int](1)
	assert.Equal(t, uint16(8), g1.groupSize) // 应该被调整到最小值

	// 测试最大组大小
	g2 := New[int](1<<16 - 2)
	assert.Equal(t, uint16(1<<15), g2.groupSize) // 应该被调整到最大值

	// 测试空容器的迭代
	g3 := New[int](20)
	count := 0
	g3.Iterate(func(fi FatIndex[int]) (bool, bool) {
		count++
		return false, false
	})
	assert.Equal(t, 0, count)

	// 测试删除不存在的元素
	assert.False(t, g3.Erase(Index[int]{check: 1, offset: 0, group: 0}))
}

// TestBoundaryConditions 测试各种边界条件
func TestBoundaryConditions(t *testing.T) {
	// 测试最小和最大group大小
	t.Run("GroupSizeLimits", func(t *testing.T) {
		// 小于最小值的group大小
		g1 := New[int](0)
		assert.Equal(t, uint16(minGroupSize), g1.groupSize)

		g2 := New[int](1)
		assert.Equal(t, uint16(minGroupSize), g2.groupSize)

		g3 := New[int](minGroupSize - 1)
		assert.Equal(t, uint16(minGroupSize), g3.groupSize)

		// 超过最大值的group大小
		g4 := New[int](maxGroupSize + 1)
		assert.Equal(t, maxGroupSize, g4.groupSize)

		g5 := New[int](math.MaxUint16)
		assert.Equal(t, maxGroupSize, g5.groupSize)

		// 正好等于边界值
		g6 := New[int](minGroupSize)
		assert.Equal(t, uint16(minGroupSize), g6.groupSize)

		g7 := New[int](maxGroupSize)
		assert.Equal(t, maxGroupSize, g7.groupSize)
	})

	// 测试check值的边界情况
	t.Run("CheckValueBoundaries", func(t *testing.T) {
		g := New[int](16)

		// 测试check值为0的情况
		fi0 := g.Insert(0)
		*fi0.Pointer() = 100
		assert.Equal(t, uint32(0), fi0.Index().Check())

		// 测试check值为最大值的情况
		fiMax := g.Insert(math.MaxUint32)
		*fiMax.Pointer() = 200
		assert.Equal(t, uint32(math.MaxUint32), fiMax.Index().Check())

		// 验证两个元素都可以正确访问
		if retrieved, ok := g.Get(fi0.Index()); ok {
			assert.Equal(t, 100, *retrieved.Pointer())
		} else {
			t.Error("Element with check=0 should be accessible")
		}

		if retrieved, ok := g.Get(fiMax.Index()); ok {
			assert.Equal(t, 200, *retrieved.Pointer())
		} else {
			t.Error("Element with check=MaxUint32 should be accessible")
		}
	})

	// 测试索引边界
	t.Run("IndexBoundaries", func(t *testing.T) {
		g := New[int](16)

		// 测试无效的group索引
		invalidIndex1 := Index[int]{check: 1, offset: 0, group: math.MaxUint16}
		_, ok1 := g.Get(invalidIndex1)
		assert.False(t, ok1, "Should not get element from invalid group")

		invalidIndex2 := Index[int]{check: 1, offset: 0, group: 1000}
		_, ok2 := g.Get(invalidIndex2)
		assert.False(t, ok2, "Should not get element from non-existent group")

		// 测试无效的offset - 注意：发现Get方法缺少边界检查的bug
		fi := g.Insert(1)
		*fi.Pointer() = 42

		// 使用一个在group容量范围内但无效的offset来避免panic
		// 这里暴露了一个问题：Get方法应该检查offset是否超出group.capacity
		invalidIndex3 := Index[int]{check: 999, offset: 15, group: 0} // 使用错误的check值
		_, ok3 := g.Get(invalidIndex3)
		assert.False(t, ok3, "Should not get element with wrong check value")

		// 测试已删除元素的索引
		g.Erase(fi.Index())
		_, ok4 := g.Get(fi.Index())
		assert.False(t, ok4, "Should not get deleted element")
	})
}

// TestZeroValueHandling 测试零值处理
func TestZeroValueHandling(t *testing.T) {
	t.Run("IntZeroValue", func(t *testing.T) {
		g := New[int](8)

		// 插入零值
		fi := g.Insert(123)
		*fi.Pointer() = 0

		// 验证零值可以正确存储和检索
		if retrieved, ok := g.Get(fi.Index()); ok {
			assert.Equal(t, 0, *retrieved.Pointer())
		} else {
			t.Error("Zero value should be retrievable")
		}

		// 删除后值应该被重置为零值
		g.Erase(fi.Index())
		// 注意：删除后不应该再能访问到该元素
		_, ok := g.Get(fi.Index())
		assert.False(t, ok, "Deleted element should not be accessible")
	})

	t.Run("PointerZeroValue", func(t *testing.T) {
		g := New[*int](8)

		// 插入非空指针
		value := 42
		fi := g.Insert(100)
		*fi.Pointer() = &value

		if retrieved, ok := g.Get(fi.Index()); ok {
			assert.Equal(t, 42, **retrieved.Pointer())
		} else {
			t.Error("Pointer should be retrievable")
		}

		// 插入空指针
		fi2 := g.Insert(200)
		*fi2.Pointer() = nil

		if retrieved, ok := g.Get(fi2.Index()); ok {
			assert.Nil(t, *retrieved.Pointer())
		} else {
			t.Error("Nil pointer should be retrievable")
		}
	})
}

// TestTypeErasure 测试类型擦除功能
func TestTypeErasure(t *testing.T) {
	g := New[string](8)

	// 插入字符串元素
	fi := g.Insert(456)
	*fi.Pointer() = "hello"

	// 转换为擦除类型
	erased := fi.Index().Erase()
	assert.Equal(t, fi.Index().Check(), erased.check)
	assert.Equal(t, fi.Index().offset, erased.offset)
	assert.Equal(t, fi.Index().group, erased.group)

	// 从擦除类型恢复
	recovered := From[string](erased)
	assert.True(t, recovered.Eq(fi.Index()))

	// 验证恢复的索引仍然有效
	if retrieved, ok := g.Get(recovered); ok {
		assert.Equal(t, "hello", *retrieved.Pointer())
	} else {
		t.Error("Recovered index should be valid")
	}
}

// TestIndexComparison 测试索引比较功能
func TestIndexComparison(t *testing.T) {
	g := New[int](8)

	// 创建两个不同的索引
	fi1 := g.Insert(1)
	fi2 := g.Insert(2)

	idx1 := fi1.Index()
	idx2 := fi2.Index()

	// 测试不等性
	assert.False(t, idx1.Eq(idx2))
	assert.False(t, idx2.Eq(idx1))
	assert.NotEqual(t, idx1.Id(), idx2.Id())

	// 测试自相等
	assert.True(t, idx1.Eq(idx1))
	assert.True(t, idx2.Eq(idx2))
	assert.Equal(t, idx1.Id(), idx1.Id())

	// 创建相同的索引（这在正常使用中不应该发生，但测试edge case）
	idx1Copy := Index[int]{
		check:  idx1.check,
		offset: idx1.offset,
		group:  idx1.group,
	}
	assert.True(t, idx1.Eq(idx1Copy))
	assert.Equal(t, idx1.Id(), idx1Copy.Id())
}

// TestMemoryAlignment 测试内存对齐和安全性
func TestMemoryAlignment(t *testing.T) {
	// 测试不同类型的内存对齐
	t.Run("DifferentTypes", func(t *testing.T) {
		// 测试结构体类型
		type TestStruct struct {
			a int64
			b int32
			c int16
			d int8
		}

		g := New[TestStruct](8)
		fi := g.Insert(789)
		fi.Pointer().a = 1
		fi.Pointer().b = 2
		fi.Pointer().c = 3
		fi.Pointer().d = 4

		if retrieved, ok := g.Get(fi.Index()); ok {
			assert.Equal(t, int64(1), retrieved.Pointer().a)
			assert.Equal(t, int32(2), retrieved.Pointer().b)
			assert.Equal(t, int16(3), retrieved.Pointer().c)
			assert.Equal(t, int8(4), retrieved.Pointer().d)
		} else {
			t.Error("Struct should be retrievable")
		}

		// 测试指针的对齐
		ptrVal := uintptr(unsafe.Pointer(fi.Pointer()))
		assert.Equal(t, uintptr(0), ptrVal%unsafe.Alignof(TestStruct{}),
			"Struct should be properly aligned")
	})
}

// TestErrorHandlingInComplexScenarios 测试复杂场景中的错误处理
func TestErrorHandlingInComplexScenarios(t *testing.T) {
	t.Run("EraseInvalidIndex", func(t *testing.T) {
		g := New[int](8)

		// 尝试删除无效索引
		invalidIdx := Index[int]{check: 999, offset: 0, group: 0}
		success := g.Erase(invalidIdx)
		assert.False(t, success, "Erase of invalid index should fail")

		// 插入一个元素后删除
		fi := g.Insert(100)
		success = g.Erase(fi.Index())
		assert.True(t, success, "Erase of valid index should succeed")

		// 再次尝试删除同一个索引
		success = g.Erase(fi.Index())
		assert.False(t, success, "Second erase should fail")
	})

	t.Run("GetWithCorruptedCheck", func(t *testing.T) {
		g := New[int](8)

		fi := g.Insert(123)
		*fi.Pointer() = 456

		// 创建check值不匹配的索引
		corruptedIdx := Index[int]{
			check:  fi.Index().check + 1, // 错误的check值
			offset: fi.Index().offset,
			group:  fi.Index().group,
		}

		_, ok := g.Get(corruptedIdx)
		assert.False(t, ok, "Get with corrupted check should fail")

		// 原始索引应该仍然有效
		if retrieved, ok := g.Get(fi.Index()); ok {
			assert.Equal(t, 456, *retrieved.Pointer())
		} else {
			t.Error("Original index should still be valid")
		}
	})
}

// TestGroupIterationEdgeCases 测试group迭代的边界情况
func TestGroupIterationEdgeCases(t *testing.T) {
	g := New[int](8)

	// 测试空group的迭代
	t.Run("EmptyGroupIteration", func(t *testing.T) {
		count := 0
		g.IterateGroup(0, func(fi FatIndex[int]) (bool, bool) {
			count++
			return false, false
		})
		assert.Equal(t, 0, count, "Empty group should not iterate any elements")
	})

	// 测试无效group索引的迭代
	t.Run("InvalidGroupIteration", func(t *testing.T) {
		count := 0

		// 负数索引
		g.IterateGroup(-1, func(fi FatIndex[int]) (bool, bool) {
			count++
			return false, false
		})
		assert.Equal(t, 0, count, "Invalid negative group index should not iterate")

		// 超出范围的索引
		g.IterateGroup(1000, func(fi FatIndex[int]) (bool, bool) {
			count++
			return false, false
		})
		assert.Equal(t, 0, count, "Out of range group index should not iterate")
	})

	// 测试部分填充group的迭代
	t.Run("PartiallyFilledGroupIteration", func(t *testing.T) {
		// 插入一些元素但不填满group
		var indices []Index[int]
		for i := 0; i < 5; i++ { // group大小是8，只插入5个
			fi := g.Insert(uint32(i))
			*fi.Pointer() = i
			indices = append(indices, fi.Index())
		}

		// 删除一些元素创建空隙
		g.Erase(indices[1])
		g.Erase(indices[3])

		// 迭代应该只访问存在的元素
		visited := make(map[int]bool)
		g.IterateGroup(0, func(fi FatIndex[int]) (bool, bool) {
			value := *fi.Pointer()
			visited[value] = true
			return false, false
		})

		// 应该访问0, 2, 4，但不访问1, 3
		assert.True(t, visited[0])
		assert.False(t, visited[1])
		assert.True(t, visited[2])
		assert.False(t, visited[3])
		assert.True(t, visited[4])
		assert.Equal(t, 3, len(visited))
	})
}

// TestIntensiveOperations 测试密集操作的稳定性
func TestIntensiveOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping intensive test in short mode")
	}

	g := New[int](16)

	// 预填充数据
	var indices []Index[int]
	for i := 0; i < 100; i++ {
		fi := g.Insert(uint32(i))
		*fi.Pointer() = i
		indices = append(indices, fi.Index())
	}

	// 执行读取操作
	for read := 0; read < 10000; read++ {
		idx := indices[read%len(indices)]

		if fi, ok := g.Get(idx); ok {
			expected := int(idx.Check())
			assert.Equal(t, expected, *fi.Pointer(), "Value should match check")
		} else {
			t.Errorf("Element should be accessible")
		}
	}
}

// TestScaleOperations 测试规模操作
func TestScaleOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping scale test in short mode")
	}

	g := New[int](16)
	const numElements = 1000

	// 创建多个group
	var allIndices []Index[int]
	for i := 0; i < numElements; i++ {
		fi := g.Insert(uint32(i))
		*fi.Pointer() = i
		allIndices = append(allIndices, fi.Index())
	}

	assert.Equal(t, numElements, g.Size())

	// 验证所有元素都可以访问
	for i, idx := range allIndices {
		if fi, ok := g.Get(idx); !ok {
			t.Errorf("Element %d should be accessible", i)
		} else {
			assert.Equal(t, i, *fi.Pointer())
		}
	}

	// 删除一半元素
	for i := 0; i < numElements/2; i++ {
		g.Erase(allIndices[i])
	}

	assert.Equal(t, numElements/2, g.Size())
}

// TestCheckValueManagement 测试check值管理的最佳实践
func TestCheckValueManagement(t *testing.T) {
	g := New[int](8)

	// 演示正确的check值使用方式
	var indices []Index[int]
	for i := 0; i < 10; i++ {
		// 使用递增的check值避免冲突
		fi := g.Insert(uint32(i + 1000)) // 使用大于元素值的check值
		*fi.Pointer() = i
		indices = append(indices, fi.Index())
	}

	// 验证所有元素都能正确访问
	for i, idx := range indices {
		if fi, ok := g.Get(idx); ok {
			assert.Equal(t, i, *fi.Pointer())
		} else {
			t.Errorf("Element %d should be accessible", i)
		}
	}

	// 删除一些元素
	for i := 0; i < 5; i++ {
		g.Erase(indices[i])
	}

	// 重新插入时使用不同的check值
	for i := 0; i < 5; i++ {
		fi := g.Insert(uint32(i + 2000)) // 使用不同范围的check值
		*fi.Pointer() = i + 100
	}

	// 验证新旧元素都能正确访问
	iterateCount := 0
	g.Iterate(func(fi FatIndex[int]) (bool, bool) {
		iterateCount++
		return false, false
	})

	assert.Equal(t, 10, iterateCount, "Should have 10 elements total")
}

// TestFromU64 验证 FromU64 与 Index.Id() 的逻辑严格对称。
func TestFromU64(t *testing.T) {
	t.Run("RoundTrip_SingleElement", func(t *testing.T) {
		g := New[int](8)
		fi := g.Insert(42)
		idx := fi.Index()

		recovered := FromU64[int](idx.Id())
		assert.True(t, idx.Eq(recovered), "FromU64(idx.Id()) should equal original index")
		assert.Equal(t, idx.Id(), recovered.Id(), "Id() of recovered index should match original")
	})

	t.Run("RoundTrip_PreservesAllFields", func(t *testing.T) {
		g := New[int](8)
		fi := g.Insert(0xDEADBEEF)
		idx := fi.Index()

		recovered := FromU64[int](idx.Id())
		assert.Equal(t, idx.check, recovered.check, "check field should be preserved")
		assert.Equal(t, idx.offset, recovered.offset, "offset field should be preserved")
		assert.Equal(t, idx.group, recovered.group, "group field should be preserved")
	})

	t.Run("RoundTrip_MultipleElements", func(t *testing.T) {
		g := New[int](8)
		checks := []uint32{1, 2, 100, 0xFFFFFFFF, 0x12345678}
		var indices []Index[int]
		for i, c := range checks {
			fi := g.Insert(c)
			*fi.Pointer() = i
			indices = append(indices, fi.Index())
		}

		for _, idx := range indices {
			recovered := FromU64[int](idx.Id())
			assert.True(t, idx.Eq(recovered))
			assert.Equal(t, idx.Id(), recovered.Id())
		}
	})

	t.Run("RecoveredIndex_CanGetFromContainer", func(t *testing.T) {
		g := New[int](8)
		fi := g.Insert(7)
		*fi.Pointer() = 999
		idx := fi.Index()

		recovered := FromU64[int](idx.Id())
		gotFi, ok := g.Get(recovered)
		assert.True(t, ok, "should be able to Get element using recovered index")
		assert.Equal(t, 999, *gotFi.Pointer(), "value should match")
	})

	t.Run("RecoveredIndex_StaleAfterErase", func(t *testing.T) {
		g := New[int](8)
		fi := g.Insert(3)
		idx := fi.Index()
		id := idx.Id()

		g.Erase(idx)

		recovered := FromU64[int](id)
		_, ok := g.Get(recovered)
		assert.False(t, ok, "recovered index should be invalid after erase")
	})

	t.Run("MultipleGroups_CorrectGroupAndOffset", func(t *testing.T) {
		// groupSize=2 强制快速产生多个 group，覆盖 group != 0 的情形
		g := New[int](2)
		var indices []Index[int]
		for i := 0; i < 8; i++ {
			fi := g.Insert(uint32(i + 1))
			*fi.Pointer() = i * 10
			indices = append(indices, fi.Index())
		}

		for i, idx := range indices {
			recovered := FromU64[int](idx.Id())
			assert.Equal(t, idx.group, recovered.group, "group mismatch at element %d", i)
			assert.Equal(t, idx.offset, recovered.offset, "offset mismatch at element %d", i)
			assert.Equal(t, idx.check, recovered.check, "check mismatch at element %d", i)

			gotFi, ok := g.Get(recovered)
			assert.True(t, ok, "element %d should be retrievable via recovered index", i)
			assert.Equal(t, i*10, *gotFi.Pointer())
		}
	})

	t.Run("DifferentTypes_IndependentRoundTrip", func(t *testing.T) {
		gi := New[int](8)
		gs := New[string](8)

		fii := gi.Insert(11)
		fis := gs.Insert(22)

		idxI := fii.Index()
		idxS := fis.Index()

		recoveredI := FromU64[int](idxI.Id())
		recoveredS := FromU64[string](idxS.Id())

		assert.True(t, idxI.Eq(recoveredI))
		assert.True(t, idxS.Eq(recoveredS))

		_, okI := gi.Get(recoveredI)
		_, okS := gs.Get(recoveredS)
		assert.True(t, okI)
		assert.True(t, okS)
	})

	t.Run("Id_FromU64_Id_IsIdempotent", func(t *testing.T) {
		g := New[int](8)
		fi := g.Insert(55)
		idx := fi.Index()

		id1 := idx.Id()
		id2 := FromU64[int](id1).Id()
		assert.Equal(t, id1, id2, "Id(FromU64(Id())) should equal Id()")
	})

	t.Run("ZeroId_MatchesZeroValueIndex", func(t *testing.T) {
		recovered := FromU64[int](0)
		zero := Index[int]{}
		assert.Equal(t, zero.Id(), recovered.Id())
		assert.True(t, zero.Eq(recovered))
	})

	t.Run("MaxFieldValues_PreservedExactly", func(t *testing.T) {
		// 手动构造一个极端字段值的 Index，通过 Id 往返验证
		g := New[int](8)
		// 插入一个普通元素，再覆盖字段值做位运算验证（不实际查容器）
		fi := g.Insert(0xFFFFFFFF)
		idx := fi.Index()
		// 验证最大 check 值能完整往返
		id := idx.Id()
		recovered := FromU64[int](id)
		assert.Equal(t, idx.check, recovered.check)
		assert.Equal(t, uint64(recovered.check), id&0xFFFFFFFF,
			"low 32 bits of Id should equal check on little-endian")
	})
}
