package cursorautomations

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func ptr[T any](v T) *T { return &v }

// writeAutomation creates an automations/<name> dir with the given files.
func writeAutomation(root, name, yamlBody, prompt string) {
	dir := filepath.Join(root, name)
	Expect(os.MkdirAll(dir, 0o755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(dir, "automation.yaml"), []byte(yamlBody), 0o644)).To(Succeed())
	if prompt != "" {
		Expect(os.WriteFile(filepath.Join(dir, "prompt.md"), []byte(prompt), 0o644)).To(Succeed())
	}
}

var _ = Describe("Load", func() {
	var root string

	BeforeEach(func() {
		root = GinkgoT().TempDir()
	})

	It("parses config and prompt, trimming the trailing newline", func() {
		writeAutomation(root, "tea", `
name: Tea
scope: TEAM_VISIBLE
enabled: true
memoryEnabled: true
model: gpt-5.5-high
triggers:
  - cron: { cron: "0 8 * * 1" }
actions:
  - slack: {}
  - requestReviewers: {}
`, "Pick a tea.\n")

		automations, err := Load(root)
		Expect(err).NotTo(HaveOccurred())
		Expect(automations).To(HaveLen(1))
		Expect(automations[0].Config.Name).To(Equal("Tea"))
		Expect(automations[0].Prompt).To(Equal("Pick a tea."))
	})

	It("rejects an automation missing a required field", func() {
		writeAutomation(root, "bad", "name: Bad\nscope: TEAM_VISIBLE\nenabled: true\nmemoryEnabled: true\n", "do a thing")
		_, err := Load(root)
		Expect(err).To(MatchError(ContainSubstring("missing required field: model")))
	})

	It("requires enabled to be set explicitly", func() {
		writeAutomation(root, "bad", "name: Bad\nscope: TEAM_VISIBLE\nmodel: gpt-5.5-high\n", "do a thing")
		_, err := Load(root)
		Expect(err).To(MatchError(ContainSubstring("enabled")))
	})

	It("requires memoryEnabled to be set explicitly", func() {
		writeAutomation(root, "bad", "name: Bad\nscope: TEAM_VISIBLE\nenabled: true\nmodel: gpt-5.5-high\n", "do a thing")
		_, err := Load(root)
		Expect(err).To(MatchError(ContainSubstring("memoryEnabled")))
	})

	It("rejects an empty prompt", func() {
		writeAutomation(root, "bad", "name: Bad\nscope: TEAM_VISIBLE\nenabled: true\nmemoryEnabled: true\nmodel: gpt-5.5-high\n", "   \n")
		_, err := Load(root)
		Expect(err).To(MatchError(ContainSubstring("prompt.md is empty")))
	})

	It("rejects an unknown trigger type", func() {
		writeAutomation(root, "bad", `
name: Bad
scope: TEAM_VISIBLE
enabled: true
memoryEnabled: true
model: gpt-5.5-high
triggers:
  - notARealTrigger: {}
`, "do a thing")
		_, err := Load(root)
		Expect(err).To(MatchError(ContainSubstring(`unknown type "notARealTrigger"`)))
	})

	It("rejects two automations sharing a name (ambiguous for name matching)", func() {
		writeAutomation(root, "a", "name: Same\nscope: TEAM_VISIBLE\nenabled: true\nmemoryEnabled: true\nmodel: gpt-5.5-high\n", "x")
		writeAutomation(root, "b", "name: Same\nscope: TEAM_VISIBLE\nenabled: true\nmemoryEnabled: true\nmodel: gpt-5.5-high\n", "y")
		_, err := Load(root)
		Expect(err).To(MatchError(ContainSubstring("duplicate automation name")))
	})
})

