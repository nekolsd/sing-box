package mux

import (
	"context"
	"testing"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/log"
	singMux "github.com/sagernet/sing-mux"
	M "github.com/sagernet/sing/common/metadata"

	"github.com/stretchr/testify/require"
)

func TestMuxDestination(t *testing.T) {
	require.True(t, IsMuxDestination(singMux.Destination))
	require.False(t, IsMuxDestination(M.Socksaddr{Fqdn: "example.com", Port: 443}))
}

func TestMuxLoggerPrependsParentAndUser(t *testing.T) {
	ctx := contextWithParentID(context.Background(), log.ID{ID: 42})
	ctx = adapter.WithContext(ctx, &adapter.InboundContext{User: "alice"})
	logger := new(muxContextLogger)

	require.Equal(t,
		[]any{"[parent=", uint32(42), "] ", "[", "alice", "] ", "message"},
		logger.prependContext(ctx, []any{"message"}),
	)
}
