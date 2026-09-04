//go:build with_gvisor

package tailscale

import (
	"context"
	"testing"

	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"

	"github.com/stretchr/testify/require"
)

func TestDNSTransportTCPConcurrentOption(t *testing.T) {
	transport, err := NewDNSTransport(
		context.Background(),
		log.NewNOPFactory().Logger(),
		"test",
		option.TailscaleDNSServerOptions{
			Endpoint:      "tailscale-endpoint",
			TCPConcurrent: true,
		},
	)
	require.NoError(t, err)
	require.True(t, transport.(*DNSTransport).tcpConcurrent)
}
