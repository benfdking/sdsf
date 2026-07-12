package cursorautomations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// testAutomation builds a desired automation the way Load would (YAML-shaped
// maps, enum names as strings).
func testAutomation(dir, name, model string) Automation {
	return Automation{
		Dir:    dir,
		Prompt: "Pick a tea.",
		Config: Config{
			Name:          name,
			Scope:         "TEAM_VISIBLE",
			Model:         model,
			Enabled:       ptr(true),
			MemoryEnabled: ptr(true),
			Triggers: []map[string]any{
				{"slackTrigger": map[string]any{
					"channels":                    []any{"C1"},
					"topLevelOnly":                true,
					"slackCompletionReactionMode": "SLACK_COMPLETION_REACTION_MODE_CUSTOM",
				}},
			},
			Actions:               []map[string]any{{"readSlack": map[string]any{}}},
			SlackNotifiedChannels: []string{"C1"},
		},
	}
}

// liveAutomationJSON is a live automation in the server's encoding, deliberately
// messier than what we send: short-form scope, protobuf enum ints, the legacy
// singular "channel" duplicate, timestamps, and server-assigned agentOptions.
func liveAutomationJSON(name, id, model string) string {
	return fmt.Sprintf(`{
		"name": %q, "enabled": true, "scope": "team_visible", "automationId": %q,
		"createdAt": "1782742636", "updatedAt": "1782743645",
		"workflow": {
			"prompts": [{"prompt": "managed prompt"}],
			"model": %q,
			"triggers": [{"slackTrigger": {"channels": ["C1"], "channel": "C1", "topLevelOnly": true, "slackCompletionReactionMode": 1}}],
			"actions": [{"readSlack": {}}],
			"memoryEnabled": true,
			"slackNotifiedChannels": ["C1"],
			"agentOptions": {"environmentPublicId": "env-1"}
		}
	}`, name, id, model)
}

func liveListJSON(automations ...string) json.RawMessage {
	entries := make([]string, len(automations))
	for i, a := range automations {
		entries[i] = fmt.Sprintf(`{"workflow": %s, "userId": 305404700, "ownerName": "Nicole Hussein"}`, a)
	}
	return json.RawMessage(`{"workflows":[` + strings.Join(entries, ",") + `]}`)
}

// jsonTags returns the sorted JSON field names of a struct type, so the guard
// tests below can compare the managed-field allowlists against the payload
// structs they must mirror.
func jsonTags(t reflect.Type) []string {
	var tags []string
	for i := 0; i < t.NumField(); i++ {
		tag := strings.Split(t.Field(i).Tag.Get("json"), ",")[0]
		if tag != "" && tag != "-" {
			tags = append(tags, tag)
		}
	}
	sort.Strings(tags)
	return tags
}

// These guards pin the baseline machinery to the payload structs: a field added
// to the update payload without extending the managed-field projection would
// silently escape drift detection (UI edits to it never re-asserted), and one
// missing from the desired-baseline view would make repo edits to it diff as
// empty and never deploy.
var _ = Describe("baseline field guards", func() {
	It("projects every workflow payload field, and nothing else", func() {
		want := jsonTags(reflect.TypeOf(innerWorkflow{}))
		got := append([]string{}, managedWorkflowFields...)
		sort.Strings(got)
		Expect(got).To(Equal(want))
	})

	It("projects every top-level update field except automationId and the workflow itself", func() {
		want := jsonTags(reflect.TypeOf(updateRequest{}))
		var filtered []string
		for _, tag := range want {
			if tag != "automationId" && tag != "workflow" {
				filtered = append(filtered, tag)
			}
		}
		got := append([]string{}, managedTopLevelFields...)
		sort.Strings(got)
		Expect(got).To(Equal(filtered))
	})

	It("records every update field except automationId in the desired baseline", func() {
		updateTags := jsonTags(reflect.TypeOf(updateRequest{}))
		var want []string
		for _, tag := range updateTags {
			if tag != "automationId" {
				want = append(want, tag)
			}
		}
		Expect(jsonTags(reflect.TypeOf(desiredBaselineView{}))).To(Equal(want))
	})
})

