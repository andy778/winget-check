# winget-check

> **Proof of Concept Notice**  
> This repository is a Proof of Concept (PoC) for proposing a **WinGet** packaging extension to the official [OpenSSF Scorecard Packaging Check](https://github.com/ossf/scorecard/blob/main/docs/checks.md#packaging).
>
> The goal is to quickly and reliably evaluate whether a project distributes official Windows packages via the official Windows Package Manager repository (`microsoft/winget-pkgs`) directly from its own automated build pipelines.

---

## 1. Rationale & Risk Assessment

Distributing software packages through official package managers improves supply chain security by reducing reliance on unverified third-party binaries or manual installer downloads. 

For Windows environments, [Windows Package Manager (winget)](https://github.com/microsoft/winget-pkgs) is the primary native package repository. However, a package listed in `winget-pkgs` may be created and updated in two distinct ways:
1. **Official CI/CD Automation:** The project's official repository runs a GitHub Action (or CI workflow) on release to automatically generate and publish WinGet manifests. This provides a direct, verifiable link between source code releases and package availability.
2. **Third-Party / Community Submissions:** A third party or automated community bot (e.g., `Komac`) submits a manifest to `winget-pkgs`. While functional, these packages lack direct, verifiable build provenance connecting their publication back to the project's own repository and maintainers.

Evaluating both the presence of a WinGet package and its connection to the project's release pipeline provides a clearer signal of packaging security and maintainability.

---

## 2. Scoring Criteria

When evaluated against a target repository (e.g. `--repo=github.com/notepad-plus-plus/notepad-plus-plus`), the check applies the following qualitative scoring tiers:

| Tier | Score Level | Description & Criteria |
| :--- | :--- | :--- |
| **High Score** | **Official CI Publishing** | A WinGet package exists in `microsoft/winget-pkgs` **and** the project automatically builds/publishes package manifests directly from its own repository workflows (e.g., via GitHub Actions). |
| **Low Score** | **Unverified / Third-Party Package** | A WinGet package referencing the project exists in `microsoft/winget-pkgs`, but there is **no verifiable connection** to the project's repository or automated workflows (e.g., community-maintained or unverified third-party bot submissions). |
| **Zero Score** | **Package Not Found** | No matching WinGet package or manifest referencing the repository URL exists in `microsoft/winget-pkgs`. |

### Edge Cases & Scoring Nuances

- **Non-GitHub CI Pipelines (Azure DevOps, GitLab CI, AppVeyor):** If a project uses external CI rather than GitHub Actions, the check falls back to verifying commit/PR author provenance in `microsoft/winget-pkgs` for maintainer signatures or authorized release tokens.
- **Automated Community Bots:** Submissions by community update bots (such as `Komac`) receive a **Low Score** unless the source repository actively runs an official workflow integration, as third-party bot updates lack cryptographic or workflow provenance from the source project.

---

## 3. Technical Implementation & Probing Signals

### Current PoC CLI Capabilities
The `winget-check` tool probes `microsoft/winget-pkgs` using the GitHub API:
- **Code Search Query:** Queries `repo:microsoft/winget-pkgs "<host>/<owner>/<repo>"` via the GitHub Search API to identify manifests pointing to the project repository.
- **Direct Tree Fallback:** If code search returns 0 results or encounters index delays, the tool derives candidate publisher and app directory names from the repository's `owner` and `repo` names (e.g., inspecting `manifests/<letter>/<Publisher>/<App>/`) via the GitHub REST Contents API.
- **Package & Version Parsing:** Extracts `Publisher.AppName` and finds the highest semantic version available across matched installer manifests.

### Proposed OpenSSF Scorecard Detection Heuristics
To distinguish between **High Score** (Official CI) and **Low Score** (Unverified/Third-Party), the proposed Scorecard probe evaluates the following signals:

1. **Workflow Analysis in Source Repo:**
   - Scans `.github/workflows/*.yml` in the project repository for recognized WinGet publishing actions (such as `vedantmgoyal2009/winget-releaser`, `microsoft/winget-pkgs-submission-action`, or custom `wingetcreate` steps).
2. **Manifest & Commit Provenance in `winget-pkgs`:**
   - Verifies whether manifest update pull requests/commits in `microsoft/winget-pkgs` originate from an authorized bot/token associated with the project's release pipeline or maintainer accounts.

### Production Considerations for Scorecard Integration
- **Search API Rate Limits:** GitHub's Code Search API has strict rate limits (30 requests/minute for authenticated users). For integration into the core OpenSSF Scorecard engine, the probe can use GitHub REST/GraphQL tree queries, pre-indexed WinGet package datasets, or local SQLite index caches rather than live search queries.

---

## 4. Building & Running

**Prerequisites:** Go 1.21+ and `export GITHUB_AUTH_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx`

```bash
# Build binary
go build -o winget-check .

# Run check (--repo accepts owner/repo, github.com/owner/repo, or full HTTPS URL)
./winget-check --repo=github.com/notepad-plus-plus/notepad-plus-plus

# Add --debug to inspect API queries
./winget-check --repo=github.com/notepad-plus-plus/notepad-plus-plus --debug
```

---

## 5. Output Examples

### Official CI Package (High Score - 10/10)

```text
repo:        github.com/JanDeDobbeleer/oh-my-posh
query time:  412ms
manifests:   12 match(es)
version:     24.24.1 (highest of 12 of 12 manifest match(es) scanned)
manifest:    manifests/j/JanDeDobbeleer/OhMyPosh/24.24.1/JanDeDobbeleer.OhMyPosh.yaml
package:     FOUND in winget as "JanDeDobbeleer.OhMyPosh"
workflow:    DETECTED (action match (uses: vedantmgoyal2009/winget-releaser) in release.yml)
provenance:  OFFICIAL (score: 10/10) - Official CI automation detected (action match (uses: vedantmgoyal2009/winget-releaser) in release.yml)
```

### Third-Party / Unverified Package (Low Score - 3/10)

```text
repo:        github.com/notepad-plus-plus/notepad-plus-plus
query time:  356ms
manifests:   0 match(es)
version:     8.9.7 (via direct tree lookup)
manifest:    manifests/n/Notepad++/Notepad++/8.9.7
package:     FOUND in winget as "Notepad++.Notepad++"
workflow:    NOT DETECTED (no WinGet release workflow pattern matched)
provenance:  THIRD_PARTY (score: 3/10) - Package exists in winget-pkgs but no official CI release workflow was detected in source repository
```

### Package Not Found (Zero Score - 0/10)

```text
repo:        github.com/gorilla/mux
query time:  255ms
manifests:   0 match(es)
package:     NOT FOUND in winget
workflow:    NOT DETECTED (no WinGet release workflow pattern matched)
provenance:  NOT_FOUND (score: 0/10) - No matching package found in microsoft/winget-pkgs
```

---

## 6. Security Scanning & Releases

### Security Scanning
[OpenSSF Scorecard](.github/workflows/scorecard.yml) and [CodeQL](.github/workflows/codeql.yml) run automatically on repository pushes and pull requests.
- View scanning alerts in the **Security** tab → **Code scanning alerts**.
- Or run Scorecard locally:
  ```bash
  scorecard --repo=github.com/andy778/winget-check
  ```

### Release Pipeline & Attestations
Pushing a tag matching `v*.*.*` (e.g. `v1.0.0`) triggers `.github/workflows/release.yml`, which cross-compiles binaries and attaches SLSA provenance attestations.

Verify release binaries with GitHub CLI:
```bash
gh attestation verify winget-check-linux-amd64 --owner andy778
```

---

## 7. Exit Codes

| Code | Meaning |
| :--- | :--- |
| `0` | Ran successfully (whether or not the package was found) |
| `1` | Runtime error (API request failure, non-200 response, JSON parse error) |
| `2` | Usage error (missing `--repo` or missing `GITHUB_AUTH_TOKEN`) |

---

## License

See [LICENSE](LICENSE).
