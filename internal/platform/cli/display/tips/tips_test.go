package tips

import (
	"strings"
	"testing"
)

func TestCorpusInvariants(t *testing.T) {
	if len(Corpus) < 30 {
		t.Fatalf("len(Corpus) = %d, want >= 30", len(Corpus))
	}

	seen := make(map[string]struct{}, len(Corpus))
	for i, tip := range Corpus {
		if tip == "" {
			t.Errorf("Corpus[%d] is empty", i)
			continue
		}
		if strings.Contains(tip, "\n") {
			t.Errorf("Corpus[%d] contains a newline: %q", i, tip)
		}
		if _, ok := seen[tip]; ok {
			t.Errorf("Corpus[%d] duplicates an earlier entry: %q", i, tip)
		}
		seen[tip] = struct{}{}
	}
}

func TestFor_DeterministicForFixedSeed(t *testing.T) {
	first := For(42)
	second := For(42)
	if first != second {
		t.Fatalf("For(42) returned %q then %q; want stable output", first, second)
	}

	want := Corpus[42%int64(len(Corpus))]
	if first != want {
		t.Fatalf("For(42) = %q, want Corpus[42 mod len(Corpus)] = %q", first, want)
	}
}

func TestFor_HandlesNegativeSeed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("For(-1) panicked: %v", r)
		}
	}()

	got := For(-1)
	if got == "" {
		t.Fatalf("For(-1) returned empty string")
	}

	found := false
	for _, tip := range Corpus {
		if tip == got {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("For(-1) = %q, which is not present in Corpus", got)
	}
}

func TestFor_HandlesZeroSeed(t *testing.T) {
	if got := For(0); got != Corpus[0] {
		t.Fatalf("For(0) = %q, want Corpus[0] = %q", got, Corpus[0])
	}
}

func TestFor_DistributesAcrossCorpus(t *testing.T) {
	counts := make(map[string]int, len(Corpus))
	for i := 0; i < len(Corpus); i++ {
		counts[For(int64(i))]++
	}

	for _, tip := range Corpus {
		switch counts[tip] {
		case 1:
			// expected
		case 0:
			t.Errorf("tip %q was never returned by For(int64(i)) for i in [0, len(Corpus))", tip)
		default:
			t.Errorf("tip %q was returned %d times; want exactly 1", tip, counts[tip])
		}
	}
}
