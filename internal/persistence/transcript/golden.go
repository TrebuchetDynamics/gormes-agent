package transcript

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
)

// JSONDiff describes the first stable field-level difference between two
// golden JSON documents.
type JSONDiff struct {
	Path    string
	Message string
	Want    any
	Got     any
}

func (d JSONDiff) Error() string {
	if d.Message != "" {
		return d.Message
	}
	return fmt.Sprintf("json mismatch at %s: got %v, want %v", d.Path, d.Got, d.Want)
}

// MarshalStableJSON writes indented JSON with a trailing newline so fixtures
// remain byte-stable and readable in diffs.
func MarshalStableJSON(v any) ([]byte, error) {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// CompareGoldenJSON reports the first semantic JSON mismatch with an exact
// field path. It ignores object key order but preserves array order.
func CompareGoldenJSON(wantRaw, gotRaw []byte) error {
	want, err := decodeGoldenJSON(wantRaw)
	if err != nil {
		return fmt.Errorf("decode golden fixture: %w", err)
	}
	got, err := decodeGoldenJSON(gotRaw)
	if err != nil {
		return fmt.Errorf("decode generated transcript: %w", err)
	}
	return compareGoldenValue("$", want, got)
}

func decodeGoldenJSON(raw []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var out any
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	if dec.More() {
		return nil, fmt.Errorf("multiple JSON values")
	}
	return out, nil
}

func compareGoldenValue(path string, want, got any) error {
	switch wantTyped := want.(type) {
	case map[string]any:
		gotTyped, ok := got.(map[string]any)
		if !ok {
			return goldenTypeDiff(path, want, got)
		}
		wantKeys := sortedGoldenKeys(wantTyped)
		gotKeys := sortedGoldenKeys(gotTyped)
		if !reflect.DeepEqual(wantKeys, gotKeys) {
			return JSONDiff{
				Path:    path,
				Want:    wantKeys,
				Got:     gotKeys,
				Message: fmt.Sprintf("json object keys mismatch at %s: got %v, want %v", path, gotKeys, wantKeys),
			}
		}
		for _, key := range wantKeys {
			if err := compareGoldenValue(path+"."+key, wantTyped[key], gotTyped[key]); err != nil {
				return err
			}
		}
		return nil
	case []any:
		gotTyped, ok := got.([]any)
		if !ok {
			return goldenTypeDiff(path, want, got)
		}
		if len(wantTyped) != len(gotTyped) {
			return JSONDiff{
				Path:    path,
				Want:    len(wantTyped),
				Got:     len(gotTyped),
				Message: fmt.Sprintf("json array length mismatch at %s: got %d, want %d", path, len(gotTyped), len(wantTyped)),
			}
		}
		for i := range wantTyped {
			if err := compareGoldenValue(path+"["+strconv.Itoa(i)+"]", wantTyped[i], gotTyped[i]); err != nil {
				return err
			}
		}
		return nil
	default:
		if !reflect.DeepEqual(want, got) {
			return JSONDiff{
				Path:    path,
				Want:    want,
				Got:     got,
				Message: fmt.Sprintf("json value mismatch at %s: got %v, want %v", path, got, want),
			}
		}
		return nil
	}
}

func goldenTypeDiff(path string, want, got any) error {
	return JSONDiff{
		Path:    path,
		Want:    fmt.Sprintf("%T", want),
		Got:     fmt.Sprintf("%T", got),
		Message: fmt.Sprintf("json type mismatch at %s: got %T, want %T", path, got, want),
	}
}

func sortedGoldenKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
