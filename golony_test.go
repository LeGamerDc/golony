package golony

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGolony_Insert(t *testing.T) {
	g := New[int](20)
	for i := 0; i <= 100; i++ {
		fi := g.Insert(uint32(i))
		*fi.pointer = i

		idx := fi.Index()
		assert.Equal(t, uint32(i), idx.check)
		assert.Equal(t, i/20, int(idx.group), g.totalSize)
		assert.Equal(t, i%20, int(idx.offset), g.totalSize)
		assert.Equal(t, i+1, int(g.totalSize), g.totalSize)
		assert.Equal(t, (i/20+1)*20, int(g.totalCapacity), g.totalSize)
	}
}
