// Command winget-check takes a --repo (like scorecard) and reports whether a
// winget package referencing that repo exists in microsoft/winget-pkgs and whether
// it is published via official CI workflows.
//
// Usage:
//
//	export GITHUB_AUTH_TOKEN=<token>   # PowerShell: $env:GITHUB_AUTH_TOKEN="..."
//	go run main.go --repo=github.com/notepad-plus-plus/notepad-plus-plus
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	goversion "github.com/hashicorp/go-version"

	"winget-check/pkg/provenance"
	"winget-check/pkg/workflow"
)

type codeSearchResult struct {
	TotalCount int `json:"total_count"`
	Items      []struct {
		Path string `json:"path"`
	} `json:"items"`
}

type contentItem struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
}

// parseRepo extracts the git host (e.g. "github.com", "gitlab.com", "codeberg.org", "sr.ht") and repo path ("owner/repo").
func parseRepo(repo string) (host string, repoPath string) {
	repo = strings.TrimSpace(repo)
	repo = strings.TrimPrefix(repo, "https://")
	repo = strings.TrimPrefix(repo, "http://")
	repo = strings.TrimSuffix(repo, ".git")
	repo = strings.Trim(repo, "/")

	parts := strings.Split(repo, "/")
	if len(parts) >= 3 && strings.Contains(parts[0], ".") {
		return parts[0], strings.Join(parts[1:], "/")
	}
	if len(parts) == 2 && strings.Contains(parts[0], ".") {
		return parts[0], parts[1]
	}
	return "github.com", repo
}

// normalizeRepo turns the various --repo forms scorecard accepts into "owner/repo".
func normalizeRepo(repo string) string {
	_, repoPath := parseRepo(repo)
	return repoPath
}

// packageIDFromPath derives "Publisher.AppName" from a manifest path like
// manifests/n/Notepad++/Notepad++/8.9/Notepad++.Notepad++.installer.yaml
var pkgPathRe = regexp.MustCompile(`^manifests/[^/]+/([^/]+)/([^/]+)/`)

func packageIDFromPath(p string) string {
	if m := pkgPathRe.FindStringSubmatch(p); m != nil {
		return m[1] + "." + m[2]
	}
	return ""
}

// versionFromPath returns the version directory of a manifest path, i.e. the
// folder immediately containing the manifest file:
//
//	manifests/n/Notepad++/Notepad++/8.9/Notepad++.Notepad++.installer.yaml -> 8.9
func versionFromPath(p string) string {
	parts := strings.Split(p, "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-2]
}

func sanitize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "plus", "+")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, "+", "")
	return s
}

func fetchContents(client *http.Client, token string, apiURL string) ([]contentItem, error) {
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "winget-check")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s", resp.Status)
	}

	var items []contentItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}

func fallbackDirectLookup(client *http.Client, token string, repo string, debug bool) (pkgID string, highestVersion string, manifestPath string, found bool) {
	parts := strings.Split(repo, "/")
	if len(parts) < 2 {
		return "", "", "", false
	}
	owner, repoName := parts[0], parts[1]
	if len(owner) == 0 {
		return "", "", "", false
	}

	letters := []string{
		strings.ToLower(string(owner[0])),
	}
	if len(repoName) > 0 && strings.ToLower(string(repoName[0])) != letters[0] {
		letters = append(letters, strings.ToLower(string(repoName[0])))
	}

	for _, letter := range letters {
		pubURL := fmt.Sprintf("https://api.github.com/repos/microsoft/winget-pkgs/contents/manifests/%s", letter)
		if debug {
			fmt.Fprintf(os.Stderr, "debug: fallback checking publishers: %s\n", pubURL)
		}
		pubItems, err := fetchContents(client, token, pubURL)
		if err != nil {
			continue
		}

		cleanOwner := sanitize(owner)
		cleanRepo := sanitize(repoName)

		var matchedPubs []string
		for _, item := range pubItems {
			cleanItem := sanitize(item.Name)
			if cleanItem == cleanOwner || cleanItem == cleanRepo || (len(cleanItem) > 2 && len(cleanOwner) > 2 && (strings.Contains(cleanItem, cleanOwner) || strings.Contains(cleanOwner, cleanItem))) {
				matchedPubs = append(matchedPubs, item.Name)
			}
		}

		for _, pubName := range matchedPubs {
			appURL := fmt.Sprintf("https://api.github.com/repos/microsoft/winget-pkgs/contents/manifests/%s/%s", letter, url.PathEscape(pubName))
			if debug {
				fmt.Fprintf(os.Stderr, "debug: fallback checking apps: %s\n", appURL)
			}
			appItems, err := fetchContents(client, token, appURL)
			if err != nil {
				continue
			}

			for _, appItem := range appItems {
				if appItem.Type != "dir" {
					continue
				}
				cleanApp := sanitize(appItem.Name)
				if cleanApp == cleanRepo || cleanApp == cleanOwner || strings.Contains(cleanApp, cleanRepo) || strings.Contains(cleanRepo, cleanApp) {
					verURL := fmt.Sprintf("https://api.github.com/repos/microsoft/winget-pkgs/contents/manifests/%s/%s/%s", letter, url.PathEscape(pubName), url.PathEscape(appItem.Name))
					if debug {
						fmt.Fprintf(os.Stderr, "debug: fallback checking versions: %s\n", verURL)
					}
					verItems, err := fetchContents(client, token, verURL)
					if err != nil {
						continue
					}

					var bestVer *goversion.Version
					var bestVerStr string
					for _, verItem := range verItems {
						if verItem.Type != "dir" {
							continue
						}
						v, err := goversion.NewVersion(verItem.Name)
						if err != nil {
							continue
						}
						if bestVer == nil || v.GreaterThan(bestVer) {
							bestVer = v
							bestVerStr = verItem.Name
						}
					}

					if bestVerStr != "" {
						pkgID = fmt.Sprintf("%s.%s", pubName, appItem.Name)
						highestVersion = bestVerStr
						manifestPath = fmt.Sprintf("manifests/%s/%s/%s/%s", letter, pubName, appItem.Name, bestVerStr)
						return pkgID, highestVersion, manifestPath, true
					}
				}
			}
		}
	}

	return "", "", "", false
}

