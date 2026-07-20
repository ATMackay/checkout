package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_CheckoutCMD(t *testing.T) {
	c := NewCheckoutCmd()
	names := make([]string, 0, len(c.Commands()))
	for _, sub := range c.Commands() {
		names = append(names, sub.Name())
	}
	// Named rather than counted: a count says nothing about which command went
	// missing, and "health" in particular is depended on by the container
	// HEALTHCHECK, which has no shell to fall back to.
	require.ElementsMatch(t, []string{"run", "version", "health"}, names)
}

func Test_BuildDirty(t *testing.T) {
	// Default value is false, overridden by ldflags.
	require.False(t, isBuildDirty())
}
