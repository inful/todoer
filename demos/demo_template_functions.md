---
title: {{.Date}}
created: {{formatDate .Date "2006-01-02T15:04:05Z07:00"}}
---

# Daily Journal - {{formatDate .Date "Monday, January 02, 2006"}}

{{$dayOfWeek := weekday .Date}}
{{$isWeekend := isWeekend .Date}}
Welcome to {{$dayOfWeek}}{{if $isWeekend}} weekend{{end}}! 

## Quick Info
- **Date**: {{.Date}} ({{upper $dayOfWeek}})
- **Week**: {{if $isWeekend}}🏖️ Weekend{{else}}💼 Weekday{{end}}
- **Week #**: {{.WeekNumber}} of {{.Year}}

{{if .PreviousDate}}
## Previous Journal
- **Date**: {{.PreviousDate}} ({{formatDate .PreviousDate "Mon, Jan 02"}})
- **Days ago**: {{daysDiff .PreviousDate .Date}}
- **Was {{weekday .PreviousDate}}**: {{if isWeekend .PreviousDate}}Weekend{{else}}Weekday{{end}}
{{end}}

## This Week Overview
{{$monday := subDays .Date (daysDiff (subDays .Date 6) .Date)}}
{{range seq 0 6}}
{{$day := addDays $monday .}}
- **{{formatDate $day "Mon 01/02"}}**: {{if eq $day $.Date}}**TODAY** 📍{{else if isWeekend $day}}Weekend 🏖️{{else}}Workday 💼{{end}}
{{end}}

## Important Dates
{{$tomorrow := addDays .Date 1}}
{{$nextWeek := addWeeks .Date 1}}
{{$nextMonth := addMonths .Date 1}}
- **Tomorrow**: {{formatDate $tomorrow "Monday, January 02"}}
- **Next week**: {{formatDate $nextWeek "Jan 02"}} (in {{daysDiff .Date $nextWeek}} days)
- **Next month**: {{formatDate $nextMonth "January 02"}} (in {{daysDiff .Date $nextMonth}} days)

{{if .TotalTodos}}
## Todo Statistics 📋

- **Total**: {{.TotalTodos}} todos
{{if .CompletedTodos}}- **Completed**: {{.CompletedTodos}} todos{{end}}
{{if .TodoDaysSpan}}- **Span**: {{.TodoDaysSpan}} days{{end}}
{{if .OldestTodoDate}}- **Oldest**: {{.OldestTodoDate}} ({{daysDiff .OldestTodoDate .Date}} days old){{end}}
{{if .TodoDates}}- **From dates**: {{join ", " .TodoDates}}{{end}}
{{else}}
## No Todos 🎉

Starting fresh today!
{{end}}

## Todos
{{.TODOS}}

## Reflection

### What went well yesterday?
{{if .PreviousDate}}- _Reflect on {{formatDate .PreviousDate "Monday, January 02"}}_{{else}}- _No previous day to reflect on_{{end}}

### Goals for today ({{title $dayOfWeek}})
- [ ] 
- [ ] 
- [ ] 

### Focus areas
{{if $isWeekend}}
- [ ] Rest and recharge
- [ ] Enjoy leisure activities
- [ ] Plan for the upcoming week
{{else}}
- [ ] Priority tasks
- [ ] Important meetings
- [ ] Progress on key projects
{{end}}

## Notes
_Space for thoughts, ideas, and observations..._

---

{{if $isWeekend}}
*Enjoy your {{lower $dayOfWeek}}! 🌟*
{{else}}
*Make it a productive {{lower $dayOfWeek}}! 💪*
{{end}}

{{/* Template functions showcase */}}
{{/*
Available template functions:
- Date arithmetic: addDays, subDays, addWeeks, addMonths, daysDiff
- Date formatting: formatDate, weekday, isWeekend  
- String manipulation: upper, lower, title, trim, replace, contains, hasPrefix, hasSuffix, split, join, repeat, len
- Utilities: default, empty, notEmpty, seq, dict
*/}}
