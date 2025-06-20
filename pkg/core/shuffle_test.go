package core

import (
	"strings"
	"testing"
	"text/template"
)

func TestShuffleFunctionality(t *testing.T) {
	funcMap := CreateTemplateFunctions()

	t.Run("Shuffle Determinism Test", func(t *testing.T) {
		// Test that shuffle actually shuffles (not deterministic)
		templateContent := `{{shuffle "line1\nline2\nline3\nline4\nline5"}}`

		tmpl, err := template.New("test").Funcs(funcMap).Parse(templateContent)
		if err != nil {
			t.Fatalf("Failed to parse template: %v", err)
		}

		originalOrder := "line1\nline2\nline3\nline4\nline5"
		sameOrderCount := 0
		totalRuns := 20

		for i := 0; i < totalRuns; i++ {
			var result strings.Builder
			err = tmpl.Execute(&result, nil)
			if err != nil {
				t.Fatalf("Failed to execute template: %v", err)
			}

			if strings.TrimSpace(result.String()) == originalOrder {
				sameOrderCount++
			}
		}

		// With 5 lines, probability of getting original order is 1/120 = 0.83%
		// In 20 runs, getting it more than 3 times would be very unlikely if truly random
		if sameOrderCount > 3 {
			t.Errorf("Shuffle appears to not be working properly. Got original order %d/%d times", sameOrderCount, totalRuns)
		}
	})

	t.Run("Practical Template Example", func(t *testing.T) {
		templateContent := `# Daily Ideas

Here are some randomly shuffled ideas for today:

{{shuffle "Practice coding\nRead a book\nGo for a walk\nCall a friend\nCook something new\nListen to music\nWrite in journal\nExercise\nLearn something new\nOrganize workspace"}}

## End of Ideas`

		tmpl, err := template.New("test").Funcs(funcMap).Parse(templateContent)
		if err != nil {
			t.Fatalf("Failed to parse template: %v", err)
		}

		var result strings.Builder
		err = tmpl.Execute(&result, nil)
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}

		output := result.String()

		// Check that all ideas are present
		expectedIdeas := []string{
			"Practice coding",
			"Read a book",
			"Go for a walk",
			"Call a friend",
			"Cook something new",
			"Listen to music",
			"Write in journal",
			"Exercise",
			"Learn something new",
			"Organize workspace",
		}

		for _, idea := range expectedIdeas {
			if !strings.Contains(output, idea) {
				t.Errorf("Expected idea %q not found in output", idea)
			}
		}

		// Check that structure is maintained
		if !strings.Contains(output, "# Daily Ideas") {
			t.Error("Title not found")
		}
		if !strings.Contains(output, "## End of Ideas") {
			t.Error("End section not found")
		}

		t.Logf("Generated template output:\n%s", output)
	})

	t.Run("ShuffleLines with Variable Assignment", func(t *testing.T) {
		templateContent := `{{$tasks := split "\n" "Task A\nTask B\nTask C\nTask D"}}
{{$shuffledTasks := shuffleLines $tasks}}
Today's random task order:
{{range $shuffledTasks}}
- {{.}}
{{end}}`

		tmpl, err := template.New("test").Funcs(funcMap).Parse(templateContent)
		if err != nil {
			t.Fatalf("Failed to parse template: %v", err)
		}

		var result strings.Builder
		err = tmpl.Execute(&result, nil)
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}

		output := result.String()

		// Check that all tasks are present
		expectedTasks := []string{"Task A", "Task B", "Task C", "Task D"}
		for _, task := range expectedTasks {
			if !strings.Contains(output, task) {
				t.Errorf("Expected task %q not found in output", task)
			}
		}

		// Check structure
		if !strings.Contains(output, "Today's random task order:") {
			t.Error("Header not found")
		}

		// Count bullet points - should have 4
		bulletCount := strings.Count(output, "- Task")
		if bulletCount != 4 {
			t.Errorf("Expected 4 bullet points, got %d", bulletCount)
		}

		t.Logf("Generated shuffled task list:\n%s", output)
	})

	t.Run("Empty and Edge Cases", func(t *testing.T) {
		tests := []struct {
			name     string
			template string
			expected string
		}{
			{
				name:     "empty string",
				template: `{{shuffle ""}}`,
				expected: "",
			},
			{
				name:     "single line",
				template: `{{shuffle "only one line"}}`,
				expected: "only one line",
			},
			{
				name:     "only whitespace",
				template: `{{shuffle "   \n  \n  "}}`,
				expected: "",
			},
			{
				name:     "shuffleLines empty array",
				template: `{{$empty := split "\n" ""}}{{join "," (shuffleLines $empty)}}`,
				expected: "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tmpl, err := template.New("test").Funcs(funcMap).Parse(tt.template)
				if err != nil {
					t.Fatalf("Failed to parse template: %v", err)
				}

				var result strings.Builder
				err = tmpl.Execute(&result, nil)
				if err != nil {
					t.Fatalf("Failed to execute template: %v", err)
				}

				if strings.TrimSpace(result.String()) != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result.String())
				}
			})
		}
	})
}
