package cachefile

import (
	"context"
	"net/netip"
	"path/filepath"
	"testing"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/logger"
	"github.com/stretchr/testify/require"
)

func TestFakeIPMetadataAppliesHighWatermark(t *testing.T) {
	cacheFile := New(context.Background(), logger.NOP(), option.CacheFileOptions{
		Path:        filepath.Join(t.TempDir(), "cache.db"),
		StoreFakeIP: true,
	})
	require.NoError(t, cacheFile.Start(adapter.StartStateInitialize))
	t.Cleanup(func() {
		require.NoError(t, cacheFile.Close())
	})

	inet4Range := netip.MustParsePrefix("198.18.0.0/29")
	require.NoError(t, cacheFile.FakeIPSaveMetadata(&adapter.FakeIPMetadata{
		Inet4Range:   inet4Range,
		Inet4Current: netip.MustParseAddr("198.18.0.2"),
	}))
	require.NoError(t, cacheFile.FakeIPStore(netip.MustParseAddr("198.18.0.5"), "high.example"))

	metadata := cacheFile.FakeIPMetadata()
	require.NotNil(t, metadata)
	require.Equal(t, netip.MustParseAddr("198.18.0.5"), metadata.Inet4Current)
}
