# Security

## Reporting a vulnerability

If you discover a security issue in todoer, please report it privately:

- **GitHub:** open a private security advisory at <https://github.com/inful/todoer/security/advisories/new> (preferred for any vulnerability that could affect users).
- **Email:** contact the maintainer via the address in their GitHub profile if the advisory form is not available.

Please **do not** file a public GitHub issue for security bugs. Public reports give attackers a head start before a fix is shipped.

When reporting, please include:

1. A clear description of the issue and its impact.
2. Reproduction steps or a minimal test case (redact any private data).
3. The version affected (commit SHA or release tag) and your environment.

You can expect an acknowledgement within a few days. The maintainer will coordinate disclosure timing with you.

## Scope

todoer is a CLI that reads and writes markdown files on the local filesystem. The realistic attack surface is small:

- **Path handling** — user-supplied paths flow through `validateFilePath` (`cmd/todoer/validation.go`), which rejects `..` traversal at component boundaries before normalisation. Symlinks are not dereferenced through trust boundaries.
- **Template execution** — `text/template` is used with the data shape exposed in `pkg/core/types.go::TemplateData`. Users control the template content and custom variables; template authors control the variables they reference. There is no `text/template/htmlescape` need because the output is markdown.
- **Frontmatter parsing** — YAML-ish `key: value` lines only. The configurable date key is escaped with `regexp.QuoteMeta` before being embedded into the date regex.
- **File writes** — every disk write goes through `cmd/todoer/fs.go::safeWriteFile` (temp file + rename + chmod `0o644`). A failed source write leaves the target untouched (the daily flow writes source first, then target).
- **Dependencies** — the dependency tree is intentionally small. The fingerprint spike depends on `github.com/inful/mdfp` (SHA-256 of the markdown body). No network calls are made at runtime.

## Out-of-scope

- Denial-of-service via a malicious journal file (todoer is a single-user tool; large or pathological inputs are not considered a security boundary).
- Bugs in upstream Go modules — please report those upstream.
- Issues that depend on the user running an untrusted template with elevated privileges (todoer runs with the user's own permissions; this is by design).

## Disclosure policy

- Critical fixes are released as soon as a patch is ready.
- Non-critical fixes may be batched into the next regular release.
- A CVE will be requested if the maintainer and reporter agree the issue warrants one.