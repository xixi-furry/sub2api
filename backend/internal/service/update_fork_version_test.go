package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForkVersionComparison(t *testing.T) {
	for _, tc := range []struct {
		current, latest string
		want            int
	}{
		{"0.2.7-fix1", "0.2.7-fix2", -1},
		{"v0.2.7-fix9", "v0.2.7-fix10", -1},
		{"0.2.7-fix2", "0.2.7-fix1", 1},
		{"0.2.7-fix1", "0.2.7-fix1", 0},
		{"0.2.7-fix10", "0.2.8-fix1", -1},
		{"0.2.8-fix1", "0.2.7-fix99", 1},
		{"0.2.7", "0.2.7-fix1", -1},
		{"0.2.7-fix1", "0.2.7", 1},
		{"0.2.7-rc1", "0.2.7", 0},
	} {
		t.Run(tc.current+"_to_"+tc.latest, func(t *testing.T) {
			require.Equal(t, tc.want, compareVersions(tc.current, tc.latest))
		})
	}
}
