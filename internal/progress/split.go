package progress

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrMalformedSplit is returned by Load when a path resolves to a split
// layout that is missing its index manifest, missing a member file the index
// lists, or whose index/member JSON does not parse. It is wrapped (use
// errors.Is) so callers can distinguish a corrupt split directory from a
// missing or malformed monolithic file. The monolithic path never returns it.
var ErrMalformedSplit = errors.New("progress: malformed split layout")

// splitIndexName is the manifest file at the root of a split layout. Its
// presence is also how Load recognises a directory as a split backlog.
const splitIndexName = "index.json"

// splitPhasesDir holds one <phaseID>.json member file per top-level phase.
const splitPhasesDir = "phases"

// SplitPhase is one top-level phase plus the map key it lives under in the
// monolithic Progress.Phases map.
type SplitPhase struct {
	ID    string
	Phase Phase
}

// SplitLayout is the in-memory decomposition of a Progress into its Meta plus
// one member per top-level phase, ordered by the same natural-numeric key
// order the stable marshaller uses. Merge is its exact inverse.
type SplitLayout struct {
	Meta   Meta
	Phases []SplitPhase
}

// splitIndex is the on-disk index.json manifest: the document Meta plus the
// ordered list of phase member files. The actual phases live in
// phases/<id>.json so a later child can move per-module rows without
// touching this shim.
type splitIndex struct {
	Meta   Meta     `json:"meta"`
	Phases []string `json:"phases"`
}

// Split decomposes a Progress into a SplitLayout. It moves no rows and loses
// nothing: Merge(Split(p)) reconstructs an equal Progress.
func Split(p *Progress) (*SplitLayout, error) {
	if p == nil {
		return nil, fmt.Errorf("progress: Split of nil Progress")
	}
	sl := &SplitLayout{Meta: p.Meta}
	for _, id := range sortedMapKeys(p.Phases) {
		sl.Phases = append(sl.Phases, SplitPhase{ID: id, Phase: p.Phases[id]})
	}
	return sl, nil
}

// Merge is the exact inverse of Split: it recomposes the monolithic Progress
// from a SplitLayout. Duplicate phase IDs are rejected so a corrupted layout
// cannot silently drop a phase.
func Merge(sl *SplitLayout) (*Progress, error) {
	if sl == nil {
		return nil, fmt.Errorf("progress: Merge of nil SplitLayout")
	}
	p := &Progress{Meta: sl.Meta, Phases: make(map[string]Phase, len(sl.Phases))}
	for _, sp := range sl.Phases {
		if _, dup := p.Phases[sp.ID]; dup {
			return nil, fmt.Errorf("%w: duplicate phase %q", ErrMalformedSplit, sp.ID)
		}
		p.Phases[sp.ID] = sp.Phase
	}
	return p, nil
}

// WriteSplit writes a Progress to dir as a split layout: an index.json
// manifest plus one phases/<id>.json member per top-level phase. Every file
// is encoded with the same no-HTML-escape, indented encoder SaveProgress
// uses, so each member is itself stable and diff-friendly.
func WriteSplit(dir string, p *Progress) error {
	sl, err := Split(p)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, splitPhasesDir), 0o755); err != nil {
		return fmt.Errorf("progress: WriteSplit mkdir: %w", err)
	}

	idx := splitIndex{Meta: sl.Meta}
	for _, sp := range sl.Phases {
		idx.Phases = append(idx.Phases, sp.ID)
		body, err := encodeStable(sp.Phase)
		if err != nil {
			return fmt.Errorf("progress: WriteSplit phase %q: %w", sp.ID, err)
		}
		member := filepath.Join(dir, splitPhasesDir, sp.ID+".json")
		if err := atomicWrite(member, body); err != nil {
			return err
		}
	}
	body, err := encodeStable(idx)
	if err != nil {
		return fmt.Errorf("progress: WriteSplit index: %w", err)
	}
	return atomicWrite(filepath.Join(dir, splitIndexName), body)
}

// loadSplit reads a split layout rooted at dir back into a Progress. Any
// structural defect (absent/corrupt index, absent/corrupt member) is reported
// as ErrMalformedSplit so the caller never receives a silently partial
// backlog.
func loadSplit(dir string) (*Progress, error) {
	raw, err := os.ReadFile(filepath.Join(dir, splitIndexName))
	if err != nil {
		return nil, fmt.Errorf("%w: read index: %v", ErrMalformedSplit, err)
	}
	var idx splitIndex
	if err := json.Unmarshal(raw, &idx); err != nil {
		return nil, fmt.Errorf("%w: parse index: %v", ErrMalformedSplit, err)
	}

	sl := &SplitLayout{Meta: idx.Meta}
	for _, id := range idx.Phases {
		member := filepath.Join(dir, splitPhasesDir, id+".json")
		mb, err := os.ReadFile(member)
		if err != nil {
			return nil, fmt.Errorf("%w: read phase %q: %v", ErrMalformedSplit, id, err)
		}
		var ph Phase
		if err := json.Unmarshal(mb, &ph); err != nil {
			return nil, fmt.Errorf("%w: parse phase %q: %v", ErrMalformedSplit, id, err)
		}
		sl.Phases = append(sl.Phases, SplitPhase{ID: id, Phase: ph})
	}
	return Merge(sl)
}

// encodeStable mirrors SaveProgress's encoder settings (no HTML escaping,
// two-space indent, trailing newline) so split member files are byte-stable
// across runs and diff-friendly.
func encodeStable(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
