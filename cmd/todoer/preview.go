package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/inful/todoer/pkg/core"
)

func cmdPreview(templateFile, date, todosFile, todosString, customVars string, config *Config) error {
	if date == "" {
		date = time.Now().Format(core.DateFormat)
	}

	var todosContent string
	if todosString != "" {
		todosContent = todosString
	} else if todosFile != "" {
		content, err := os.ReadFile(todosFile)
		if err != nil {
			return fmt.Errorf("failed to read todos file: %w", err)
		}
		todosContent = string(content)
	} else {
		todosContent = `- [[2025-06-20]]
  - [ ] Task from Friday
  - [x] Completed Friday task
- [[2025-06-21]]
  - [ ] Task from Saturday
  - [x] Completed Saturday task
    Continuation for completed
  - [ ] Another open Saturday task
    - [ ] Subtask
      - [ ] Sub-subtask
- [[2025-06-22]]
  - [ ] Task from Sunday with #2025-06-22 tag
  - [x] Completed Sunday task`
	}

	custom := config.Custom
	if customVars != "" {
		parsed, err := parseCustomVarsJSON(customVars)
		if err != nil {
			return fmt.Errorf("failed to parse custom vars: %w", err)
		}
		custom = parsed
	}

	tmplSource := resolveTemplate(templateFile)
	if tmplSource.err != nil {
		return fmt.Errorf("error resolving template: %w", tmplSource.err)
	}

	journal, err := core.ParseTodosSection(todosContent)
	if err != nil {
		return fmt.Errorf("failed to parse todos section: %w", err)
	}

	output, err := core.CreateFromTemplate(core.TemplateOptions{
		Content:      tmplSource.content,
		TodosContent: todosContent,
		CurrentDate:  date,
		PreviousDate: "",
		Journal:      journal,
		CustomVars:   custom,
	})
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	fmt.Println(output)
	return nil
}

func parseCustomVarsJSON(jsonStr string) (map[string]interface{}, error) {
	if jsonStr == "" {
		return nil, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return nil, err
	}

	return m, nil
}
