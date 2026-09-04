package dialer

import (
	"context"
	"io"
	"net"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	M "github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/service"

	"github.com/stretchr/testify/require"
)

type concurrentDNSRouter struct {
	adapter.DNSRouter
	addresses []netip.Addr
}

func (r *concurrentDNSRouter) Lookup(context.Context, string, adapter.DNSQueryOptions) ([]netip.Addr, error) {
	return r.addresses, nil
}

func TestConcurrentFastOpenWritesAfterWinnerSelected(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })
	destination := M.SocksaddrFromNet(listener.Addr())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = service.ContextWith[adapter.DNSRouter](ctx, &concurrentDNSRouter{addresses: []netip.Addr{
		netip.MustParseAddr("127.0.0.2"), destination.Addr,
	}})
	d := NewResolveDialer(ctx, &DefaultDialer{}, false, true, "", adapter.DNSQueryOptions{}, 0)
	conn, err := d.DialContext(ctx, "tcp", M.Socksaddr{Fqdn: "example.test", Port: destination.Port})
	require.NoError(t, err)
	defer conn.Close()
	// The unreachable address must not win merely because its TFO connection
	// was constructed before any TCP handshake took place.
	require.Equal(t, destination.String(), conn.RemoteAddr().String())
	accepted, err := listener.Accept()
	require.NoError(t, err)
	defer accepted.Close()
	require.NoError(t, accepted.SetDeadline(time.Now().Add(time.Second)))
	_, err = conn.Write([]byte("hello"))
	require.NoError(t, err)
	payload := make([]byte, 5)
	_, err = io.ReadFull(accepted, payload)
	require.NoError(t, err)
	require.Equal(t, "hello", string(payload))
}

func TestConcurrentFastOpenSingleAddressRemainsLazy(t *testing.T) {
	conn, err := DialTCPConcurrent(t.Context(), &DefaultDialer{}, M.ParseSocksaddr("example.test:443"), []netip.Addr{netip.MustParseAddr("192.0.2.1")})
	require.NoError(t, err)
	defer conn.Close()
	slow, ok := conn.(*slowOpenConn)
	require.True(t, ok)
	require.Nil(t, slow.conn.Load())
	require.NoError(t, slow.ctx.Err())
}

type interfaceDialCall struct {
	strategy          *C.NetworkStrategy
	primary, fallback []C.InterfaceType
	delay             time.Duration
}

type concurrentInterfaceDialer struct {
	ParallelInterfaceDialer
	access sync.Mutex
	calls  []interfaceDialCall
	wait   sync.WaitGroup
}

func (d *concurrentInterfaceDialer) DialContext(context.Context, string, M.Socksaddr) (net.Conn, error) {
	return &trackedConn{}, nil
}

func (d *concurrentInterfaceDialer) DialParallelInterface(_ context.Context, _ string, _ M.Socksaddr, strategy *C.NetworkStrategy, primary, fallback []C.InterfaceType, delay time.Duration) (net.Conn, error) {
	d.access.Lock()
	d.calls = append(d.calls, interfaceDialCall{strategy, primary, fallback, delay})
	d.access.Unlock()
	d.wait.Done()
	return &trackedConn{}, nil
}

func (d *concurrentInterfaceDialer) ListenSerialInterfacePacket(_ context.Context, _ M.Socksaddr, strategy *C.NetworkStrategy, primary, fallback []C.InterfaceType, delay time.Duration) (net.PacketConn, error) {
	d.calls = append(d.calls, interfaceDialCall{strategy, primary, fallback, delay})
	return net.ListenPacket("udp4", "127.0.0.1:0")
}

func TestResolveConcurrentPreservesInterfaceParameters(t *testing.T) {
	for _, destination := range []string{"example.test:443", "192.0.2.1:443"} {
		t.Run(destination, func(t *testing.T) {
			ctx := service.ContextWith[adapter.DNSRouter](context.Background(), &concurrentDNSRouter{addresses: []netip.Addr{netip.MustParseAddr("192.0.2.1"), netip.MustParseAddr("192.0.2.2")}})
			upstream := new(concurrentInterfaceDialer)
			expectedCalls := 1
			if M.ParseSocksaddr(destination).IsDomain() {
				expectedCalls = 2
			}
			upstream.wait.Add(expectedCalls)
			d := NewResolveDialer(ctx, upstream, true, true, "", adapter.DNSQueryOptions{}, 0).(ParallelInterfaceDialer)
			strategy := C.NetworkStrategyFallback
			primary := []C.InterfaceType{C.InterfaceTypeWIFI}
			fallback := []C.InterfaceType{C.InterfaceTypeCellular}
			conn, err := d.DialParallelInterface(ctx, "tcp", M.ParseSocksaddr(destination), &strategy, primary, fallback, time.Second)
			require.NoError(t, err)
			defer conn.Close()
			upstream.wait.Wait()
			require.Len(t, upstream.calls, expectedCalls)
			for _, call := range upstream.calls {
				require.Equal(t, interfaceDialCall{&strategy, primary, fallback, time.Second}, call)
			}
		})
	}
}

func TestResolvePacketPreservesInterfaceParameters(t *testing.T) {
	ctx := service.ContextWith[adapter.DNSRouter](context.Background(), &concurrentDNSRouter{})
	upstream := new(concurrentInterfaceDialer)
	d := NewResolveDialer(ctx, upstream, true, true, "", adapter.DNSQueryOptions{}, 0).(ParallelInterfaceDialer)
	strategy := C.NetworkStrategyFallback
	primary := []C.InterfaceType{C.InterfaceTypeWIFI}
	fallback := []C.InterfaceType{C.InterfaceTypeCellular}
	conn, err := d.ListenSerialInterfacePacket(ctx, M.ParseSocksaddr("192.0.2.1:53"), &strategy, primary, fallback, time.Second)
	require.NoError(t, err)
	defer conn.Close()
	require.Equal(t, []interfaceDialCall{{&strategy, primary, fallback, time.Second}}, upstream.calls)
}
