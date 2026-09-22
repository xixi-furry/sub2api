package service

import (
	"strconv"
	"strings"
)

// Build-time override keeps upstream builds unchanged while fork artifacts use
// their own releases for both update checks and rollback downloads.
var githubRepo = "Wei-Shaw/sub2api"

// forkRevision only extends upstream's comparison for our stable -fixN releases.
func forkRevision(version string) int {
	_, suffix, ok := strings.Cut(version, "-fix")
	if !ok || suffix == "" {
		return 0
	}
	for _, char := range suffix {
		if char < '0' || char > '9' {
			return 0
		}
	}
	revision, err := strconv.Atoi(suffix)
	if err != nil || revision < 1 {
		return 0
	}
	return revision
}
