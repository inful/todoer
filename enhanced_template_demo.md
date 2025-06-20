---
title: {{.Date}}
created: {{.DateLong}}
week: {{.WeekNumber}}
{{if .PreviousDate}}previous_journal: {{.PreviousDateLong}}{{end}}
---

# Weekly Journal - Week {{.WeekNumber}} of {{.Year}}

## {{.DayName}}, {{.DateLong}}

{{if .PreviousDate}}
### Todos (carried over from {{.PreviousDayName}}, {{.PreviousDateLong}})
{{else}}
### Todos
{{end}}

{{.TODOS}}

### Today's Focus
- Key priorities for {{.DayName}}

### Notes
- Daily reflections and notes

### Tomorrow's Prep
- Items to prepare for the next day