func main() {
	repoFlag := flag.String("repo", "", "repository to check, e.g. github.com/owner/repo")
	debugFlag := flag.Bool("debug", false, "print search queries and API request details to stderr")
	flag.Parse()

	if *repoFlag == "" {
		fmt.Fprintln(os.Stderr, "error: --repo is required")
		os.Exit(2)
	}
	token := os.Getenv("GITHUB_AUTH_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "error: GITHUB_AUTH_TOKEN is not set")
		os.Exit(2)
	}

	host, repoPath := parseRepo(*repoFlag)
	parts := strings.Split(repoPath, "/")
	if len(parts) < 2 {
		fmt.Fprintf(os.Stderr, "error: invalid repo format %q, expected owner/repo\n", repoPath)
		os.Exit(2)
	}
	owner, repoName := parts[0], parts[1]

	client := &http.Client{Timeout: 15 * time.Second}

	// PR 1: Detect WinGet publishing workflows in target repository
	wfDetected, wfDetail, err := workflow.DetectWinGetWorkflow(client, token, owner, repoName, *debugFlag)
	if err != nil && *debugFlag {
		fmt.Fprintf(os.Stderr, "debug: workflow check error: %v\n", err)
	}

	// Search microsoft/winget-pkgs for package manifest
	query := fmt.Sprintf(`repo:microsoft/winget-pkgs "%s/%s"`, host, repoPath)
	apiURL := "https://api.github.com/search/code?q=" + url.QueryEscape(query) + "&per_page=100"

	if *debugFlag {
		fmt.Fprintf(os.Stderr, "debug: query: %s\n", query)
		fmt.Fprintf(os.Stderr, "debug: url:   %s\n", apiURL)
	}

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "winget-check")

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed after %v: %v\n", elapsed, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "GitHub API returned %s (after %v): %s\n", resp.Status, elapsed, string(body))
		os.Exit(1)
	}

	var result codeSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("repo:        %s/%s\n", host, repoPath)
	fmt.Printf("query time:  %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("manifests:   %d match(es)\n", result.TotalCount)

	pkgFound := false
	pkgID := ""
	bestPath := ""
	versionStr := ""

	if result.TotalCount == 0 {
		fallbackPkgID, ver, path, found := fallbackDirectLookup(client, token, repoPath, *debugFlag)
		if found {
			pkgFound = true
			pkgID = fallbackPkgID
			bestPath = path
			versionStr = fmt.Sprintf("%s (via direct tree lookup)", ver)
		}
	} else {
		pkgFound = true
		bestPath = result.Items[0].Path
		pkgID = packageIDFromPath(bestPath)
		var bestVer *goversion.Version
		if pkgID != "" {
			for _, it := range result.Items {
				if packageIDFromPath(it.Path) != pkgID {
					continue
				}
				v, err := goversion.NewVersion(versionFromPath(it.Path))
				if err != nil {
					continue
				}
				if bestVer == nil || v.GreaterThan(bestVer) {
					bestVer, bestPath = v, it.Path
				}
			}
		}
		if bestVer != nil {
			versionStr = fmt.Sprintf("%s (highest of %d of %d manifest match(es) scanned)", bestVer.Original(), len(result.Items), result.TotalCount)
		} else {
			versionStr = fmt.Sprintf("unknown (no parseable version among %d of %d manifest match(es) scanned)", len(result.Items), result.TotalCount)
		}
	}

	// PR 2: Classify Package Origin (Official vs 3rd-Party)
	classRes := provenance.ClassifyPackageOrigin(pkgFound, wfDetected, wfDetail)

	if pkgFound {
		fmt.Printf("version:     %s\n", versionStr)
		fmt.Printf("manifest:    %s\n", bestPath)
		fmt.Printf("package:     FOUND in winget as %q\n", pkgID)
	} else {
		fmt.Println("package:     NOT FOUND in winget")
	}

	if wfDetected {
		fmt.Printf("workflow:    DETECTED (%s)\n", wfDetail)
	} else {
		fmt.Printf("workflow:    NOT DETECTED (%s)\n", wfDetail)
	}

	fmt.Printf("provenance:  %s (score: %d/10) - %s\n", classRes.Origin, classRes.Score, classRes.Description)
}
