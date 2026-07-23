//go:build !integration

package constants

import (
	"runtime/debug"
	"testing"
)

func Test_parseVCS(t *testing.T) {
	tests := []struct {
		name       string
		settings   []debug.BuildSetting
		wantCommit string
		wantDate   string
		wantDirty  bool
	}{
		{
			name:       "no vcs settings",
			settings:   []debug.BuildSetting{{Key: "-buildmode", Value: "exe"}},
			wantCommit: vcsUnknown,
			wantDate:   vcsUnknown,
			wantDirty:  false,
		},
		{
			name: "clean build",
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abc123"},
				{Key: "vcs.time", Value: "2026-07-20T18:44:14Z"},
				{Key: "vcs.modified", Value: "false"},
			},
			wantCommit: "abc123",
			wantDate:   "2026-07-20T18:44:14Z",
			wantDirty:  false,
		},
		{
			name: "dirty build",
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "def456"},
				{Key: "vcs.time", Value: "2026-07-21T09:00:00Z"},
				{Key: "vcs.modified", Value: "true"},
			},
			wantCommit: "def456",
			wantDate:   "2026-07-21T09:00:00Z",
			wantDirty:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commit, date, dirty := parseVCS(tt.settings)
			if commit != tt.wantCommit {
				t.Errorf("commit = %q, want %q", commit, tt.wantCommit)
			}
			if date != tt.wantDate {
				t.Errorf("date = %q, want %q", date, tt.wantDate)
			}
			if dirty != tt.wantDirty {
				t.Errorf("dirty = %v, want %v", dirty, tt.wantDirty)
			}
		})
	}
}
