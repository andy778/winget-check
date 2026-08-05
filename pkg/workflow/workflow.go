package workflow

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// WorkflowItem represents a file entry returned by GitHub contents API.
type WorkflowItem struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	DownloadURL string `json:"download_url"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
}

// Patterns used to detect WinGet publishing actions/commands.
var (
	actionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)uses:\s*["']?vedantmgoyal2009/winget-releaser`),
		regexp.MustCompile(`(?i)uses:\s*["']?microsoft/winget-pkgs-submission-action`),
		regexp.MustCompile(`(?i)uses:\s*["']?winget-releaser/winget-releaser`),
		regexp.MustCompile(`(?i)uses:\s*["']?ks2204/winget-releaser`),
	}

	commandPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)wingetcreate\s+submit`),
		regexp.MustCompile(`(?i)wingetcreate\s+update`),
		regexp.MustCompile(`(?i)microsoft/winget-pkgs`),
	}
)

// InspectWorkflowContent checks a single workflow file's YAML content for WinGet publishing signals.
func InspectWorkflowContent(content string) (bool, string) {
	for _, pattern := range actionPatterns {
		if loc := pattern.FindString(content); loc != "" {
			return true, fmt.Sprintf("action match (%s)", strings.TrimSpace(loc))
		}
	}

	for _, pattern := range commandPatterns {
		if loc := pattern.FindString(content); loc != "" {
			return true, fmt.Sprintf("command match (%s)", strings.TrimSpace(loc))
		}
	}

	return false, ""
}

// DetectWinGetWorkflow checks the target repository's .github/workflows for WinGet release workflows.
func DetectWinGetWorkflow(client *http.Client, token, owner, repo string, debug bool) (bool, string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/.github/workflows", url.PathEscape(owner), url.PathEscape(repo))

	if debug {
		fmt.Fprintf(os.Stderr, "debug: checking workflows at %s\n", apiURL)
	}

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return false, "", err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "winget-check")

	resp, err := client.Do(req)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, "no .github/workflows directory found", nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, "", fmt.Errorf("GitHub API returned %s: %s", resp.Status, string(body))
	}

	var items []WorkflowItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return false, "", fmt.Errorf("failed to parse workflows response: %w", err)
	}

	for _, item := range items {
		if item.Type != "file" || (!strings.HasSuffix(item.Name, ".yml") && !strings.HasSuffix(item.Name, ".yaml")) {
			continue
		}

		fileURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", url.PathEscape(owner), url.PathEscape(repo), item.Path)
		fileReq, err := http.NewRequest(http.MethodGet, fileURL, nil)
		if err != nil {
			continue
		}
		if token != "" {
			fileReq.Header.Set("Authorization", "Bearer "+token)
		}
		fileReq.Header.Set("Accept", "application/vnd.github+json")
		fileReq.Header.Set("User-Agent", "winget-check")

		fileResp, err := client.Do(fileReq)
		if err != nil {
			continue
		}
		defer fileResp.Body.Close()

		if fileResp.StatusCode != http.StatusOK {
			continue
		}

		var fileItem WorkflowItem
		if err := json.NewDecoder(fileResp.Body).Decode(&fileItem); err != nil {
			continue
		}

		var rawContent string
		if fileItem.Encoding == "base64" {
			cleaned := strings.ReplaceAll(fileItem.Content, "\n", "")
			cleaned = strings.ReplaceAll(cleaned, "\r", "")
			decoded, err := base64.StdEncoding.DecodeString(cleaned)
			if err == nil {
				rawContent = string(decoded)
			}
		} else {
			rawContent = fileItem.Content
		}

		if rawContent != "" {
			if detected, matchDetail := InspectWorkflowContent(rawContent); detected {
				return true, fmt.Sprintf("%s in %s", matchDetail, item.Name), nil
			}
		}
	}

	return false, "no WinGet release workflow pattern matched", nil
}
