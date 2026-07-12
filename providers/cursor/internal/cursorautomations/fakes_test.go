package cursorautomations

import (
	"context"
	"encoding/json"
	"fmt"
)

// fakeAPI is a programmable API double that records every call. List responses
// are consumed in order (the last repeats), so tests can serve different
// pre-write and post-write lists.
type fakeAPI struct {
	listResponses []json.RawMessage
	listErr       error
	listErrOnCall int // 1-based call number listErr fires on; 0 means every call
	listCalls     int

	createdID   string
	createErr   error
	createCalls []string // automation dirs, in call order

	updateErr   error
	updateCalls []string // "<dir>:<id>", in call order
}

func (f *fakeAPI) List(ctx context.Context) (json.RawMessage, error) {
	f.listCalls++
	if f.listErr != nil && (f.listErrOnCall == 0 || f.listErrOnCall == f.listCalls) {
		return nil, f.listErr
	}
	if len(f.listResponses) == 0 {
		return json.RawMessage(`{"workflows":[]}`), nil
	}
	i := f.listCalls - 1
	if i >= len(f.listResponses) {
		i = len(f.listResponses) - 1
	}
	return f.listResponses[i], nil
}

func (f *fakeAPI) Create(ctx context.Context, a Automation) (string, error) {
	f.createCalls = append(f.createCalls, a.Dir)
	if f.createErr != nil {
		return "", f.createErr
	}
	return f.createdID, nil
}

func (f *fakeAPI) Update(ctx context.Context, a Automation, automationID string) error {
	f.updateCalls = append(f.updateCalls, fmt.Sprintf("%s:%s", a.Dir, automationID))
	return f.updateErr
}
