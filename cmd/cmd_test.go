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
	// isBuildDirty now reflects the toolchain's VCS "modified" stamp rather than
	// an ldflag default, so its value depends on how this test binary was built;
	// just exercise it. The parsing itself is covered by constants.Test_parseVCS.
	_ = isBuildDirty()
}
