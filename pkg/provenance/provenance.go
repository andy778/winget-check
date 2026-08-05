package provenance

import "fmt"

// Origin represents the origin classification of a WinGet package.
type Origin string

const (
	// OriginOfficial indicates the package is published via official CI workflows (High Score).
	OriginOfficial Origin = "OFFICIAL"
	// OriginThirdParty indicates a package exists but lacks official CI workflow provenance (Low Score).
	OriginThirdParty Origin = "THIRD_PARTY"
	// OriginNotFound indicates no package was found in microsoft/winget-pkgs (Zero Score).
	OriginNotFound Origin = "NOT_FOUND"
)

// ClassificationResult contains the qualitative tier, score, and explanation.
type ClassificationResult struct {
	Origin      Origin `json:"origin"`
	Score       int    `json:"score"`
	Description string `json:"description"`
}

// ClassifyPackageOrigin correlates package presence in winget-pkgs with workflow detection in the source repository.
func ClassifyPackageOrigin(packageFound bool, workflowDetected bool, workflowDetail string) ClassificationResult {
	if !packageFound {
		return ClassificationResult{
			Origin:      OriginNotFound,
			Score:       0,
			Description: "No matching package found in microsoft/winget-pkgs",
		}
	}

	if workflowDetected {
		return ClassificationResult{
			Origin:      OriginOfficial,
			Score:       10,
			Description: fmt.Sprintf("Official CI automation detected (%s)", workflowDetail),
		}
	}

	return ClassificationResult{
		Origin:      OriginThirdParty,
		Score:       3,
		Description: "Package exists in winget-pkgs but no official CI release workflow was detected in source repository",
	}
}
