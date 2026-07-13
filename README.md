# winget-check

[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/andy778/winget-check/badge)](https://securityscorecards.dev/viewer/?uri=github.com/andy778/winget-check)

This is a proof of concept, built to test whether "does a winget package exist
for this repo" is worth proposing as a check/probe in
[OpenSSF Scorecard](https://github.com/ossf/scorecard) itself, rather than a
standalone tool. Its own Scorecard score and CI setup double as a testbed for
that.

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

Add `--debug` to print the full code-search query and request URL to stderr
before the request is made:

```bash
go run main.go --repo=github.com/notepad-plus-plus/notepad-plus-plus --debug
```

```
debug: query: repo:microsoft/winget-pkgs "github.com/notepad-plus-plus/notepad-plus-plus"
debug: url:   https://api.github.com/search/code?q=repo%3Amicrosoft%2Fwinget-pkgs+%22github.com%2Fnotepad-plus-plus%2Fnotepad-plus-plus%22
```

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
version:     8.9.6 (highest of 37 of 37 manifest match(es) scanned)
manifest:    manifests/n/Notepad++/Notepad++/8.9.6/Notepad++.Notepad++.installer.yaml
result:      FOUND in winget as "Notepad++.Notepad++"
```

The `version:` line is the highest version parsed from the manifests that
belong to the same winget package ID as the most relevant match, and
`manifest:` is that version's manifest. Note GitHub code search returns up to
100 results per query, so on packages with a very large number of manifests
the "highest" is only scanned from the first 100 matches - the two numbers in
the `version:` line show how many of the total matches were actually scanned.
If none of the scanned manifests have a parseable version, the line reads
`version:     unknown (...)` instead.

A package that is not in winget:

```
repo:        some-owner/some-repo
query time:  255ms
manifests:   0 match(es)
result:      NOT FOUND in winget
```

## Releases

Pushing a tag matching `v*.*.*` (e.g. `v1.0.0`) triggers a
[release workflow](.github/workflows/release.yml) that cross-compiles binaries
for Linux, macOS (amd64/arm64) and Windows, exports an SBOM
(`winget-check.spdx.json`) from GitHub's native
[dependency graph](https://docs.github.com/en/code-security/supply-chain-security/understanding-your-software-supply-chain/exporting-a-software-bill-of-materials-for-your-repository),
and publishes everything together as a GitHub Release.

## Exit codes

| Code | Meaning                                                        |
| ---- | ------------------------------------------------------------- |
| `0`  | Ran successfully (whether or not the package was found)        |
| `1`  | Runtime error (request failed, non-200 API response, bad JSON) |
| `2`  | Usage error (missing `--repo` or missing `GITHUB_AUTH_TOKEN`)  |

## License

See [LICENSE](LICENSE).
