package tun

import (
	"testing"

	"github.com/sagernet/sing-box/option"
	"github.com/stretchr/testify/require"
)

func TestValidateExcludeICMP(t *testing.T) {
	tests := []struct {
		name             string
		options          option.TunInboundOptions
		goos             string
		nftablesDisabled bool
		expectedError    string
	}{
		{
			name: "disabled",
		},
		{
			name:          "android",
			options:       option.TunInboundOptions{ExcludeICMP: true, AutoRedirect: true},
			goos:          "android",
			expectedError: "`exclude_icmp` is only supported on Linux",
		},
		{
			name:          "unsupported platform",
			options:       option.TunInboundOptions{ExcludeICMP: true, AutoRedirect: true},
			goos:          "darwin",
			expectedError: "`exclude_icmp` is only supported on Linux",
		},
		{
			name:          "auto redirect required",
			options:       option.TunInboundOptions{ExcludeICMP: true},
			goos:          "linux",
			expectedError: "`auto_redirect` is required by `exclude_icmp`",
		},
		{
			name:             "nftables required",
			options:          option.TunInboundOptions{ExcludeICMP: true, AutoRedirect: true},
			goos:             "linux",
			nftablesDisabled: true,
			expectedError:    "`exclude_icmp` requires nftables (`DISABLE_NFTABLES` must not be true)",
		},
		{
			name:    "enabled",
			options: option.TunInboundOptions{ExcludeICMP: true, AutoRedirect: true},
			goos:    "linux",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateExcludeICMP(test.options, test.goos, test.nftablesDisabled)
			if test.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, test.expectedError)
			}
		})
	}
}
