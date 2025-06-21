# Shell Completion for todoer

The todoer CLI does not support runtime shell completion script generation. To enable tab completion for bash, zsh, or fish, use the following manual integration steps:

## Bash
Add this to your `~/.bashrc` or `~/.bash_profile`:

```bash
_todoer_completions()
{
    COMPREPLY=()
    local cur prev opts
    cur="${COMP_WORDS[COMP_CWORD]}"
    opts="new process preview --help --debug"
    if [[ ${cur} == -* ]] ; then
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        return 0
    fi
}
complete -F _todoer_completions todoer
```

## Zsh
Add this to your `~/.zshrc`:

```zsh
_todoer_completions() {
    local -a opts
    opts=(new process preview --help --debug)
    _describe 'command' opts
}
compdef _todoer_completions todoer
```

## Fish
Add this to your `~/.config/fish/completions/todoer.fish`:

```fish
complete -c todoer -f -a 'new process preview --help --debug'
```

## Notes
- These are basic completions for top-level commands and flags. For more advanced completions, you may extend these scripts.
- After editing your shell config, restart your shell or source the file to enable completion.
