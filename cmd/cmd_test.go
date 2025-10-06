package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_CheckoutCMD(t *testing.T) {
	c := NewCheckoutCmd()
	require.Len(t, c.Commands(), 2)
}

func Test_BuildDirty(t *testing.T) {
	// Default value is false, overridden by ldflags.
	require.False(t, isBuildDirty())
}
