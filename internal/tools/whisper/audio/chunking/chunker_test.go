package chunking

import (
	"slices"
	"testing"
	"time"
)

func TestChunkPCMWithOverlap(t *testing.T) {
	got := ChunkPCMWithOverlap([]int16{0, 1, 2, 3, 4, 5}, 4, time.Second, 500*time.Millisecond)
	want := [][]int16{{0, 1, 2, 3}, {2, 3, 4, 5}}
	if len(got) != len(want) {
		t.Fatalf("chunk count = %d, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if !slices.Equal(got[i], want[i]) {
			t.Fatalf("chunk %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestChunkPCMWithOverlapClampsOversizedOverlap(t *testing.T) {
	got := ChunkPCMWithOverlap([]int16{0, 1, 2, 3}, 2, time.Second, 2*time.Second)
	want := [][]int16{{0, 1}, {1, 2}, {2, 3}}
	if len(got) != len(want) {
		t.Fatalf("chunk count = %d, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if !slices.Equal(got[i], want[i]) {
			t.Fatalf("chunk %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestChunkPCMFixedWindowEdges(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		if got := ChunkPCM(nil, 16000, time.Second); len(got) != 0 {
			t.Fatalf("ChunkPCM(empty) = %v, want no chunks", got)
		}
	})

	t.Run("single short chunk", func(t *testing.T) {
		got := ChunkPCM([]int16{1, 2, 3}, 10, time.Second)
		if len(got) != 1 || !slices.Equal(got[0], []int16{1, 2, 3}) {
			t.Fatalf("ChunkPCM short = %v", got)
		}
	})

	t.Run("equal chunks and ragged tail", func(t *testing.T) {
		got := ChunkPCM([]int16{0, 1, 2, 3, 4}, 4, 500*time.Millisecond)
		want := [][]int16{{0, 1}, {2, 3}, {4}}
		if len(got) != len(want) {
			t.Fatalf("chunk count = %d, want %d: %v", len(got), len(want), got)
		}
		for i := range want {
			if !slices.Equal(got[i], want[i]) {
				t.Fatalf("chunk %d = %v, want %v", i, got[i], want[i])
			}
		}
	})

	t.Run("preserves silent samples", func(t *testing.T) {
		got := ChunkPCM([]int16{0, 0, 0, 0}, 4, 250*time.Millisecond)
		if len(got) != 4 {
			t.Fatalf("silent chunk count = %d, want 4: %v", len(got), got)
		}
		for i, chunk := range got {
			if !slices.Equal(chunk, []int16{0}) {
				t.Fatalf("silent chunk %d = %v, want one zero", i, chunk)
			}
		}
	})
}
