package libbox

import (
	"crypto/x509"
	"testing"

	C "github.com/sagernet/sing-box/constant"

	"github.com/stretchr/testify/require"
)

func TestHTTPClientAppendCertificateStore(t *testing.T) {
	client := NewHTTPClient().(*httpClient)
	t.Cleanup(client.Close)
	client.tls.RootCAs = x509.NewCertPool()

	require.NoError(t, client.AppendCertificateStore(C.CertificateStoreMozilla))
	rootPool := client.tls.RootCAs
	require.NotNil(t, rootPool)
	require.False(t, rootPool.Equal(x509.NewCertPool()))
	require.Error(t, client.AppendCertificateStore("unknown"))
}
