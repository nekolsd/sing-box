package libbox

import (
	"testing"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-tun"

	"github.com/stretchr/testify/require"
)

func TestAutoRedirectExcludeICMPRoundTrip(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		content, err := encodeAutoRedirectOptions(adapter.AutoRedirectOptions{
			TunOptions:   &tun.Options{Name: "tun0", ExcludeICMP: enabled},
			TableName:    "sing-box",
			RedirectPort: 12345,
		})
		require.NoError(t, err)
		decoded, table, port, err := decodeAutoRedirectOptions(content)
		require.NoError(t, err)
		require.Equal(t, enabled, decoded.ExcludeICMP, "platform redirect must retain the ICMP policy")
		require.Equal(t, "sing-box", table)
		require.EqualValues(t, 12345, port)
	}
}

func TestAutoRedirectLegacyOptionsKeepICMPEnabled(t *testing.T) {
	decoded, _, _, err := decodeAutoRedirectOptions([]byte(`{"interface_name":"tun0","table_name":"sing-box","redirect_port":12345}`))
	require.NoError(t, err)
	require.False(t, decoded.ExcludeICMP)
}