var _ = Describe("truncate", func() {
	It("cuts at a rune boundary so multi-byte characters don't render as mojibake", func() {
		s := strings.Repeat("a", 119) + "—" // em dash: 3 bytes, straddles the 120-byte limit
		got := truncate(s, 120)
		Expect(utf8.ValidString(got)).To(BeTrue())
		Expect(got).To(Equal(strings.Repeat("a", 119) + "…"))
	})

	It("leaves short strings untouched", func() {
		Expect(truncate("short", 120)).To(Equal("short"))
	})
})

var _ = Describe("ProjectLive", func() {
	It("keeps only the managed fields, in the server's own encoding", func() {
		projected, err := ProjectLive(json.RawMessage(liveAutomationJSON("Tea", "id-1", "gpt-5.5-high")))
		Expect(err).NotTo(HaveOccurred())

		var m map[string]any
		Expect(json.Unmarshal(projected, &m)).To(Succeed())
		Expect(m).To(HaveLen(4))
		Expect(m["name"]).To(Equal("Tea"))
		Expect(m["enabled"]).To(Equal(true))
		Expect(m["scope"]).To(Equal("team_visible"))

		wf, ok := m["workflow"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(wf).To(HaveKey("prompts"))
		Expect(wf).To(HaveKey("model"))
		Expect(wf).To(HaveKey("triggers"))
		Expect(wf).To(HaveKey("actions"))
		Expect(wf).To(HaveKey("memoryEnabled"))
		Expect(wf).To(HaveKey("slackNotifiedChannels"))
		// Server-side keys can never surface as drift: they're projected away.
		Expect(wf).NotTo(HaveKey("agentOptions"))
		Expect(m).NotTo(HaveKey("automationId"))
		Expect(m).NotTo(HaveKey("createdAt"))
		Expect(m).NotTo(HaveKey("updatedAt"))

		// Values stay verbatim: the enum int is not mapped to a name.
		trigger := wf["triggers"].([]any)[0].(map[string]any)["slackTrigger"].(map[string]any)
		Expect(trigger["slackCompletionReactionMode"]).To(Equal(float64(1)))
		Expect(trigger["channel"]).To(Equal("C1"))
	})

	It("is deterministic, so identical live states always compare equal", func() {
		raw := json.RawMessage(liveAutomationJSON("Tea", "id-1", "gpt-5.5-high"))
		first, err := ProjectLive(raw)
		Expect(err).NotTo(HaveOccurred())
		second, err := ProjectLive(raw)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(first)).To(Equal(string(second)))
	})

	It("fails on an unparseable automation", func() {
		_, err := ProjectLive(json.RawMessage(`"not an object"`))
		Expect(err).To(MatchError(ContainSubstring("parsing live automation")))
	})
})

var _ = Describe("DesiredBaseline", func() {
	It("renders the update-shaped payload without an automationId", func() {
		b, err := testAutomation("tea", "Tea", "gpt-5.5-high").DesiredBaseline()
		Expect(err).NotTo(HaveOccurred())

		var m map[string]any
		Expect(json.Unmarshal(b, &m)).To(Succeed())
		Expect(m["name"]).To(Equal("Tea"))
		Expect(m["enabled"]).To(Equal(true))
		Expect(m["scope"]).To(Equal("AUTOMATION_SCOPE_TEAM_VISIBLE"))
		Expect(m).NotTo(HaveKey("automationId"))

		wf := m["workflow"].(map[string]any)
		prompt := wf["prompts"].([]any)[0].(map[string]any)["prompt"].(string)
		Expect(prompt).To(HavePrefix(managedPromptHeader))
	})

	It("fails on a config value JSON can't encode", func() {
		a := testAutomation("tea", "Tea", "gpt-5.5-high")
		a.Config.Triggers = []map[string]any{{"cron": math.Inf(1)}}
		_, err := a.DesiredBaseline()
		Expect(err).To(MatchError(ContainSubstring("encoding desired config")))
	})
})

