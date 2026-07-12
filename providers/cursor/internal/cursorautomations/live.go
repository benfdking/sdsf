package cursorautomations

import (
	"encoding/json"
	"fmt"
)

// LiveAutomation is what sync needs from a listed automation: the name
// (adoption key), the server id, and the raw automation object so drift
// detection can project it onto the managed fields.
type LiveAutomation struct {
	Name         string
	AutomationID string
	Raw          json.RawMessage
}

// ParseList extracts automations from a /list-automations response. It fails
// closed: a response missing "workflows" is an error, not an empty list, so a
// reshaped response can't be read as "no automations" and trigger re-creation.
func ParseList(raw []byte) ([]LiveAutomation, error) {
	var resp struct {
		Workflows *[]struct {
			Workflow json.RawMessage `json:"workflow"`
		} `json:"workflows"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	if resp.Workflows == nil {
		return nil, fmt.Errorf("unexpected list response: missing \"workflows\"")
	}

	live := make([]LiveAutomation, 0, len(*resp.Workflows))
	for _, w := range *resp.Workflows {
		var meta struct {
			Name         string `json:"name"`
			AutomationID string `json:"automationId"`
		}
		if len(w.Workflow) > 0 {
			if err := json.Unmarshal(w.Workflow, &meta); err != nil {
				return nil, fmt.Errorf("parsing listed automation: %w", err)
			}
		}
		live = append(live, LiveAutomation{
			Name:         meta.Name,
			AutomationID: meta.AutomationID,
			Raw:          w.Workflow,
		})
	}
	return live, nil
}
