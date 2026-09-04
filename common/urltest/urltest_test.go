package urltest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateLink(t *testing.T) {
	require.Equal(t, DefaultURLTestLink, ValidateLink(""))
	require.Equal(t, DefaultURLTestLink, ValidateLink("ftp://example.com/test"))
	require.Equal(t, "http://example.com/test", ValidateLink("http://example.com/test"))
	require.Equal(t, "HTTPS://example.com/test", ValidateLink("HTTPS://example.com/test"))
}
