package cursorautomations

import (
	"encoding/json"
	"fmt"
)

// Baselines let a sync run prove "nothing changed" without ever comparing our
// YAML-derived payload against Cursor's live representation directly. That
// comparison would have to bridge two encodings — we send enum names, Cursor
// returns protobuf ints; Cursor adds legacy singular duplicates (channel/repo),
// server-assigned fields (agentOptions), and materialised defaults — and any
// mapping we got wrong would resurface as a perpetual no-op update. Instead,
// after every successful write we record two snapshots in state, and each is
// only ever compared against its own encoding domain:
//
//   - appliedPayload: the desired config we sent (our encoding). Compared
//     against the next run's desired config to detect repo-side changes.
//   - appliedLive: the live automation just after our write, projected onto the
//     managed fields (the server's encoding). Compared against the next run's
//     live automation to detect drift from edits outside the repo.

// managedTopLevelFields are the managed keys on the live automation object.
var managedTopLevelFields = []string{"name", "enabled", "scope"}

// managedWorkflowFields are the keys inside the live "workflow" object that the
// YAML config expresses — the only fields a sync can change, and therefore the
// only fields drift detection can meaningfully compare or re-assert. Values
// under these keys are kept verbatim, so if Cursor ever nested a volatile
// server-side value inside one, it would surface as drift on every run — loud
// in the sync log (a repeating field diff), and the cue to revisit this
// projection. A test guards these lists against the payload structs in
// types.go, so a field added to the payload can't silently escape drift
// detection.
var managedWorkflowFields = []string{
	"prompts", "model", "triggers", "actions",
	"memoryEnabled", "slackNotifiedChannels", "gitConfig",
}

// ProjectLive reduces a live automation object (as listed by Cursor) to the
// fields the sync manages, re-encoded deterministically, for use as a drift
// baseline. Projecting via an allowlist — rather than stripping known server
// fields — means unknown or volatile server-side keys (timestamps, run
// metadata, agentOptions, …) can never surface as false drift. Values are kept
// verbatim in the server's encoding; see the package comment above for why.
func ProjectLive(raw json.RawMessage) (json.RawMessage, error) {
	var automation map[string]any
	if err := json.Unmarshal(raw, &automation); err != nil {
		return nil, fmt.Errorf("parsing live automation: %w", err)
	}

	projected := map[string]any{}
	for _, k := range managedTopLevelFields {
		if v, ok := automation[k]; ok {
			projected[k] = v
		}
	}
	if wf, ok := automation["workflow"].(map[string]any); ok {
		projectedWorkflow := map[string]any{}
		for _, k := range managedWorkflowFields {
			if v, ok := wf[k]; ok {
				projectedWorkflow[k] = v
			}
		}
		projected["workflow"] = projectedWorkflow
	}

	out, err := json.Marshal(projected)
	if err != nil {
		return nil, fmt.Errorf("encoding projected automation: %w", err)
	}
	return out, nil
}

// desiredBaselineView is the desired config in the shape sent to
// /update-automation, minus the automationId (an identity, not config).
type desiredBaselineView struct {
	Name     string        `json:"name"`
	Enabled  bool          `json:"enabled"`
	Scope    string        `json:"scope"`
	Workflow innerWorkflow `json:"workflow"`
}

// DesiredBaseline renders the automation's desired config for recording as (and
// comparing against) the appliedPayload baseline.
func (a Automation) DesiredBaseline() (json.RawMessage, error) {
	out, err := json.Marshal(desiredBaselineView{
		Name:     a.Config.Name,
		Enabled:  a.Config.enabled(),
		Scope:    normaliseScope(a.Config.Scope),
		Workflow: a.workflowPayload(),
	})
	if err != nil {
		return nil, fmt.Errorf("encoding desired config: %w", err)
	}
	return out, nil
}
