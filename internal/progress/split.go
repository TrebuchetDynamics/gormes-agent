package progress

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

// splitModulesDir holds one <module>.json member file per subsystem module
// when the layout is module-keyed (C5b).
const splitModulesDir = "modules"

// Split layout keying modes. Phase keying (C1) is the default and the empty
// string is treated as "phase" for back-compat with indexes written before
// C5b. Module keying groups item bodies by progress.Module while the index
// carries the full structural skeleton so Merge reconstructs byte-identically.
const (
	splitKeyByPhase  = "phase"
	splitKeyByModule = "module"
)

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

// splitIndex is the on-disk index.json manifest. Phase keying (C1) records
// Meta + the ordered phase member list and omits the C5b fields, so a
// phase-keyed index is byte-identical to what C1 wrote. Module keying records
// KeyBy="module", the ordered module member list, and the full structural
// Skeleton (every phase/subphase's metadata + ordering) so loadSplit can
// reattach module-grouped item bodies into their real phase/subphase and
// Merge reconstructs the byte-identical monolith.
type splitIndex struct {
	Meta     Meta        `json:"meta"`
	KeyBy    string      `json:"key_by,omitempty"`
	Phases   []string    `json:"phases,omitempty"`
	Skeleton []skelPhase `json:"skeleton,omitempty"`
	Modules  []string    `json:"modules,omitempty"`
}

// skelPhase/skelSubphase carry the structural identity that module-grouped
// member files cannot: phase/subphase metadata and the per-subphase item
// count + ordering. Item bodies live in modules/<module>.json keyed by
// (phase, subphase, position) so every row reattaches to its real slot.
type skelPhase struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Deliverable    string         `json:"deliverable"`
	DependencyNote string         `json:"dependency_note,omitempty"`
	Subphases      []skelSubphase `json:"subphases"`
}

type skelSubphase struct {
	ID         string                     `json:"id"`
	Name       string                     `json:"name"`
	Priority   string                     `json:"priority,omitempty"`
	Status     Status                     `json:"status,omitempty"`
	DriftState *DriftState                `json:"drift_state,omitempty"`
	ItemCount  int                        `json:"item_count,omitempty"`
	Extra      map[string]json.RawMessage `json:"extra,omitempty"`
}

