package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_CheckoutCMD(t *testing.T) {
	c := NewCheckoutCmd()
	require.Len(t, c.Commands(), 2)
}
