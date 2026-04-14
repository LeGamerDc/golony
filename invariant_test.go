package golony

import (
	"fmt"
	"math/rand"
	"testing"
)

func validateGroupState[T any](t *testing.T, m *Golony[T], groupIndex int, g *group[T]) {
	t.Helper()

	var liveCount uint16
	for offset := uint16(0); offset < g.capacity; offset++ {
		if g.skips[offset] == 0 {
			liveCount++
			continue
		}
		if offset > 0 && g.skips[offset-1] != 0 {
			continue
		}
		blockLen := g.skips[offset]
		if blockLen == 0 {
			t.Fatalf("group %d offset %d: free block len is 0", groupIndex, offset)
		}
		blockEnd := offset + blockLen - 1
		if blockEnd >= g.capacity {
			t.Fatalf("group %d offset %d: free block end %d out of range (capacity=%d)", groupIndex, offset, blockEnd, g.capacity)
		}
		if g.skips[blockEnd] != blockLen {
			t.Fatalf("group %d block [%d,%d]: tail marker=%d want=%d", groupIndex, offset, blockEnd, g.skips[blockEnd], blockLen)
		}
	}

	if liveCount != g.size {
		t.Fatalf("group %d: liveCount=%d size=%d", groupIndex, liveCount, g.size)
	}

	seenHeads := map[uint16]struct{}{}
	for head := g.freeListHead; head != null; head = g.elements[head].next {
		if _, ok := seenHeads[head]; ok {
			t.Fatalf("group %d: free list cycle at %d", groupIndex, head)
		}
		seenHeads[head] = struct{}{}
		if head >= g.capacity {
			t.Fatalf("group %d: free list head %d out of range", groupIndex, head)
		}
		if g.skips[head] == 0 {
			t.Fatalf("group %d: free list head %d points to live slot", groupIndex, head)
		}
		if head > 0 && g.skips[head-1] != 0 {
			t.Fatalf("group %d: free list head %d is not a block head", groupIndex, head)
		}
		next := g.elements[head].next
		if next != null && g.elements[next].prev != head {
			t.Fatalf("group %d: next.prev mismatch for %d -> %d", groupIndex, head, next)
		}
		prev := g.elements[head].prev
		if prev == null {
			if head != g.freeListHead {
				t.Fatalf("group %d: non-head node %d has null prev", groupIndex, head)
			}
		} else if g.elements[prev].next != head {
			t.Fatalf("group %d: prev.next mismatch for %d <- %d", groupIndex, head, prev)
		}
	}

	for offset := uint16(0); offset < g.capacity; offset++ {
		if g.skips[offset] == 0 {
			continue
		}
		if offset > 0 && g.skips[offset-1] != 0 {
			continue
		}
		if _, ok := seenHeads[offset]; !ok {
			t.Fatalf("group %d: block head %d missing from free list", groupIndex, offset)
		}
	}
}

func validateContainerState[T comparable](t *testing.T, m *Golony[T], want map[Index[T]]T) {
	t.Helper()

	var totalLive uint32
	nonFullGroups := map[*group[T]]struct{}{}
	for i, g := range m.groups {
		if g == nil {
			continue
		}
		validateGroupState(t, m, i, g)
		totalLive += uint32(g.size)
		if g.size < g.capacity {
			nonFullGroups[g] = struct{}{}
		}
	}
	if totalLive != m.totalSize {
		t.Fatalf("total live=%d totalSize=%d", totalLive, m.totalSize)
	}

	seenFreeGroups := map[*group[T]]struct{}{}
	for g := m.freeGroupHead; g != nil; g = g.freeNext {
		if _, ok := seenFreeGroups[g]; ok {
			t.Fatal("freeGroupHead chain contains a cycle")
		}
		seenFreeGroups[g] = struct{}{}
		if g.size == g.capacity {
			t.Fatal("freeGroupHead chain contains a full group")
		}
	}
	if len(seenFreeGroups) != len(nonFullGroups) {
		t.Fatalf("free group count=%d want=%d", len(seenFreeGroups), len(nonFullGroups))
	}
	for g := range nonFullGroups {
		if _, ok := seenFreeGroups[g]; !ok {
			t.Fatal("non-full group missing from freeGroupHead chain")
		}
	}

	got := make(map[Index[T]]T, len(want))
	m.Iterate(func(fi FatIndex[T]) (erase, stop bool) {
		got[fi.Index()] = *fi.Pointer()
		return false, false
	})
	if len(got) != len(want) {
		t.Fatalf("iterate count=%d want=%d", len(got), len(want))
	}
	for idx, wantValue := range want {
		fi, ok := m.Get(idx)
		if !ok {
			t.Fatalf("missing live index %+v", idx)
		}
		if gotValue := *fi.Pointer(); gotValue != wantValue {
			t.Fatalf("index %+v value=%v want=%v", idx, gotValue, wantValue)
		}
		if iterValue, ok := got[idx]; !ok || iterValue != wantValue {
			t.Fatalf("iterate mismatch for %+v: got=%v present=%v want=%v", idx, iterValue, ok, wantValue)
		}
	}
	for idx := range got {
		if _, ok := want[idx]; !ok {
			t.Fatalf("unexpected iterated index %+v", idx)
		}
	}
}

