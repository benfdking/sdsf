package cursorautomations

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// State records each managed automation's server-assigned id, keyed by directory
// name. It's a cache over the live Cursor list, not the source of truth: a lost
// or unreadable state file is recovered by adopting live automations by name, so
// losing it degrades to re-adoption rather than re-creating duplicates.
type State struct {
	Automations map[string]TrackedAutomation `json:"automations"`
}

type TrackedAutomation struct {
	AutomationID string `json:"automationId"`

	// AppliedPayload is the desired config as of the last successful write (our
	// own encoding); AppliedLive is the live automation projected onto the
	// managed fields, captured just after that write (the server's encoding).
	// Together they let the next run prove an automation unchanged — and say
	// field-by-field what changed when it isn't — without cross-encoding
	// comparisons (see baseline.go). Absent on entries written before baselines
	// existed; sync then applies once to establish them.
	AppliedPayload json.RawMessage `json:"appliedPayload,omitempty"`
	AppliedLive    json.RawMessage `json:"appliedLive,omitempty"`
}

// LoadState reads the state file. A missing or unparseable file yields empty
// state, not an error, so a first run or a lost/corrupt cache falls back to
// live-list adoption instead of failing the sync.
func LoadState(path string) *State {
	empty := &State{Automations: map[string]TrackedAutomation{}}

	data, err := os.ReadFile(path)
	if err != nil {
		return empty
	}
	var loaded State
	if json.Unmarshal(data, &loaded) != nil || loaded.Automations == nil {
		return empty
	}
	return &loaded
}

func (s *State) ID(dir string) (string, bool) {
	t, ok := s.Automations[dir]
	return t.AutomationID, ok && t.AutomationID != ""
}

// Record stores a newly resolved id for dir. It deliberately replaces the whole
// entry: a new identity (created or adopted) invalidates any baselines recorded
// against the old one.
func (s *State) Record(dir, id string) {
	s.Automations[dir] = TrackedAutomation{AutomationID: id}
}

// RecordBaseline stores the post-write baselines for dir, preserving its id.
func (s *State) RecordBaseline(dir string, payload, live json.RawMessage) {
	t := s.Automations[dir]
	t.AppliedPayload = payload
	t.AppliedLive = live
	s.Automations[dir] = t
}

// Baseline returns the stored baselines for dir; ok only when both are present,
// since proving "unchanged" needs the pair.
func (s *State) Baseline(dir string) (payload, live json.RawMessage, ok bool) {
	t := s.Automations[dir]
	return t.AppliedPayload, t.AppliedLive, len(t.AppliedPayload) > 0 && len(t.AppliedLive) > 0
}

func (s *State) Forget(dir string) {
	delete(s.Automations, dir)
}

// ResolveTarget decides how to sync an automation: the id to use ("" means
// create) and the source ("state", "adopt", "create"). A recorded id is used only
// if it's still live; a stale one falls through to adopt-by-name or create, so we
// never update a deleted automation and a lost state cache re-adopts not duplicates.
func ResolveTarget(dir, name string, state *State, liveByName map[string]string, liveIDs map[string]bool) (id, source string) {
	if id, ok := state.ID(dir); ok && liveIDs[id] {
		return id, "state"
	}
	if adopted, found := liveByName[name]; found {
		return adopted, "adopt"
	}
	return "", "create"
}

// Save writes the state via a temp file and rename so a crash mid-write can't
// leave a truncated file that the next load would silently read as empty.
func (s *State) Save(path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".state-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}
