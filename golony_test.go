package golony

import (
	"math/rand"
	"testing"

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
