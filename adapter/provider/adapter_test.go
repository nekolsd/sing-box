package provider

import (
	"context"
	"testing"

	"github.com/sagernet/sing-box/option"

	"github.com/stretchr/testify/require"
)

func TestAdapterCloseCancelsContext(t *testing.T) {
	adapter := NewAdapter(context.Background(), nil, nil, nil, nil, nil, "test", "test", option.ProviderHealthCheckOptions{})

	require.NoError(t, adapter.Close())
	require.ErrorIs(t, adapter.ctx.Err(), context.Canceled)
}