var _ = Describe("RenderCreatePayload", func() {
	It("normalises a short scope and sources the prompt from prompt.md", func() {
		a := Automation{
			Prompt: "Pick a tea.",
			Config: Config{
				Name:          "Tea",
				Scope:         "TEAM_VISIBLE",
				Model:         "gpt-5.5-high",
				Enabled:       ptr(true),
				MemoryEnabled: ptr(true),
				GitConfig:     &gitConfig{Repos: []string{"https://github.com/org/repo"}, Branch: "master"},
			},
		}

		req := a.RenderCreatePayload(12340957).(createRequest)
		Expect(req.Scope).To(Equal("AUTOMATION_SCOPE_TEAM_VISIBLE"))
		Expect(req.Enabled).To(BeTrue())
		Expect(req.TeamID).To(Equal(int64(12340957)))
		Expect(req.Workflow.MemoryEnabled).To(BeTrue())
		Expect(req.Workflow.Prompts).To(HaveLen(1))
		Expect(req.Workflow.Prompts[0].Prompt).To(HavePrefix(managedPromptHeader))
		Expect(req.Workflow.Prompts[0].Prompt).To(ContainSubstring("Pick a tea."))
		Expect(req.Workflow.GitConfig.Repos).To(ConsistOf("https://github.com/org/repo"))
		Expect(req.Workflow.GitConfig.Branch).To(Equal("master"))
	})

	It("leaves an already-qualified scope untouched", func() {
		a := Automation{Config: Config{Scope: "AUTOMATION_SCOPE_TEAM_EDITABLE_USER"}}
		Expect(a.RenderCreatePayload(0).(createRequest).Scope).To(Equal("AUTOMATION_SCOPE_TEAM_EDITABLE_USER"))
	})
})

var _ = DescribeTable("parseCreatedID reads the id from whichever documented position is present",
	func(body, want string) {
		id, err := parseCreatedID([]byte(body))
		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(Equal(want))
	},
	Entry("nested workflow.workflow", `{"workflow":{"workflow":{"automationId":"deep"}}}`, "deep"),
	Entry("workflow.automationId", `{"workflow":{"automationId":"mid"}}`, "mid"),
	Entry("top-level automationId", `{"automationId":"top"}`, "top"),
	Entry("prefers the deepest when several are set",
		`{"automationId":"top","workflow":{"automationId":"mid","workflow":{"automationId":"deep"}}}`, "deep"),
	Entry("empty when none present, so Create fails loudly", `{"workflow":{}}`, ""),
)

var _ = Describe("ParseList", func() {
	It("extracts the name and id of each live automation", func() {
		live, err := ParseList([]byte(`{"workflows":[
			{"workflow":{"name":"Tea","automationId":"id-1"}},
			{"workflow":{"name":"Coffee","automationId":"id-2"}}
		]}`))
		Expect(err).NotTo(HaveOccurred())
		Expect(live).To(HaveLen(2))
		Expect(live[0].Name).To(Equal("Tea"))
		Expect(live[0].AutomationID).To(Equal("id-1"))
	})

	It("fails closed when the response is missing the workflows field", func() {
		_, err := ParseList([]byte(`{"items":[]}`))
		Expect(err).To(MatchError(ContainSubstring("workflows")))
	})

	It("treats an explicit empty workflows list as zero automations, not an error", func() {
		live, err := ParseList([]byte(`{"workflows":[]}`))
		Expect(err).NotTo(HaveOccurred())
		Expect(live).To(BeEmpty())
	})
})

