# Demo: Weekday Template Functions

This demo shows how to use the `isMonday`, `isTuesday`, `isWednesday`, `isThursday`, `isFriday`, `isSaturday`, and `isSunday` template functions in your todoer templates.

```
{{- $date := .Date }}
Today is: {{weekday $date}}

{{- if isMonday $date}}It's Monday!{{end}}
{{- if isTuesday $date}}It's Tuesday!{{end}}
{{- if isWednesday $date}}It's Wednesday!{{end}}
{{- if isThursday $date}}It's Thursday!{{end}}
{{- if isFriday $date}}It's Friday!{{end}}
{{- if isSaturday $date}}It's Saturday!{{end}}
{{- if isSunday $date}}It's Sunday!{{end}}

{{if isWeekend $date}}It's the weekend!{{else}}It's a weekday.{{end}}
```

{{.TODOS}}

**How it works:**
- `.Date` is the current date passed to the template.
- Each `isXxx` function returns true if the date is that day.
- `weekday` prints the name of the day.
- `isWeekend` checks for Saturday or Sunday.

You can use these conditionals to customize your journal output for specific days.
