package webhook

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

func routeFiltersMatch(filters any, payload map[string]any, eventType string, headers map[string]string) bool {
	return routeFiltersMatchWithHome(filters, payload, eventType, headers, "")
}

func routeFiltersMatchWithHome(filters any, payload map[string]any, eventType string, headers map[string]string, profileHome string) bool {
	if filters == nil {
		return true
	}
	switch specs := filters.(type) {
	case map[string]any:
		if len(specs) == 0 {
			return true
		}
		return filterMatches(specs, payload, eventType, headers, profileHome)
	case []any:
		for _, spec := range specs {
			if !filterMatches(spec, payload, eventType, headers, profileHome) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func filterMatches(raw any, payload map[string]any, eventType string, headers map[string]string, profileHome string) bool {
	matched, valid := evaluateFilter(raw, payload, eventType, headers, profileHome)
	return valid && matched
}

func evaluateFilter(raw any, payload map[string]any, eventType string, headers map[string]string, profileHome string) (bool, bool) {
	spec, ok := raw.(map[string]any)
	if !ok {
		return false, false
	}
	if rawItems, exists := spec["all"]; exists {
		items, ok := rawItems.([]any)
		if !ok {
			return false, false
		}
		matched := true
		for _, item := range items {
			childMatched, valid := evaluateFilter(item, payload, eventType, headers, profileHome)
			if !valid {
				return false, false
			}
			matched = matched && childMatched
		}
		return matched, true
	}
	if rawItems, exists := spec["any"]; exists {
		items, ok := rawItems.([]any)
		if !ok {
			return false, false
		}
		matched := false
		for _, item := range items {
			childMatched, valid := evaluateFilter(item, payload, eventType, headers, profileHome)
			if !valid {
				return false, false
			}
			matched = matched || childMatched
		}
		return matched, true
	}
	if item, exists := spec["not"]; exists {
		matched, valid := evaluateFilter(item, payload, eventType, headers, profileHome)
		return !matched, valid
	}
	field, ok := spec["field"].(string)
	if !ok || strings.TrimSpace(field) == "" {
		return false, false
	}
	value, found := resolveFilterField(field, payload, eventType, headers)
	if raw, exists := spec["exists"]; exists {
		want, ok := raw.(bool)
		return found == want, ok
	}
	if raw, exists := spec["missing"]; exists {
		want, ok := raw.(bool)
		return want && !found, ok && want
	}
	if want, exists := spec["equals"]; exists {
		return found && filterValuesEqual(value, want), true
	}
	if want, exists := spec["not_equals"]; exists {
		return !found || !filterValuesEqual(value, want), true
	}
	if needle, exists := spec["contains"]; exists {
		return found && filterContains(value, needle), true
	}
	if raw, exists := spec["in"]; exists {
		values, ok := raw.([]any)
		if !ok {
			return false, false
		}
		for _, candidate := range values {
			if found && filterValuesEqual(value, candidate) {
				return true, true
			}
		}
		return false, true
	}
	if raw, exists := spec["in_file"]; exists {
		values, ok := loadFilterFileValues(raw, profileHome)
		if !ok {
			return false, false
		}
		if !found {
			return false, true
		}
		for _, candidate := range values {
			if filterValuesEqual(value, candidate) {
				return true, true
			}
		}
		return false, true
	}
	if raw, exists := spec["regex"]; exists {
		pattern, ok := raw.(string)
		if !ok {
			return false, false
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false, false
		}
		return found && re.MatchString(stringifyFilterValue(value)), true
	}
	return false, false
}

const maxFilterFileBytes = 1 << 20

func loadFilterFileValues(pathValue any, profileHome string) ([]any, bool) {
	raw, ok := pathValue.(string)
	raw = strings.TrimSpace(raw)
	profileHome = strings.TrimSpace(profileHome)
	if !ok || raw == "" || profileHome == "" {
		return nil, false
	}

	rel := raw
	for _, alias := range []string{"~/.hermes", "~/.gormes"} {
		if raw == alias {
			rel = "."
			break
		}
		if strings.HasPrefix(raw, alias+"/") {
			rel = strings.TrimPrefix(raw, alias+"/")
			break
		}
	}
	if strings.HasPrefix(rel, "~") || filepath.IsAbs(rel) {
		return nil, false
	}
	root, err := filepath.Abs(profileHome)
	if err != nil {
		return nil, false
	}
	candidate := filepath.Join(root, filepath.FromSlash(rel))
	within, err := filepath.Rel(root, candidate)
	if err != nil || within == ".." || strings.HasPrefix(within, ".."+string(filepath.Separator)) {
		return nil, false
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, false
	}
	rootInfo, err := os.Stat(resolvedRoot)
	if err != nil || !rootInfo.IsDir() {
		return nil, false
	}
	resolvedCandidate, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return nil, false
	}
	within, err = filepath.Rel(resolvedRoot, resolvedCandidate)
	if err != nil || within == ".." || strings.HasPrefix(within, ".."+string(filepath.Separator)) {
		return nil, false
	}
	file, err := os.Open(resolvedCandidate)
	if err != nil {
		return nil, false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxFilterFileBytes {
		return nil, false
	}
	data, err := io.ReadAll(io.LimitReader(file, maxFilterFileBytes+1))
	if err != nil || len(data) > maxFilterFileBytes || !utf8.Valid(data) {
		return nil, false
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		values := make([]any, 0)
		for _, line := range strings.Split(string(data), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				values = append(values, line)
			}
		}
		return values, true
	}
	switch value := decoded.(type) {
	case []any:
		return value, true
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		values := make([]any, len(keys))
		for i, key := range keys {
			values[i] = key
		}
		return values, true
	default:
		return []any{value}, true
	}
}

func filterContains(value, needle any) bool {
	switch typed := value.(type) {
	case []any:
		for _, candidate := range typed {
			if filterValuesEqual(candidate, needle) {
				return true
			}
		}
		return false
	case map[string]any:
		key, ok := needle.(string)
		if !ok {
			return false
		}
		_, ok = typed[key]
		return ok
	default:
		return strings.Contains(stringifyFilterValue(value), stringifyFilterValue(needle))
	}
}

func filterValuesEqual(left, right any) bool {
	if reflect.DeepEqual(left, right) {
		return true
	}
	leftNumber, leftOK := filterNumber(left)
	rightNumber, rightOK := filterNumber(right)
	return leftOK && rightOK && leftNumber == rightNumber
}

func filterNumber(value any) (float64, bool) {
	ref := reflect.ValueOf(value)
	if !ref.IsValid() {
		return 0, false
	}
	switch ref.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(ref.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(ref.Uint()), true
	case reflect.Float32, reflect.Float64:
		return ref.Float(), true
	default:
		return 0, false
	}
}

func stringifyFilterValue(value any) string {
	switch typed := value.(type) {
	case bool:
		if typed {
			return "True"
		}
		return "False"
	case nil:
		return "None"
	case map[string]any, []any:
		if data, err := json.Marshal(typed); err == nil {
			return string(data)
		}
	}
	return fmt.Sprint(value)
}

func resolveFilterField(field string, payload map[string]any, eventType string, headers map[string]string) (any, bool) {
	var parts []string
	for _, part := range strings.Split(strings.TrimSpace(field), ".") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return nil, false
	}
	var current any = payload
	if parts[0] == "payload" {
		current = payload
		if nested, ok := payload["payload"]; ok {
			current = nested
		}
		parts = parts[1:]
	} else if parts[0] == "event" || parts[0] == "event_type" {
		current = eventType
		parts = parts[1:]
	} else if parts[0] == "headers" {
		current = headers
		parts = parts[1:]
	}
	for _, part := range parts {
		var ok bool
		switch value := current.(type) {
		case map[string]any:
			current, ok = value[part]
		case map[string]string:
			for key, candidate := range value {
				if strings.EqualFold(key, part) {
					current, ok = candidate, true
					break
				}
			}
		case []any:
			index, err := strconv.Atoi(part)
			if err == nil && index >= 0 && index < len(value) {
				current, ok = value[index], true
			}
		default:
			return nil, false
		}
		if !ok {
			return nil, false
		}
	}
	return current, true
}
