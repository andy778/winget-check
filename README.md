# winget-check

[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/andy778/winget-check/badge)](https://securityscorecards.dev/viewer/?uri=github.com/andy778/winget-check)

A small command-line tool that takes a repository (the same `--repo` form
[OpenSSF Scorecard](https://github.com/ossf/scorecard) accepts) and reports
whether a [winget](https://github.com/microsoft/winget-pkgs) package referencing
that repo exists in `microsoft/winget-pkgs`.

It works by running a GitHub [code search](https://docs.github.com/en/rest/search/search#search-code)
against the `microsoft/winget-pkgs` repository for manifests that mention the
given GitHub repo URL, then derives the winget package ID (`Publisher.AppName`)
from the manifest path.

## Requirements

- [Go](https://go.dev/dl/) 1.21 or newer
- A GitHub personal access token (a classic or fine-grained token with public
  read access is enough — code search requires authentication)

## Setup

Set your token in the `GITHUB_AUTH_TOKEN` environment variable:

```bash
# bash / zsh
export GITHUB_AUTH_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
```

```powershell
# PowerShell
$env:GITHUB_AUTH_TOKEN = "ghp_xxxxxxxxxxxxxxxxxxxx"
```

## Usage

Run directly with `go run`:

```bash
go run main.go --repo=github.com/notepad-plus-plus/notepad-plus-plus
```

The `--repo` flag accepts the various forms Scorecard understands, for example:

- `github.com/owner/repo`
- `https://github.com/owner/repo`
- `owner/repo`
- a trailing `.git` or `/` is tolerated

### Build a binary

```bash
go build -o winget-check .
./winget-check --repo=github.com/notepad-plus-plus/notepad-plus-plus
```

On Windows the binary is `winget-check.exe`:

```powershell
go build -o winget-check.exe .
.\winget-check.exe --repo=github.com/notepad-plus-plus/notepad-plus-plus
```

## Output

A package that exists in winget:

```
repo:        notepad-plus-plus/notepad-plus-plus
query time:  412ms
manifests:   37 match(es)
result:      FOUND in winget as "Notepad++.Notepad++"
```

A package that is not in winget:

```
repo:        some-owner/some-repo
query time:  255ms
manifests:   0 match(es)
result:      NOT FOUND in winget
```

## Exit codes

| Code | Meaning                                                        |
| ---- | ------------------------------------------------------------- |
| `0`  | Ran successfully (whether or not the package was found)        |
| `1`  | Runtime error (request failed, non-200 API response, bad JSON) |
| `2`  | Usage error (missing `--repo` or missing `GITHUB_AUTH_TOKEN`)  |

## License

See [LICENSE](LICENSE).
