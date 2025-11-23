# Todoer

Todoer is a CLI tool that manages todos in daily markdown journals. It carries over incomplete tasks to the next day while preserving completed tasks with date annotations.

This README gives you a quick, task‑oriented introduction. For full details, see:

- `docs/HOWTO.md` – task‑based how‑to guides
- `docs/REFERENCE.md` – CLI, template, and API reference
- `docs/BACKGROUND.md` – design and behavior explanation

## Quick start

```bash
# Build the binary
go build -o todoer ./cmd/todoer

# Create today’s journal (uses defaults or config)
./todoer new

# Or process an existing journal into a new file
./todoer process source.md target.md
```

By default, journals are stored under `ROOT/YYYY/MM/YYYY-MM-DD.md` and only the configured todos section (default `## Todos`) is processed.

## Basic workflow

1. **Pick a journals directory** (for example `~/Documents/journals`).
2. Optional: configure it in `~/.config/todoer/config.toml`:

	```toml
	root_dir = "~/Documents/journals"
	# Optional: custom template
	template_file = "~/.config/todoer/template.md"
	```

3. **Create today’s journal**:

```bash
./todoer new --root-dir "~/Documents/journals"
```

This creates `YYYY/MM/YYYY-MM-DD.md`, finds the most recent previous journal, moves incomplete todos to today, and tags completed todos in the previous file.

4. **Edit your journal** during the day using normal markdown checkboxes:

```markdown
## Todos

- [[2025-06-20]]
	- [ ] Uncompleted task
	- [x] Completed task
```

5. **Repeat** `./todoer new` the next day to carry over incomplete tasks.

## Configuration and templates

- Configuration can come from CLI flags, environment variables, or a config file. See `docs/HOWTO.md` for precedence and examples.
- Journals are created from a template. You can supply your own template file and use Todoer’s template variables and functions. See `docs/REFERENCE.md` for the full list.

## Next steps

- Read `docs/HOWTO.md` to configure storage, templates, and custom variables.
- Use `docs/REFERENCE.md` when you need precise CLI, template, or library details.
- Consult `docs/BACKGROUND.md` if you want to understand how Todoer processes journals and why it behaves the way it does.

