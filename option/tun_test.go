package option

import (
	"testing"

	"github.com/sagernet/sing/common/json"

	"github.com/stretchr/testify/require"
)

func TestTunEndpointIndependentNatPresence(t *testing.T) {
	t.Parallel()

	var unset TunInboundOptions
	require.NoError(t, json.Unmarshal([]byte(`{}`), &unset))
	require.Nil(t, unset.EndpointIndependentNat)

	var disabled TunInboundOptions
	require.NoError(t, json.Unmarshal([]byte(`{"endpoint_independent_nat": false}`), &disabled))
	require.NotNil(t, disabled.EndpointIndependentNat)
	require.False(t, *disabled.EndpointIndependentNat)

	var enabled TunInboundOptions
	require.NoError(t, json.Unmarshal([]byte(`{"endpoint_independent_nat": true}`), &enabled))
	require.NotNil(t, enabled.EndpointIndependentNat)
	require.True(t, *enabled.EndpointIndependentNat)
}