var _ = Describe("State", func() {
	var path string

	BeforeEach(func() {
		path = filepath.Join(GinkgoT().TempDir(), ".cursor-automations-state.json")
	})

	It("round-trips recorded ids through save and load", func() {
		s := LoadState(path)
		s.Record("tea", "id-1")
		Expect(s.Save(path)).To(Succeed())

		id, ok := LoadState(path).ID("tea")
		Expect(ok).To(BeTrue())
		Expect(id).To(Equal("id-1"))
	})

	It("treats a missing file as empty state, so a first run adopts rather than fails", func() {
		_, ok := LoadState(filepath.Join(GinkgoT().TempDir(), "nope.json")).ID("tea")
		Expect(ok).To(BeFalse())
	})

	It("treats a corrupt file as empty state", func() {
		Expect(os.WriteFile(path, []byte("{not json"), 0o644)).To(Succeed())
		_, ok := LoadState(path).ID("tea")
		Expect(ok).To(BeFalse())
	})

	It("forgets a stale entry", func() {
		s := LoadState(path)
		s.Record("tea", "id-1")
		s.Forget("tea")
		_, ok := s.ID("tea")
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("APIError", func() {
	// The verbatim 400 body Cursor returned when the token owner's account had no
	// Slack connected (from the first live sync run). Kept as a fixture so the
	// classifier keeps matching the real message if this test is revisited.
	const slackNotConnectedBody = `{"error":{"message":"Cannot resolve Slack channel: You must connect your Slack account first. Alternatively, use a channel ID (e.g., C0123456789) instead of a channel name.","details":[{"error":"ERROR_BAD_REQUEST"}]}}`

	It("classifies 401/403 as an auth failure", func() {
		Expect((&APIError{StatusCode: 401}).IsAuth()).To(BeTrue())
		Expect((&APIError{StatusCode: 403}).IsAuth()).To(BeTrue())
		Expect((&APIError{StatusCode: 400}).IsAuth()).To(BeFalse())
	})

	It("classifies the Slack-not-connected 400 from its message", func() {
		Expect((&APIError{StatusCode: 400, Body: slackNotConnectedBody}).IsSlackNotConnected()).To(BeTrue())
	})

	It("matches case-insensitively, so a casing tweak on Cursor's side still classifies", func() {
		Expect((&APIError{StatusCode: 400, Body: `{"message":"You Must Connect Your Slack Account First."}`}).IsSlackNotConnected()).To(BeTrue())
	})

	It("does not treat an unrelated 400 as Slack-not-connected", func() {
		Expect((&APIError{StatusCode: 400, Body: `{"error":{"message":"something else"}}`}).IsSlackNotConnected()).To(BeFalse())
	})

	It("does not treat the Slack message on a non-400 status as Slack-not-connected", func() {
		Expect((&APIError{StatusCode: 500, Body: slackNotConnectedBody}).IsSlackNotConnected()).To(BeFalse())
	})
})

var _ = Describe("ResolveTarget", func() {
	It("uses the recorded id when it's still live", func() {
		s := &State{Automations: map[string]TrackedAutomation{"tea": {AutomationID: "id-1"}}}
		id, source := ResolveTarget("tea", "Tea", s, map[string]string{"Tea": "id-1"}, map[string]bool{"id-1": true})
		Expect(source).To(Equal("state"))
		Expect(id).To(Equal("id-1"))
	})

	It("ignores a recorded id that's no longer live and adopts the live one by name", func() {
		s := &State{Automations: map[string]TrackedAutomation{"tea": {AutomationID: "dead"}}}
		id, source := ResolveTarget("tea", "Tea", s, map[string]string{"Tea": "live-1"}, map[string]bool{"live-1": true})
		Expect(source).To(Equal("adopt"))
		Expect(id).To(Equal("live-1"))
	})

	It("adopts a live automation by name when state has no id", func() {
		s := &State{Automations: map[string]TrackedAutomation{}}
		id, source := ResolveTarget("tea", "Tea", s, map[string]string{"Tea": "live-1"}, map[string]bool{"live-1": true})
		Expect(source).To(Equal("adopt"))
		Expect(id).To(Equal("live-1"))
	})

	It("creates when neither state nor the live list knows it", func() {
		s := &State{Automations: map[string]TrackedAutomation{}}
		id, source := ResolveTarget("tea", "Tea", s, map[string]string{}, map[string]bool{})
		Expect(source).To(Equal("create"))
		Expect(id).To(BeEmpty())
	})
})
