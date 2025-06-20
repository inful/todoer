---
title: {{.Date}}
---

# Daily Journal - {{formatDate .Date "Monday, January 02, 2006"}}

**Today:** {{weekday .Date}} {{if isWeekend .Date}}🏖️{{else}}💼{{end}}

## Randomized Daily Ideas 🎲

{{shuffle "Take a 10-minute walk outside\nCall someone you haven't talked to in a while\nTry a new recipe or cooking technique\nRead for 30 minutes\nOrganize one small area of your space\nWrite down three things you're grateful for\nListen to a new podcast or music genre\nDo 5 minutes of stretching or exercise\nLearn one new fact about something interesting\nCreate something with your hands"}}

## Randomized Priority Tasks 📋

{{$tasks := split "\n" "Review and respond to important emails\nWork on your main project for 2 hours\nPlan tomorrow's schedule\nTake care of one administrative task\nBrainstorm solutions for a current challenge"}}
{{range $index, $task := shuffleLines $tasks}}
{{add $index 1}}. {{$task}}
{{end}}

## Regular Todos
{{.TODOS}}

## Daily Reflection

### What went well yesterday?
{{if .PreviousDate}}- _From {{formatDate .PreviousDate "Monday, January 02"}}_{{else}}- _No previous day to reflect on_{{end}}

### Random focus for today
{{$focuses := split "\n" "Be present in conversations\nPractice patience\nSeek to understand before being understood\nShow kindness to yourself\nLook for opportunities to help others\nStay curious about new perspectives"}}
**Focus:** {{index (shuffleLines $focuses) 0}}

### Goals
- [ ] Complete priority task #1
- [ ] Try one randomized idea
- [ ] Practice today's focus

## End of Day Check-in
_How did the randomized elements work out today?_

---
*Generated on {{formatDate .Date "Monday, Jan 02, 2006"}} • Shuffle functions make each day unique!*
