package clashapi

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/urltest"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/protocol/group"
	M "github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/service"
)

type delayTestManager struct {
	adapter.OutboundManager
	outbounds map[string]adapter.Outbound
}

func (m *delayTestManager) Outbound(tag string) (adapter.Outbound, bool) {
	outbound, ok := m.outbounds[tag]
	return outbound, ok
}

type delayTestOutbound struct {
	adapter.Outbound
	tag   string
	delay time.Duration
}

func (o *delayTestOutbound) Tag() string       { return o.tag }
func (o *delayTestOutbound) Network() []string { return []string{"tcp", "udp"} }
func (o *delayTestOutbound) DialContext(ctx context.Context, _ string, _ M.Socksaddr) (net.Conn, error) {
	select {
	case <-time.After(o.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	client, server := net.Pipe()
	go func() {
		defer server.Close()
		_ = server.SetDeadline(time.Now().Add(time.Second))
		if req, err := http.ReadRequest(bufio.NewReader(server)); err == nil {
			req.Body.Close()
			fmt.Fprint(server, "HTTP/1.1 204 No Content\r\nConnection: close\r\n\r\n")
		}
	}()
	return client, nil
}

func TestGroupDelayUpdatesSelection(t *testing.T) {
	for _, requestedURL := range []string{"", "http://example.test/generate_204", "http://different.test/check"} {
		label := requestedURL
		if label == "" {
			label = "without URL"
		}
		t.Run(label, func(t *testing.T) {
			history := urltest.NewHistoryStorage()
			manager := &delayTestManager{outbounds: map[string]adapter.Outbound{
				"slow": &delayTestOutbound{tag: "slow", delay: 80 * time.Millisecond},
				"fast": &delayTestOutbound{tag: "fast", delay: 5 * time.Millisecond},
			}}
			ctx := service.ContextWith[adapter.OutboundManager](context.Background(), manager)
			ctx = service.ContextWithPtr(ctx, history)
			logger := log.NewNOPFactory().NewLogger("audit")
			const link = "http://example.test/generate_204"
			outbound, err := group.NewURLTest(ctx, nil, logger, "auto", option.URLTestOutboundOptions{
				GroupCommonOption: option.GroupCommonOption{Outbounds: []string{"slow", "fast"}},
				URL:               link, Tolerance: 1,
			})
			if err != nil {
				t.Fatal(err)
			}
			auto := outbound.(*group.URLTest)
			if err := auto.Start(); err != nil {
				t.Fatal(err)
			}
			defer auto.Close()
			history.StoreURLTestHistory("slow", &adapter.URLTestHistory{Time: time.Now(), Delay: 1})
			history.StoreURLTestHistory("fast", &adapter.URLTestHistory{Time: time.Now(), Delay: 200})
			auto.PerformUpdateCheck()
			if auto.Now() != "slow" {
				t.Fatal("initial selection should be slow")
			}
			query := "/group/auto/delay?timeout=1000"
			if requestedURL != "" {
				query += "&url=" + url.QueryEscape(requestedURL)
			}
			request := httptest.NewRequest(http.MethodGet, query, nil)
			request = request.WithContext(context.WithValue(request.Context(), CtxKeyProxy, outbound))
			response := httptest.NewRecorder()
			server := &Server{outbound: manager, urlTestHistory: history, logger: logger}
			getGroupDelay(server)(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("HTTP %d: %s", response.Code, response.Body.String())
			}
			if auto.URLTestLink() != link {
				t.Fatal("request URL changed the configured group URL")
			}
			if auto.Now() != "fast" {
				t.Fatalf("delay API returned %s, but selected outbound remains %q", response.Body.String(), auto.Now())
			}
		})
	}
}
