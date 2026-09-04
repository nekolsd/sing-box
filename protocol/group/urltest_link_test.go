package group

import (
	"context"
	"testing"

	"github.com/sagernet/sing-box/common/urltest"
	"github.com/sagernet/sing-box/option"

	"github.com/stretchr/testify/require"
)

func TestOutboundGroupURLTestLinks(t *testing.T) {
	selectorOutbound, err := NewSelector(context.Background(), nil, nil, "selector", option.SelectorOutboundOptions{
		URL: "https://selector.example/generate_204",
	})
	require.NoError(t, err)
	require.Equal(t, "https://selector.example/generate_204", selectorOutbound.(*Selector).URLTestLink())

	urlTestOutbound, err := NewURLTest(context.Background(), nil, nil, "urltest", option.URLTestOutboundOptions{
		URL: "invalid",
	})
	require.NoError(t, err)
	require.Equal(t, urltest.DefaultURLTestLink, urlTestOutbound.(*URLTest).URLTestLink())

	loadBalanceOutbound, err := NewLoadBalance(context.Background(), nil, nil, "load-balance", option.LoadBalanceOutboundOptions{})
	require.NoError(t, err)
	require.Equal(t, urltest.DefaultURLTestLink, loadBalanceOutbound.(*LoadBalance).URLTestLink())
}
