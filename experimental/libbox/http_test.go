package libbox

import (
	"testing"

	C "github.com/sagernet/sing-box/constant"

	"github.com/stretchr/testify/require"
)

func TestHTTPClientAppendCertificateStore(t *testing.T) {
	client := NewHTTPClient()
	t.Cleanup(client.Close)

	require.NoError(t, client.AppendCertificateStore(C.CertificateStoreMozilla))
	rootPool := client.(*httpClient).tls.RootCAs
	require.NotNil(t, rootPool)
	require.NotEmpty(t, rootPool.Subjects())
	require.Error(t, client.AppendCertificateStore("unknown"))
}