var _ = Describe("DiffJSON", func() {
	It("returns no diffs for equal documents regardless of key order", func() {
		diffs, err := DiffJSON(
			json.RawMessage(`{"a":1,"b":{"c":true}}`),
			json.RawMessage(`{"b":{"c":true},"a":1}`),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(diffs).To(BeEmpty())
	})

	It("reports a nested change with its full path", func() {
		diffs, err := DiffJSON(
			json.RawMessage(`{"workflow":{"triggers":[{"slackTrigger":{"topLevelOnly":true}}]}}`),
			json.RawMessage(`{"workflow":{"triggers":[{"slackTrigger":{"topLevelOnly":false}}]}}`),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(diffs).To(HaveLen(1))
		Expect(diffs[0].Path).To(Equal("workflow.triggers[0].slackTrigger.topLevelOnly"))
		Expect(diffs[0].String()).To(Equal("workflow.triggers[0].slackTrigger.topLevelOnly: true → false"))
	})

	It("renders added and removed keys as (unset)", func() {
		diffs, err := DiffJSON(
			json.RawMessage(`{"model":"gpt-5.5-high"}`),
			json.RawMessage(`{"memoryEnabled":true}`),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(diffs).To(HaveLen(2))
		Expect(diffs[0].String()).To(Equal(`memoryEnabled: (unset) → true`))
		Expect(diffs[1].String()).To(Equal(`model: "gpt-5.5-high" → (unset)`))
	})

	It("diffs a resized array as a whole rather than per shifted index", func() {
		diffs, err := DiffJSON(
			json.RawMessage(`{"triggers":[{"cron":{}}]}`),
			json.RawMessage(`{"triggers":[{"cron":{}},{"linear":{}}]}`),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(diffs).To(HaveLen(1))
		Expect(diffs[0].Path).To(Equal("triggers"))
	})

	It("reports a reshaped value (object to scalar) at its path", func() {
		diffs, err := DiffJSON(
			json.RawMessage(`{"gitConfig":{"branch":"main"}}`),
			json.RawMessage(`{"gitConfig":"main"}`),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(diffs).To(HaveLen(1))
		Expect(diffs[0].Path).To(Equal("gitConfig"))
	})
})

var _ = Describe("State baselines", func() {
	var path string

	BeforeEach(func() {
		path = filepath.Join(GinkgoT().TempDir(), ".cursor-automations-state.json")
	})

	It("round-trips baselines through save and load", func() {
		s := LoadState(path)
		s.Record("tea", "id-1")
		s.RecordBaseline("tea", json.RawMessage(`{"a":1}`), json.RawMessage(`{"b":2}`))
		Expect(s.Save(path)).To(Succeed())

		payload, live, ok := LoadState(path).Baseline("tea")
		Expect(ok).To(BeTrue())
		Expect(string(payload)).To(MatchJSON(`{"a":1}`))
		Expect(string(live)).To(MatchJSON(`{"b":2}`))
	})

	It("resets baselines when a new id is recorded (new identity, stale snapshots)", func() {
		s := LoadState(path)
		s.Record("tea", "id-1")
		s.RecordBaseline("tea", json.RawMessage(`{}`), json.RawMessage(`{}`))
		s.Record("tea", "id-2")
		_, _, ok := s.Baseline("tea")
		Expect(ok).To(BeFalse())
	})

	It("treats a legacy id-only state entry as having no baseline", func() {
		s := LoadState(path)
		s.Record("tea", "id-1")
		_, _, ok := s.Baseline("tea")
		Expect(ok).To(BeFalse())
	})

	It("preserves the id when recording a baseline", func() {
		s := LoadState(path)
		s.Record("tea", "id-1")
		s.RecordBaseline("tea", json.RawMessage(`{}`), json.RawMessage(`{}`))
		id, ok := s.ID("tea")
		Expect(ok).To(BeTrue())
		Expect(id).To(Equal("id-1"))
	})
})

var _ = Describe("Sync", func() {
	var (
		statePath string
		state     *State
		out       *bytes.Buffer
		ctx       context.Context
	)

	BeforeEach(func() {
		statePath = filepath.Join(GinkgoT().TempDir(), ".cursor-automations-state.json")
		state = LoadState(statePath)
		out = &bytes.Buffer{}
		ctx = context.Background()
	})

	// establishBaseline seeds state as if a previous run had applied this exact
	// automation and snapshotted this exact live JSON.
	establishBaseline := func(a Automation, id, liveJSON string) {
		desired, err := a.DesiredBaseline()
		Expect(err).NotTo(HaveOccurred())
		liveView, err := ProjectLive(json.RawMessage(liveJSON))
		Expect(err).NotTo(HaveOccurred())
		state.Record(a.Dir, id)
		state.RecordBaseline(a.Dir, desired, liveView)
	}

	It("skips the write entirely when nothing changed", func() {
		a := testAutomation("tea", "Tea", "gpt-5.5-high")
		liveJSON := liveAutomationJSON("Tea", "id-1", "gpt-5.5-high")
		establishBaseline(a, "id-1", liveJSON)
		fake := &fakeAPI{listResponses: []json.RawMessage{liveListJSON(liveJSON)}}

		res := Sync(ctx, fake, []Automation{a}, state, statePath, out)

		Expect(res.Failures).To(BeZero())
		Expect(res.Unchanged).To(Equal(1))
		Expect(fake.updateCalls).To(BeEmpty())
		Expect(fake.createCalls).To(BeEmpty())
		Expect(fake.listCalls).To(Equal(1), "no writes means no post-write list either")
		Expect(out.String()).To(ContainSubstring("unchanged, skipping"))
	})

	It("updates with a field-level diff when the repo config changed", func() {
		previous := testAutomation("tea", "Tea", "gpt-5.5-high")
		liveJSON := liveAutomationJSON("Tea", "id-1", "gpt-5.5-high")
		establishBaseline(previous, "id-1", liveJSON)

		changed := testAutomation("tea", "Tea", "gpt-6")
		postJSON := liveAutomationJSON("Tea", "id-1", "gpt-6")
		fake := &fakeAPI{listResponses: []json.RawMessage{liveListJSON(liveJSON), liveListJSON(postJSON)}}

		res := Sync(ctx, fake, []Automation{changed}, state, statePath, out)

		Expect(res.Failures).To(BeZero())
		Expect(res.Updated).To(Equal(1))
		Expect(fake.updateCalls).To(Equal([]string{"tea:id-1"}))
		Expect(out.String()).To(ContainSubstring("config changed:"))
		Expect(out.String()).To(ContainSubstring(`workflow.model: "gpt-5.5-high" → "gpt-6"`))

		// Baselines are refreshed from the new desired config and the post-write list.
		payload, live, ok := LoadState(statePath).Baseline("tea")
		Expect(ok).To(BeTrue())
		wantDesired, err := changed.DesiredBaseline()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(payload)).To(MatchJSON(string(wantDesired)))
		wantLive, err := ProjectLive(json.RawMessage(postJSON))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(live)).To(MatchJSON(string(wantLive)))
	})

	It("re-asserts and reports the drifted fields when the live automation was edited outside the repo", func() {
		a := testAutomation("tea", "Tea", "gpt-5.5-high")
		baselineJSON := liveAutomationJSON("Tea", "id-1", "gpt-5.5-high")
		establishBaseline(a, "id-1", baselineJSON)

		driftedJSON := liveAutomationJSON("Tea", "id-1", "gpt-6")
		fake := &fakeAPI{listResponses: []json.RawMessage{liveListJSON(driftedJSON), liveListJSON(baselineJSON)}}

		res := Sync(ctx, fake, []Automation{a}, state, statePath, out)

		Expect(res.Failures).To(BeZero())
		Expect(res.Updated).To(Equal(1))
		Expect(fake.updateCalls).To(Equal([]string{"tea:id-1"}))
		Expect(out.String()).To(ContainSubstring("drifted"))
		Expect(out.String()).To(ContainSubstring(`workflow.model: "gpt-5.5-high" → "gpt-6"`))
	})

	It("applies once to establish a baseline when state has an id but no baseline", func() {
		a := testAutomation("tea", "Tea", "gpt-5.5-high")
		state.Record("tea", "id-1")
		liveJSON := liveAutomationJSON("Tea", "id-1", "gpt-5.5-high")
		fake := &fakeAPI{listResponses: []json.RawMessage{liveListJSON(liveJSON)}}

		res := Sync(ctx, fake, []Automation{a}, state, statePath, out)

		Expect(res.Failures).To(BeZero())
		Expect(res.Updated).To(Equal(1))
		Expect(out.String()).To(ContainSubstring("no baseline recorded"))
		_, _, ok := LoadState(statePath).Baseline("tea")
		Expect(ok).To(BeTrue())
	})

	It("creates without a follow-up update, so a new automation is written exactly once", func() {
		a := testAutomation("tea", "Tea", "gpt-5.5-high")
		createdJSON := liveAutomationJSON("Tea", "new-1", "gpt-5.5-high")
		fake := &fakeAPI{
			createdID:     "new-1",
			listResponses: []json.RawMessage{liveListJSON(), liveListJSON(createdJSON)},
		}

		res := Sync(ctx, fake, []Automation{a}, state, statePath, out)

		Expect(res.Failures).To(BeZero())
		Expect(res.Created).To(Equal(1))
		Expect(fake.createCalls).To(Equal([]string{"tea"}))
		Expect(fake.updateCalls).To(BeEmpty(), "create carries the full config; a follow-up update is a duplicate write")

		loaded := LoadState(statePath)
		id, ok := loaded.ID("tea")
		Expect(ok).To(BeTrue())
		Expect(id).To(Equal("new-1"))
		_, _, ok = loaded.Baseline("tea")
		Expect(ok).To(BeTrue())
	})

	It("adopts a live automation by name and applies once to establish a baseline", func() {
		a := testAutomation("tea", "Tea", "gpt-5.5-high")
		liveJSON := liveAutomationJSON("Tea", "live-1", "gpt-5.5-high")
		fake := &fakeAPI{listResponses: []json.RawMessage{liveListJSON(liveJSON)}}

		res := Sync(ctx, fake, []Automation{a}, state, statePath, out)

		Expect(res.Failures).To(BeZero())
		Expect(res.Updated).To(Equal(1))
		Expect(fake.updateCalls).To(Equal([]string{"tea:live-1"}))
		Expect(out.String()).To(ContainSubstring("adopted by name"))
	})

	It("keeps the old baseline when an update fails, and classifies the error", func() {
		previous := testAutomation("tea", "Tea", "gpt-5.5-high")
		liveJSON := liveAutomationJSON("Tea", "id-1", "gpt-5.5-high")
		establishBaseline(previous, "id-1", liveJSON)
		oldPayload, _, _ := state.Baseline("tea")

		changed := testAutomation("tea", "Tea", "gpt-6")
		fake := &fakeAPI{
			listResponses: []json.RawMessage{liveListJSON(liveJSON)},
			updateErr:     &APIError{StatusCode: 401, Body: "expired"},
		}

		res := Sync(ctx, fake, []Automation{changed}, state, statePath, out)

		Expect(res.Failures).To(Equal(1))
		Expect(res.AuthFailed).To(BeTrue())
		payload, _, ok := state.Baseline("tea")
		Expect(ok).To(BeTrue())
		Expect(string(payload)).To(Equal(string(oldPayload)))
		Expect(fake.listCalls).To(Equal(1), "nothing was written, so no post-write list")
	})

	It("fails everything when the initial list fails, classifying auth errors", func() {
		a := testAutomation("tea", "Tea", "gpt-5.5-high")
		fake := &fakeAPI{listErr: &APIError{StatusCode: 401, Body: "expired"}}

		res := Sync(ctx, fake, []Automation{a}, state, statePath, out)

		Expect(res.Failures).To(Equal(1))
		Expect(res.AuthFailed).To(BeTrue())
		Expect(fake.createCalls).To(BeEmpty())
		Expect(fake.updateCalls).To(BeEmpty())
	})

	It("warns and leaves the baseline unestablished when the post-write list fails", func() {
		a := testAutomation("tea", "Tea", "gpt-5.5-high")
		state.Record("tea", "id-1")
		liveJSON := liveAutomationJSON("Tea", "id-1", "gpt-5.5-high")
		fake := &fakeAPI{
			listResponses: []json.RawMessage{liveListJSON(liveJSON)},
			listErr:       &APIError{StatusCode: 500, Body: "flake"},
			listErrOnCall: 2,
		}

		res := Sync(ctx, fake, []Automation{a}, state, statePath, out)

		Expect(res.Failures).To(BeZero(), "the write itself succeeded")
		Expect(res.Updated).To(Equal(1))
		Expect(out.String()).To(ContainSubstring("could not list automations to capture baselines"))
		_, _, ok := LoadState(statePath).Baseline("tea")
		Expect(ok).To(BeFalse(), "next run re-applies rather than trusting a stale snapshot")
	})

	It("warns when a written automation is missing from the post-write list", func() {
		a := testAutomation("tea", "Tea", "gpt-5.5-high")
		fake := &fakeAPI{
			createdID:     "new-1",
			listResponses: []json.RawMessage{liveListJSON()},
		}

		res := Sync(ctx, fake, []Automation{a}, state, statePath, out)

		Expect(res.Failures).To(BeZero())
		Expect(res.Created).To(Equal(1))
		Expect(out.String()).To(ContainSubstring("missing from post-write list"))
		_, _, ok := LoadState(statePath).Baseline("tea")
		Expect(ok).To(BeFalse())
	})
})
