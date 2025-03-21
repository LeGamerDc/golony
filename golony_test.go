package golony

import (
	"fmt"
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

func TestGolony_FullDelete(t *testing.T) {
	g := New[int](40)

	// 第一轮：插入100个元素
	for i := 0; i < 200; i++ {
		fi := g.Insert(uint32(i))
		*fi.Pointer() = i
	}
	assert.Equal(t, 200, int(g.totalSize))
	assert.Equal(t, 200, int(g.totalCapacity)) // 应该是 6 组，每组20个

	// 删除所有元素
	g.Iterate(func(fi FatIndex[int]) (bool, bool) {
		return true, false // 删除每个元素
	})
	assert.Equal(t, 0, int(g.totalSize))
	assert.Equal(t, 200, int(g.totalCapacity)) // capacity 应该保持不变

	for i := 0; i < 5; i++ {
		fmt.Printf("%d ", g.groups[i].skips[0])
	}
	fmt.Println()

	// 第二轮：再次插入100个元素
	for i := 0; i < 200; i++ {
		fi := g.Insert(uint32(i))
		*fi.Pointer() = i
	}
	assert.Equal(t, 200, int(g.totalSize))
	assert.Equal(t, 200, int(g.totalCapacity)) // 应该复用之前的空间
}
