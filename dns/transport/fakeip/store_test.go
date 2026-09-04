package fakeip

import (
	"context"
	"net/netip"
	"path/filepath"
	"testing"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/experimental/cachefile"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/logger"
	"github.com/sagernet/sing/service"

	"github.com/stretchr/testify/require"
)

func TestStoreCreateSkipsOccupiedAddress(t *testing.T) {
	store := NewStore(context.Background(), logger.NOP(), netip.MustParsePrefix("198.18.0.0/29"), netip.Prefix{})
	require.NoError(t, store.Start())

	occupiedAddress := netip.MustParseAddr("198.18.0.2")
	require.NoError(t, store.storage.FakeIPStore(occupiedAddress, "occupied.example"))

	address, err := store.Create("new.example", false)
	require.NoError(t, err)
	require.Equal(t, netip.MustParseAddr("198.18.0.3"), address)

	domain, loaded := store.storage.FakeIPLoad(occupiedAddress)
	require.True(t, loaded)
	require.Equal(t, "occupied.example", domain)
}

func TestStoreCreateReturnsErrorWhenAddressRangeExhausted(t *testing.T) {
	store := NewStore(context.Background(), logger.NOP(), netip.MustParsePrefix("198.18.0.0/30"), netip.Prefix{})
	require.NoError(t, store.Start())

	require.NoError(t, store.storage.FakeIPStore(netip.MustParseAddr("198.18.0.2"), "occupied.example"))

	_, err := store.Create("new.example", false)
	require.ErrorContains(t, err, "address range is exhausted")
}

func TestStoreStartUsesCacheHighWatermark(t *testing.T) {
	cacheFile := cachefile.New(context.Background(), logger.NOP(), option.CacheFileOptions{
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
	require.NoError(t, cacheFile.FakeIPStore(netip.MustParseAddr("198.18.0.5"), "cached.example"))

	ctx := service.ContextWith[adapter.CacheFile](context.Background(), cacheFile)
	store := NewStore(ctx, logger.NOP(), inet4Range, netip.Prefix{})
	require.NoError(t, store.Start())

	address, err := store.Create("new.example", false)
	require.NoError(t, err)
	require.Equal(t, netip.MustParseAddr("198.18.0.6"), address)
}