func TestGolony_InternalInvariants_Randomized(t *testing.T) {
	for seed := int64(0); seed < 200; seed++ {
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
			rng := rand.New(rand.NewSource(seed))
			g := New[int](8)
			want := map[Index[int]]int{}
			live := make([]Index[int], 0, 64)
			nextCheck := uint32(1)

			for step := 0; step < 300; step++ {
				op := rng.Intn(4)
				switch {
				case len(live) == 0 || op == 0:
					fi := g.Insert(nextCheck)
					*fi.Pointer() = step
					want[fi.Index()] = step
					live = append(live, fi.Index())
					nextCheck++
				case op == 1:
					pos := rng.Intn(len(live))
					idx := live[pos]
					if !g.Erase(idx) {
						t.Fatalf("erase failed for live index %+v", idx)
					}
					delete(want, idx)
					live = append(live[:pos], live[pos+1:]...)
				case op == 2:
					pos := rng.Intn(len(live))
					idx := live[pos]
					fi, ok := g.Get(idx)
					if !ok {
						t.Fatalf("get failed for live index %+v", idx)
					}
					if got := *fi.Pointer(); got != want[idx] {
						t.Fatalf("get value=%d want=%d for %+v", got, want[idx], idx)
					}
				case op == 3:
					visited := map[Index[int]]int{}
					g.Iterate(func(fi FatIndex[int]) (erase, stop bool) {
						visited[fi.Index()] = *fi.Pointer()
						return false, false
					})
					if len(visited) != len(want) {
						t.Fatalf("iterate count=%d want=%d", len(visited), len(want))
					}
					for idx, wantValue := range want {
						if gotValue, ok := visited[idx]; !ok || gotValue != wantValue {
							t.Fatalf("iterate mismatch for %+v: got=%d present=%v want=%d", idx, gotValue, ok, wantValue)
						}
					}
				}

				validateContainerState(t, g, want)
			}
		})
	}
}

func TestRepro_ZeroValueIndexCanAliasLiveElement(t *testing.T) {
	g := New[int](8)
	fi := g.Insert(0)
	*fi.Pointer() = 123

	retrieved, ok := g.Get(Index[int]{})
	if !ok {
		t.Fatal("zero-value index should have resolved to the first slot when check=0")
	}
	if got := *retrieved.Pointer(); got != 123 {
		t.Fatalf("zero-value index returned %d want 123", got)
	}
}

func TestRepro_StalePointerCanMutateReusedSlot(t *testing.T) {
	g := New[int](8)

	first := g.Insert(1)
	stale := first.Pointer()
	*stale = 10

	if !g.Erase(first.Index()) {
		t.Fatal("expected erase to succeed")
	}

	second := g.Insert(2)
	*second.Pointer() = 20

	*stale = 99

	retrieved, ok := g.Get(second.Index())
	if !ok {
		t.Fatal("expected second element to remain accessible")
	}
	if got := *retrieved.Pointer(); got != 99 {
		t.Fatalf("stale pointer write did not reach the reused slot: got %d want 99", got)
	}
}

func TestRepro_IterateInsertionSemanticsDependOnAllocationPath(t *testing.T) {
	t.Run("insertion into current group is visited immediately", func(t *testing.T) {
		g := New[int](8)
		fi := g.Insert(1)
		*fi.Pointer() = 1

		var visited []int
		inserted := false
		g.Iterate(func(current FatIndex[int]) (erase, stop bool) {
			visited = append(visited, *current.Pointer())
			if !inserted {
				inserted = true
				next := g.Insert(2)
				*next.Pointer() = 2
			}
			return false, false
		})

		if fmt.Sprint(visited) != "[1 2]" {
			t.Fatalf("visited=%v want [1 2]", visited)
		}
	})

	t.Run("insertion into a new group is not visited in the same iteration", func(t *testing.T) {
		g := New[int](8)
		for i := 0; i < 8; i++ {
			fi := g.Insert(uint32(i + 1))
			*fi.Pointer() = i + 1
		}

		var visited []int
		inserted := false
		g.Iterate(func(current FatIndex[int]) (erase, stop bool) {
			visited = append(visited, *current.Pointer())
			if !inserted {
				inserted = true
				next := g.Insert(99)
				*next.Pointer() = 99
			}
			return false, false
		})

		if fmt.Sprint(visited) != "[1 2 3 4 5 6 7 8]" {
			t.Fatalf("visited=%v want [1 2 3 4 5 6 7 8]", visited)
		}
	})
}
