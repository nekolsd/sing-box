package certificate

import (
	"crypto/x509"
	"testing"

	C "github.com/sagernet/sing-box/constant"

	"github.com/stretchr/testify/require"
)

func TestIncludedPEM(t *testing.T) {
	for _, store := range []string{C.CertificateStoreMozilla, C.CertificateStoreChrome} {
		pemContent, err := IncludedPEM(store)
		require.NoError(t, err)
		require.NotEmpty(t, pemContent)
		pool := x509.NewCertPool()
		require.True(t, pool.AppendCertsFromPEM([]byte(pemContent)))
	}

	_, err := IncludedPEM("unknown")
	require.ErrorContains(t, err, "unknown certificate store")
}
