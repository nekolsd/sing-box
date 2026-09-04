package route

import (
	"context"
	"net"
	"testing"

	"github.com/sagernet/sing-box/log"

	"github.com/stretchr/testify/require"
)

func TestShouldLogNeighborDeduplicatesUnscopedLogs(t *testing.T) {
	router := new(Router)
	ctx := context.Background()
	mac := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}

	require.False(t, router.shouldLogNeighbor(ctx, nil, "", false, false))
	require.True(t, router.shouldLogNeighbor(ctx, mac, "", true, false))
	require.False(t, router.shouldLogNeighbor(ctx, mac, "", true, false))
	require.True(t, router.shouldLogNeighbor(ctx, mac, "host-a", true, true))
	require.False(t, router.shouldLogNeighbor(ctx, mac, "host-a", true, true))
	require.True(t, router.shouldLogNeighbor(ctx, mac, "host-b", true, true))
	require.True(t, router.shouldLogNeighbor(ctx, nil, "host-only", false, true))
	require.False(t, router.shouldLogNeighbor(ctx, nil, "host-only", false, true))
}

func TestShouldLogNeighborKeepsConnectionScopedLogs(t *testing.T) {
	router := new(Router)
	ctx := log.ContextWithNewID(context.Background())
	mac := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}

	require.True(t, router.shouldLogNeighbor(ctx, mac, "host", true, true))
	require.True(t, router.shouldLogNeighbor(ctx, mac, "host", true, true))
}
