package tun

import (
	"testing"

	"github.com/sagernet/sing-box/option"
	singTun "github.com/sagernet/sing-tun"

	"github.com/stretchr/testify/require"
)

func TestLegacyUDPNATModeMapping(t *testing.T) {
	testCases := []struct {
		name     string
		options  option.TunInboundOptions
		expected singTun.NATMapping
	}{
		{
			name:     "default",
			expected: singTun.NATMappingEndpointIndependent,
		},
		{
			name:     "legacy endpoint independent",
			options:  option.TunInboundOptions{UDPNATMode: "endpoint_independent"},
			expected: singTun.NATMappingEndpointIndependent,
		},
		{
			name:     "legacy destination dependent",
			options:  option.TunInboundOptions{UDPNATMode: "destination_dependent"},
			expected: singTun.NATMappingAddressAndPortDependent,
		},
		{
			name:     "legacy dnat alias",
			options:  option.TunInboundOptions{UDPNATMode: "dnat"},
			expected: singTun.NATMappingAddressAndPortDependent,
		},
		{
			name: "new non-default mapping takes precedence",
			options: option.TunInboundOptions{
				UDPMapping: option.UDPNATBehaviorAddressDependent,
				UDPNATMode: "destination_dependent",
			},
			expected: singTun.NATMappingAddressDependent,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mapping, err := udpNATMappingFromOptions(testCase.options)
			require.NoError(t, err)
			require.Equal(t, testCase.expected, mapping)
		})
	}
}

func TestLegacyUDPNATModeRejectsUnknownValue(t *testing.T) {
	_, err := udpNATMappingFromOptions(option.TunInboundOptions{UDPNATMode: "unknown"})
	require.ErrorContains(t, err, "unknown udp_nat_mode")
}
