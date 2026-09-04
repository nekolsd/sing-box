package uot

import (
	"testing"

	M "github.com/sagernet/sing/common/metadata"
	singUoT "github.com/sagernet/sing/common/uot"

	"github.com/stretchr/testify/require"
)

func TestUoTDestination(t *testing.T) {
	require.True(t, IsUoTDestination(M.Socksaddr{Fqdn: singUoT.MagicAddress}))
	require.True(t, IsUoTDestination(M.Socksaddr{Fqdn: singUoT.LegacyMagicAddress}))
	require.False(t, IsUoTDestination(M.Socksaddr{Fqdn: "example.com", Port: 443}))
}