// moduleRow is one item body plus the coordinates that place it back into its
// real phase/subphase at its original position.
type moduleRow struct {
	Phase    string `json:"phase"`
	Subphase string `json:"subphase"`
	Pos      int    `json:"pos"`
	Item     Item   `json:"item"`
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
// WriteSplit writes a phase-keyed split layout (the C1 default). Output is
// byte-identical to what C1 wrote: WriteSplit == WriteSplitBy(dir, p, phase).
func WriteSplit(dir string, p *Progress) error {
	return WriteSplitBy(dir, p, splitKeyByPhase)
}

// WriteSplitBy writes a Progress to dir as a split layout keyed either by
// phase (default, byte-identical to C1) or by module. Module keying groups
// item bodies under modules/<module>.json by progress.Module while the index
// keeps the full structural skeleton, so the layout is operator-readable per
// subsystem yet still reconstructs the byte-identical monolith via Merge.
func WriteSplitBy(dir string, p *Progress, keyBy string) error {
	sl, err := Split(p)
	if err != nil {
		return err
	}
	switch keyBy {
	case "", splitKeyByPhase:
		return writeSplitByPhase(dir, sl)
	case splitKeyByModule:
		return writeSplitByModule(dir, sl)
	default:
		return fmt.Errorf("%w: unknown key_by %q", ErrMalformedSplit, keyBy)
	}
}

func writeSplitByPhase(dir string, sl *SplitLayout) error {
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

func writeSplitByModule(dir string, sl *SplitLayout) error {
	if err := os.MkdirAll(filepath.Join(dir, splitModulesDir), 0o755); err != nil {
		return fmt.Errorf("progress: WriteSplit mkdir: %w", err)
	}
	idx := splitIndex{Meta: sl.Meta, KeyBy: splitKeyByModule}
	byModule := map[string][]moduleRow{}
	for _, sp := range sl.Phases {
		skp := skelPhase{
			ID: sp.ID, Name: sp.Phase.Name,
			Deliverable: sp.Phase.Deliverable, DependencyNote: sp.Phase.DependencyNote,
		}
		for _, subID := range sortedMapKeys(sp.Phase.Subphases) {
			sub := sp.Phase.Subphases[subID]
			sks := skelSubphase{
				ID: subID, Name: sub.Name, Priority: sub.Priority,
				Status: sub.Status, DriftState: sub.DriftState,
				ItemCount: len(sub.Items), Extra: cloneRawMap(sub.Extra),
			}
			for pos, it := range sub.Items {
				m := Module(it, sp.ID, subID)
				byModule[m] = append(byModule[m], moduleRow{
					Phase: sp.ID, Subphase: subID, Pos: pos, Item: it,
				})
			}
			skp.Subphases = append(skp.Subphases, sks)
		}
		idx.Skeleton = append(idx.Skeleton, skp)
	}

	mods := make([]string, 0, len(byModule))
	for m := range byModule {
		mods = append(mods, m)
	}
	slices.Sort(mods)
	for _, m := range mods {
		idx.Modules = append(idx.Modules, m)
		body, err := encodeStable(byModule[m])
		if err != nil {
			return fmt.Errorf("progress: WriteSplit module %q: %w", m, err)
		}
		if err := atomicWrite(filepath.Join(dir, splitModulesDir, m+".json"), body); err != nil {
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

	switch idx.KeyBy {
	case "", splitKeyByPhase:
		return loadSplitByPhase(dir, idx)
	case splitKeyByModule:
		return loadSplitByModule(dir, idx)
	default:
		return nil, fmt.Errorf("%w: unknown key_by %q", ErrMalformedSplit, idx.KeyBy)
	}
}

func loadSplitByPhase(dir string, idx splitIndex) (*Progress, error) {
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

// loadSplitByModule reattaches module-grouped item bodies into the skeleton's
// real phases/subphases at their original positions, then Merges to the
// byte-identical monolith. Any missing/corrupt module file or an item slot
// the modules failed to fill is ErrMalformedSplit — never a silently partial
// backlog.
func loadSplitByModule(dir string, idx splitIndex) (*Progress, error) {
	// Gather every item body keyed by phase/subphase/position.
	type coord struct{ phase, sub string }
	bodies := map[coord]map[int]Item{}
	for _, m := range idx.Modules {
		member := filepath.Join(dir, splitModulesDir, m+".json")
		mb, err := os.ReadFile(member)
		if err != nil {
			return nil, fmt.Errorf("%w: read module %q: %v", ErrMalformedSplit, m, err)
		}
		var rows []moduleRow
		if err := json.Unmarshal(mb, &rows); err != nil {
			return nil, fmt.Errorf("%w: parse module %q: %v", ErrMalformedSplit, m, err)
		}
		for _, r := range rows {
			c := coord{r.Phase, r.Subphase}
			if bodies[c] == nil {
				bodies[c] = map[int]Item{}
			}
			bodies[c][r.Pos] = r.Item
		}
	}

	sl := &SplitLayout{Meta: idx.Meta}
	for _, skp := range idx.Skeleton {
		ph := Phase{
			Name: skp.Name, Deliverable: skp.Deliverable,
			DependencyNote: skp.DependencyNote,
			Subphases:      make(map[string]Subphase, len(skp.Subphases)),
		}
		for _, sks := range skp.Subphases {
			sub := Subphase{
				Name: sks.Name, Priority: sks.Priority,
				Status: sks.Status, DriftState: sks.DriftState,
				Extra: cloneRawMap(sks.Extra),
			}
			if sks.ItemCount > 0 {
				slot := bodies[coord{skp.ID, sks.ID}]
				items := make([]Item, sks.ItemCount)
				for pos := 0; pos < sks.ItemCount; pos++ {
					it, ok := slot[pos]
					if !ok {
						return nil, fmt.Errorf("%w: missing item %s/%s[%d] in module members",
							ErrMalformedSplit, skp.ID, sks.ID, pos)
					}
					items[pos] = it
				}
				sub.Items = items
			}
			ph.Subphases[sks.ID] = sub
		}
		sl.Phases = append(sl.Phases, SplitPhase{ID: skp.ID, Phase: ph})
	}
	return Merge(sl)
}

func cloneRawMap(in map[string]json.RawMessage) map[string]json.RawMessage {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]json.RawMessage, len(in))
	for key, value := range in {
		out[key] = append(json.RawMessage(nil), value...)
	}
	return out
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
