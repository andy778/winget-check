package provenance

import "testing"

func TestClassifyPackageOrigin(t *testing.T) {
	tests := []struct {
		name             string
		packageFound     bool
		workflowDetected bool
		workflowDetail   string
		wantOrigin       Origin
		wantScore        int
	}{
		{
			name:             "Official CI publishing",
			packageFound:     true,
			workflowDetected: true,
			workflowDetail:   "action match (uses: vedantmgoyal2009/winget-releaser) in release.yml",
			wantOrigin:       OriginOfficial,
			wantScore:        10,
		},
		{
			name:             "Third-party or community package",
			packageFound:     true,
			workflowDetected: false,
			workflowDetail:   "no WinGet release workflow pattern matched",
			wantOrigin:       OriginThirdParty,
			wantScore:        3,
		},
		{
			name:             "Package not found",
			packageFound:     false,
			workflowDetected: false,
			workflowDetail:   "no WinGet release workflow pattern matched",
			wantOrigin:       OriginNotFound,
			wantScore:        0,
		},
		{
			name:             "Workflow present but no package published yet",
			packageFound:     false,
			workflowDetected: true,
			workflowDetail:   "action match (uses: microsoft/winget-pkgs-submission-action) in release.yml",
			wantOrigin:       OriginNotFound,
			wantScore:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyPackageOrigin(tt.packageFound, tt.workflowDetected, tt.workflowDetail)
			if got.Origin != tt.wantOrigin {
				t.Errorf("ClassifyPackageOrigin().Origin = %v; want %v", got.Origin, tt.wantOrigin)
			}
			if got.Score != tt.wantScore {
				t.Errorf("ClassifyPackageOrigin().Score = %v; want %v", got.Score, tt.wantScore)
			}
		})
	}
}
