---
date: {{.Date}}
project: {{.Custom.ProjectName}}
version: {{.Custom.Version}}
author: {{.Custom.Author}}
---

# {{.Custom.ProjectName}} - Daily Journal

**Date:** {{.DateLong}} ({{.DayName}})  
**Version:** {{.Custom.Version}}  
**Author:** {{.Custom.Author}}  
{{if .Custom.Debug}}**Debug Mode:** Enabled{{end}}

## Project Overview

{{if .PreviousDate}}Previous entry: {{.PreviousDateLong}} ({{.PreviousDayName}}){{end}}

### Statistics

- **Total active todos**: {{.TotalTodos}}
- **Completed todos**: {{.CompletedTodos}}
{{if .OldestTodoDate}}- **Oldest todo date**: {{.OldestTodoDate}}{{end}}
{{if .TodoDaysSpan}}- **Days spanned by todos**: {{.TodoDaysSpan}}{{end}}
{{if .TodoDates}}- **Todo dates**: {{range $i, $date := .TodoDates}}{{if $i}}, {{end}}{{$date}}{{end}}{{end}}

### Task Categories

{{range .Custom.Tags}}- {{.}}
{{end}}

## Today's Tasks (Max: {{.Custom.MaxTasks}})

{{.TODOS}}

## Daily Notes

Reflections for {{.MonthName}} {{.Day}}, {{.Year}}

## Next Steps

Planning for tomorrow...
