package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing/common/x/list"

	"github.com/stretchr/testify/require"
)

type closeErrorProvider struct {
	err error
}

func (p *closeErrorProvider) Type() string {
	return "test"
}

func (p *closeErrorProvider) Tag() string {
	return "provider"
}

func (p *closeErrorProvider) Outbounds() []adapter.Outbound {
	return nil
}

func (p *closeErrorProvider) Outbound(string) (adapter.Outbound, bool) {
	return nil, false
}

func (p *closeErrorProvider) UpdatedAt() time.Time {
	return time.Time{}
}

func (p *closeErrorProvider) HealthCheck(context.Context) (map[string]uint16, error) {
	return nil, nil
}

func (p *closeErrorProvider) RegisterCallback(adapter.ProviderUpdateCallback) *list.Element[adapter.ProviderUpdateCallback] {
	return nil
}

func (p *closeErrorProvider) UnregisterCallback(*list.Element[adapter.ProviderUpdateCallback]) {
}

func (p *closeErrorProvider) Close() error {
	return p.err
}

func TestManagerCloseReturnsProviderError(t *testing.T) {
	expectedErr := errors.New("close failed")
	manager := NewManager(context.Background(), log.NewNOPFactory().NewLogger("provider"), nil)
	manager.started = true
	manager.providers = []adapter.Provider{&closeErrorProvider{err: expectedErr}}

	err := manager.Close()

	require.ErrorContains(t, err, expectedErr.Error())
}
