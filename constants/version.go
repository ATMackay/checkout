package constants

import "runtime/debug"

// Version is the semantic version. It defaults to a development placeholder and
// is overridden at build time via ldflag:
//
//	-ldflags "-X 'github.com/ATMackay/checkout/constants.Version=$(git describe --tags)'"
var Version = "0.0.0-dev"

// BuildDate is the wall-clock time the binary was compiled, injected via ldflag:
//
//	-ldflags "-X 'github.com/ATMackay/checkout/constants.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
//
// Unlike CommitDate (from VCS) it is not reproducible — it answers "when was
// this binary built", which the commit date cannot.
var BuildDate = vcsUnknown

// vcsUnknown is the fallback when a binary was built without VCS stamping
// (e.g. `go run`, or `go build -buildvcs=false`).
const vcsUnknown = "unknown"

// Build metadata is sourced from the Go toolchain's embedded VCS stamps, which
// `go build -buildvcs=true` records automatically — no ldflags, no shell git
// plumbing. GitCommit is the commit SHA, CommitDate the commit timestamp
// (RFC3339, and reproducible, unlike a wall-clock build date), and Dirty is
// whether the working tree had uncommitted changes at build time.
var (
	GitCommit  = vcsUnknown
	CommitDate = vcsUnknown
	Dirty      bool
)

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		GitCommit, CommitDate, Dirty = parseVCS(info.Settings)
	}
}

// parseVCS extracts the commit SHA, commit time and dirty flag from the build
// settings the toolchain embeds. Split out from init so it is unit-testable.
func parseVCS(settings []debug.BuildSetting) (commit, date string, dirty bool) {
	commit, date = vcsUnknown, vcsUnknown
	for _, s := range settings {
		switch s.Key {
		case "vcs.revision":
			commit = s.Value
		case "vcs.time":
			date = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	return commit, date, dirty
}
