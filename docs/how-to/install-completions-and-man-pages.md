# Install shell completions & man pages

Completions for `bash`, `zsh`, `fish`, and `powershell` are emitted via Cobra's
built-in `completion` subcommand:

```bash
# Load completions for the current shell session (bash)
source <(planwerk-agent completion bash)

# Install persistently (zsh, Homebrew example)
planwerk-agent completion zsh > "$(brew --prefix)/share/zsh/site-functions/_planwerk-agent"

# Fish
planwerk-agent completion fish > ~/.config/fish/completions/planwerk-agent.fish
```

When installed from Homebrew, deb, or rpm packages, completions and man pages
(`man planwerk-agent`) are installed automatically. Packages are produced by
`goreleaser` — see `.goreleaser.yml`.

For local development, regenerate the artifacts into `completions/` and
`docs/man/`:

```bash
make completions
make man
```
