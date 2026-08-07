package cursorautomations

// managedPromptHeader is prepended to every prompt this package pushes, so
// somebody reading an automation in the Cursor dashboard can see it is managed
// elsewhere. It is written only: nothing here parses it back, so changing the
// wording is safe apart from producing one prompt update per automation.
const managedPromptHeader = "<!-- Managed by Terraform (cursor_automation). Don't edit here; changes are overwritten on the next apply. -->"

// creationSource tags who created an automation. The API accepts the same
// value the dashboard sends.
const creationSource = "AUTOMATION_CREATION_SOURCE_PORTAL_WEB"

// workflow is the nested object carrying an automation's actual configuration.
//
// Both create and update send the complete desired state, and the server
// replaces what it holds. That drives the omitempty choices: an omitted slice
// clears the corresponding list, which is what a caller that set no triggers
// means. memoryEnabled deliberately keeps no omitempty — dropping a false would
// leave the server applying its own default, so turning memory off would not
// stick.
type workflow struct {
	Prompts               []promptStep     `json:"prompts,omitempty"`
	Model                 string           `json:"model,omitempty"`
	Triggers              []map[string]any `json:"triggers,omitempty"`
	Actions               []map[string]any `json:"actions,omitempty"`
	MemoryEnabled         bool             `json:"memoryEnabled"`
	SlackNotifiedChannels []string         `json:"slackNotifiedChannels,omitempty"`
	GitConfig             *GitConfig       `json:"gitConfig,omitempty"`
}

// promptStep is one entry in an automation's prompt list.
type promptStep struct {
	Prompt string `json:"prompt"`
}

// createRequest is the body of /create-automation. The server assigns the id,
// so none is sent.
type createRequest struct {
	Name           string   `json:"name"`
	Enabled        bool     `json:"enabled"`
	Workflow       workflow `json:"workflow"`
	Scope          string   `json:"scope"`
	TeamID         int64    `json:"teamId"`
	CreationSource string   `json:"creationSource"`
}

// updateRequest is the body of /update-automation, which targets an existing
// automation by id.
type updateRequest struct {
	Name         string   `json:"name"`
	Enabled      bool     `json:"enabled"`
	Workflow     workflow `json:"workflow"`
	AutomationID string   `json:"automationId"`
	Scope        string   `json:"scope"`
}

// promptWithHeader is the prompt as the server should store it.
func (a Automation) promptWithHeader() string {
	return managedPromptHeader + "\n\n" + a.Prompt
}

// buildWorkflow builds the configuration object shared by create and update.
func (a Automation) buildWorkflow() workflow {
	return workflow{
		Prompts:               []promptStep{{Prompt: a.promptWithHeader()}},
		Model:                 a.Config.Model,
		Triggers:              a.Config.Triggers,
		Actions:               a.Config.Actions,
		MemoryEnabled:         a.Config.isMemoryEnabled(),
		SlackNotifiedChannels: a.Config.SlackNotifiedChannels,
		GitConfig:             a.Config.GitConfig,
	}
}

func (a Automation) toCreateRequest(teamID int64) createRequest {
	return createRequest{
		Name:           a.Config.Name,
		Enabled:        a.Config.isEnabled(),
		Workflow:       a.buildWorkflow(),
		Scope:          qualifiedScope(a.Config.Scope),
		TeamID:         teamID,
		CreationSource: creationSource,
	}
}

func (a Automation) toUpdateRequest(automationID string) updateRequest {
	return updateRequest{
		Name:         a.Config.Name,
		Enabled:      a.Config.isEnabled(),
		Workflow:     a.buildWorkflow(),
		AutomationID: automationID,
		Scope:        qualifiedScope(a.Config.Scope),
	}
}

// RenderCreatePayload returns the body that a create would send, for callers
// that want to show a plan without performing one. It stands in for update too:
// which of the two runs is decided at apply time by matching the name against
// the automations the account can already see.
func (a Automation) RenderCreatePayload(teamID int64) any {
	return a.toCreateRequest(teamID)
}
