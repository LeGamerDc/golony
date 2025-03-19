package golony

import (
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
	g.Iterate(func(fi FatIndex[int]) bool {
		assert.Equal(t, vi, *fi.Pointer())
		vi += 2
		return true
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
