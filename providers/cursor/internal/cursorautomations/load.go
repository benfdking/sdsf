package cursorautomations

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load discovers and parses every <name>/automation.yaml under dir, validating
// each. A single bad automation fails the whole load so CI stays red.
func Load(dir string) ([]Automation, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading automations directory: %w", err)
	}

	var automations []Automation
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		configPath := filepath.Join(dir, e.Name(), "automation.yaml")
		if _, err := os.Stat(configPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat %s: %w", configPath, err)
		}

		automation, err := parse(dir, e.Name())
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}

		automations = append(automations, automation)
	}

	// Names must be unique so adopting an existing automation by name (when state
	// has no recorded id) is unambiguous.
	byName := map[string]string{}
	for _, a := range automations {
		if prev, ok := byName[a.Config.Name]; ok {
			return nil, fmt.Errorf("duplicate automation name %q in %q and %q", a.Config.Name, prev, a.Dir)
		}
		byName[a.Config.Name] = a.Dir
	}

	return automations, nil
}

func parse(dir, name string) (Automation, error) {
	configData, err := os.ReadFile(filepath.Join(dir, name, "automation.yaml"))
	if err != nil {
		return Automation{}, fmt.Errorf("reading automation.yaml: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return Automation{}, fmt.Errorf("parsing automation.yaml: %w", err)
	}

	// A missing prompt.md reads as empty and is rejected by validate (a prompt is
	// required); only other read errors are real failures.
	promptData, err := os.ReadFile(filepath.Join(dir, name, "prompt.md"))
	if err != nil && !os.IsNotExist(err) {
		return Automation{}, fmt.Errorf("reading prompt.md: %w", err)
	}

	automation := Automation{
		Dir:    name,
		Config: config,
		Prompt: strings.TrimRight(string(promptData), "\n"),
	}

	if err := automation.validate(); err != nil {
		return Automation{}, err
	}

	return automation, nil
}
