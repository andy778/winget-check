// Command winget-check takes a --repo (like scorecard) and reports whether a
// winget package referencing that repo exists in microsoft/winget-pkgs.
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
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	goversion "github.com/hashicorp/go-version"
)

type codeSearchResult struct {
	TotalCount int `json:"total_count"`
	Items      []struct {
		Path string `json:"path"`
	} `json:"items"`
}

// normalizeRepo turns the various --repo forms scorecard accepts into "owner/repo".
func normalizeRepo(repo string) string {
	repo = strings.TrimSpace(repo)
	repo = strings.TrimPrefix(repo, "https://")
	repo = strings.TrimPrefix(repo, "http://")
	repo = strings.TrimPrefix(repo, "github.com/")
	repo = strings.TrimSuffix(repo, ".git")
	return strings.Trim(repo, "/")
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

func main() {
	repoFlag := flag.String("repo", "", "repository to check, e.g. github.com/owner/repo")
	debugFlag := flag.Bool("debug", false, "print the search query and request URL to stderr")
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

	repo := normalizeRepo(*repoFlag)
	query := fmt.Sprintf(`repo:microsoft/winget-pkgs "github.com/%s"`, repo)
	apiURL := "https://api.github.com/search/code?q=" + url.QueryEscape(query) + "&per_page=100"

	if *debugFlag {
		fmt.Fprintf(os.Stderr, "debug: query: %s\n", query)
		fmt.Fprintf(os.Stderr, "debug: url:   %s\n", apiURL)
	}

	req, _ := http.NewRequest(http.MethodGet, apiURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "winget-check")

	client := &http.Client{Timeout: 15 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed after %v: %v\n", elapsed, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "GitHub API returned %s (after %v)\n", resp.Status, elapsed)
		os.Exit(1)
	}

	var result codeSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("repo:        %s\n", repo)
	fmt.Printf("query time:  %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("manifests:   %d match(es)\n", result.TotalCount)

	if result.TotalCount == 0 {
		fmt.Println("result:      NOT FOUND in winget")
		return
	}

	// Pick the manifest with the highest version among the matches. Code search
	// returns matches in relevance order, not version order, so scan them all.
	bestPath := ""
	var bestVer *goversion.Version
	for _, it := range result.Items {
		v, err := goversion.NewVersion(versionFromPath(it.Path))
		if err != nil {
			continue // unparseable version string, skip
		}
		if bestVer == nil || v.GreaterThan(bestVer) {
			bestVer, bestPath = v, it.Path
		}
	}
	if bestPath == "" && len(result.Items) > 0 {
		bestPath = result.Items[0].Path // fall back if no version could be parsed
	}

	pkgID := packageIDFromPath(bestPath)
	if bestVer != nil {
		fmt.Printf("version:     %s (highest of %d manifest match(es))\n", bestVer.Original(), len(result.Items))
	}
	fmt.Printf("manifest:    %s\n", bestPath)
	fmt.Printf("result:      FOUND in winget as %q\n", pkgID)
}
