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

func TestStoreRestartPreservesCachedMappings(t *testing.T) {
	for _, prefix := range []string{"198.18.0.0/29", "fd00::/125"} {
		t.Run(prefix, func(t *testing.T) {
			addressRange := netip.MustParsePrefix(prefix)
			isIPv6 := addressRange.Addr().Is6()
			first := addressRange.Addr().Next().Next()
			occupied := map[netip.Addr]string{
				first.Next():               "first.example",
				first.Next().Next().Next(): "second.example",
			}
			cacheOptions := option.CacheFileOptions{
				Path:        filepath.Join(t.TempDir(), "cache.db"),
				StoreFakeIP: true,
			}
			cacheFile := cachefile.New(context.Background(), logger.NOP(), cacheOptions)
			require.NoError(t, cacheFile.Start(adapter.StartStateInitialize))
			t.Cleanup(func() {
				require.NoError(t, cacheFile.Close())
			})

			metadata := &adapter.FakeIPMetadata{}
			if isIPv6 {
				metadata.Inet6Range, metadata.Inet6Current = addressRange, first
			} else {
				metadata.Inet4Range, metadata.Inet4Current = addressRange, first
			}
			require.NoError(t, cacheFile.FakeIPSaveMetadata(metadata))
			for address, domain := range occupied {
				require.NoError(t, cacheFile.FakeIPStore(address, domain))
			}
			require.NoError(t, cacheFile.Close())

			// Reopen mappings persisted beyond the saved allocation cursor.
			cacheFile = cachefile.New(context.Background(), logger.NOP(), cacheOptions)
			require.NoError(t, cacheFile.Start(adapter.StartStateInitialize))
			ctx := service.ContextWith[adapter.CacheFile](context.Background(), cacheFile)
			store := NewStore(ctx, logger.NOP(), metadata.Inet4Range, metadata.Inet6Range)
			require.NoError(t, store.Start())

			// Fill the remaining slots, including the slot reached by wrapping.
			var allocated []netip.Addr
			for _, domain := range []string{"new1.example", "new2.example", "new3.example"} {
				address, err := store.Create(domain, isIPv6)
				require.NoError(t, err)
				allocated = append(allocated, address)
				occupied[address] = domain
			}
			require.ElementsMatch(t, []netip.Addr{first, first.Next().Next(), first.Next().Next().Next().Next()}, allocated)
			_, err := store.Create("exhausted.example", isIPv6)
			require.ErrorContains(t, err, "address range is exhausted")
			for address, domain := range occupied {
				actualDomain, loaded := store.Lookup(address)
				require.True(t, loaded)
				require.Equal(t, domain, actualDomain)
				actualAddress, err := store.Create(domain, isIPv6)
				require.NoError(t, err)
				require.Equal(t, address, actualAddress)
			}
		})
	}
}
