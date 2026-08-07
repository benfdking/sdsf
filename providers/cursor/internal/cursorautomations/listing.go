package cursorautomations

import (
	"encoding/json"
	"fmt"
)

// LiveAutomation is one automation as the server currently holds it: the name
// callers match on to adopt an existing automation, the server's id, and the
// untouched object so a caller can compare whichever fields it manages.
type LiveAutomation struct {
	Name         string
	AutomationID string
	Raw          json.RawMessage
}

// ParseList reads a /list-automations response.
//
// It fails closed. A response with no "workflows" key is an error rather than
// an empty result, because reading a reshaped response as "this team has no
// automations" would let a caller re-create automations that already exist.
func ParseList(raw []byte) ([]LiveAutomation, error) {
	var resp struct {
		// A pointer distinguishes an absent key from an empty list.
		Workflows *[]struct {
			Workflow json.RawMessage `json:"workflow"`
		} `json:"workflows"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parsing list response: %w", err)
	}
	if resp.Workflows == nil {
		return nil, fmt.Errorf(`unexpected list response: missing "workflows"`)
	}

	automations := make([]LiveAutomation, 0, len(*resp.Workflows))
	for _, entry := range *resp.Workflows {
		var meta struct {
			Name         string `json:"name"`
			AutomationID string `json:"automationId"`
		}
		if len(entry.Workflow) > 0 {
			if err := json.Unmarshal(entry.Workflow, &meta); err != nil {
				return nil, fmt.Errorf("parsing listed automation: %w", err)
			}
		}
		automations = append(automations, LiveAutomation{
			Name:         meta.Name,
			AutomationID: meta.AutomationID,
			Raw:          entry.Workflow,
		})
	}
	return automations, nil
}
