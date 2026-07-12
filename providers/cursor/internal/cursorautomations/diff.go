package cursorautomations

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

// FieldDiff is one field-level difference between two JSON documents, addressed
// by path so a sync log can say exactly what changed and where.
type FieldDiff struct {
	Path         string // e.g. workflow.triggers[0].slackTrigger.topLevelOnly
	Before       any
	After        any
	BeforeAbsent bool // the key exists only on the after side
	AfterAbsent  bool // the key exists only on the before side
}

func (d FieldDiff) String() string {
	return fmt.Sprintf("%s: %s → %s",
		d.Path, renderValue(d.Before, d.BeforeAbsent), renderValue(d.After, d.AfterAbsent))
}

func renderValue(v any, absent bool) string {
	if absent {
		return "(unset)"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return truncate(string(b), 120)
}

// DiffJSON deep-compares two JSON documents and returns one FieldDiff per
// differing leaf, added/removed key, or reshaped value. It does no
// normalisation — both documents must already share an encoding (see
// baseline.go for how sync guarantees that).
func DiffJSON(before, after json.RawMessage) ([]FieldDiff, error) {
	var b, a any
	if err := json.Unmarshal(before, &b); err != nil {
		return nil, fmt.Errorf("parsing before document: %w", err)
	}
	if err := json.Unmarshal(after, &a); err != nil {
		return nil, fmt.Errorf("parsing after document: %w", err)
	}
	var diffs []FieldDiff
	walkDiff(&diffs, "", b, a)
	return diffs, nil
}

func walkDiff(diffs *[]FieldDiff, path string, before, after any) {
	switch b := before.(type) {
	case map[string]any:
		a, ok := after.(map[string]any)
		if !ok {
			*diffs = append(*diffs, FieldDiff{Path: orRoot(path), Before: before, After: after})
			return
		}
		for _, k := range unionKeys(b, a) {
			bv, bok := b[k]
			av, aok := a[k]
			childPath := joinPath(path, k)
			switch {
			case bok && !aok:
				*diffs = append(*diffs, FieldDiff{Path: childPath, Before: bv, AfterAbsent: true})
			case !bok && aok:
				*diffs = append(*diffs, FieldDiff{Path: childPath, After: av, BeforeAbsent: true})
			default:
				walkDiff(diffs, childPath, bv, av)
			}
		}
	case []any:
		a, ok := after.([]any)
		// A reshaped or resized array diffs as a whole: index-by-index reports on
		// shifted elements would be noise, not signal.
		if !ok || len(b) != len(a) {
			*diffs = append(*diffs, FieldDiff{Path: orRoot(path), Before: before, After: after})
			return
		}
		for i := range b {
			walkDiff(diffs, fmt.Sprintf("%s[%d]", path, i), b[i], a[i])
		}
	default:
		if !reflect.DeepEqual(before, after) {
			*diffs = append(*diffs, FieldDiff{Path: orRoot(path), Before: before, After: after})
		}
	}
}

func unionKeys(a, b map[string]any) []string {
	seen := map[string]bool{}
	for k := range a {
		seen[k] = true
	}
	for k := range b {
		seen[k] = true
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func joinPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func orRoot(path string) string {
	if path == "" {
		return "(root)"
	}
	return path
}
