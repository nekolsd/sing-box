package libbox

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompareSemver(t *testing.T) {
	t.Parallel()

	require.False(t, CompareSemver("1.13.0-rc.4", "1.13.0"))
	require.True(t, CompareSemver("1.13.1", "1.13.0"))
	require.False(t, CompareSemver("v1.13.0", "1.13.0"))
	require.False(t, CompareSemver("1.13.0-", "1.13.0"))
	require.True(t, CompareSemver("1.15.0-alpha.2-nekolsd-2", "1.15.0-alpha.2-nekolsd"))
	require.False(t, CompareSemver("1.15.0-alpha.2-nekolsd", "1.15.0-alpha.2-nekolsd-2"))
	require.False(t, CompareSemver("1.15.0-alpha.2-nekolsd-1", "1.15.0-alpha.2-nekolsd"))
	require.False(t, CompareSemver("1.15.0-alpha.2-nekolsd", "1.15.0-alpha.2-nekolsd-1"))
	require.True(t, CompareSemver("1.15.0-alpha.3-nekolsd", "1.15.0-alpha.2-nekolsd-9"))
	require.True(t, CompareSemver("1.15.0-nekolsd", "1.15.0-alpha.2-nekolsd-9"))
	require.True(t, CompareSemver("1.15.1-nekolsd", "1.15.0-nekolsd-5"))
	require.True(t, CompareSemver("1.15.0-nekolsd", "1.15.0"))
	require.False(t, CompareSemver("1.15.0", "1.15.0-nekolsd"))
	require.False(t, CompareSemver("1.15.0-nekolsd-0", "1.15.0-nekolsd"))
	require.False(t, CompareSemver("1.15.0-nekolsd", "1.15.0-nekolsd-0"))
	require.False(t, CompareSemver("1.15.0-nekolsd-x", "1.15.0-nekolsd"))
	require.False(t, CompareSemver("1.15.0-nekolsd-", "1.15.0-nekolsd"))
	require.False(t, CompareSemver("1.15.0-nekolsd.2", "1.15.0-nekolsd"))
}
