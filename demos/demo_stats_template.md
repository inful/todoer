---
date: {{.Date}}
---

# Daily Journal - {{.DateLong}} ({{.DayName}})

## Summary

Today is {{.DateLong}}, which is a {{.DayName}}.

{{if .PreviousDate}}Previous entry: {{.PreviousDateLong}} ({{.PreviousDayName}}){{end}}

## Todo Statistics

- **Total active todos**: {{.TotalTodos}}
- **Completed todos**: {{.CompletedTodos}}
- **Uncompleted todos**: {{.UncompletedTodos}}
- **Uncompleted top-level todos**: {{.UncompletedTopLevelTodos}}
{{if .OldestTodoDate}}- **Oldest todo date**: {{.OldestTodoDate}}{{end}}
{{if .TodoDaysSpan}}- **Days spanned by todos**: {{.TodoDaysSpan}}{{end}}
{{if .TodoDates}}- **Todo dates**: {{range $i, $date := .TodoDates}}{{if $i}}, {{end}}{{$date}}{{end}}{{end}}

## Today's Tasks

{{ .TODOS }}

## Notes

Today's reflections...

## Tomorrow's Planning

For tomorrow, I plan to...
