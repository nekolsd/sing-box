package dialer

import (
	"context"
	"errors"
	"io"
	"net"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	M "github.com/sagernet/sing/common/metadata"
)

type dialBehavior struct {
	delay     time.Duration
	err       error
	ignoreCtx bool
}

type testConcurrentDialer struct {
	behaviors map[string]dialBehavior

	connectionsMu sync.Mutex
	connections   []*trackedConn
	cancelCount   atomic.Int32
	barrier       dialBarrier
}

type dialBarrier struct {
	targets map[string]bool
	count   int32
	started atomic.Int32
	release chan struct{}
}

func (d *testConcurrentDialer) DialContext(ctx context.Context, _ string, destination M.Socksaddr) (net.Conn, error) {
	d.waitBarrier(destination.Addr.String())
	behavior, exists := d.behaviors[destination.Addr.String()]
	if !exists {
		return nil, errors.New("missing behavior")
	}
	if behavior.ignoreCtx {
		time.Sleep(behavior.delay)
	} else {
		select {
		case <-time.After(behavior.delay):
		case <-ctx.Done():
			d.cancelCount.Add(1)
			return nil, ctx.Err()
		}
	}
	if behavior.err != nil {
		return nil, behavior.err
	}
	conn := &trackedConn{}
	d.connectionsMu.Lock()
	d.connections = append(d.connections, conn)
	d.connectionsMu.Unlock()
	return conn, nil
}

func (d *testConcurrentDialer) ListenPacket(context.Context, M.Socksaddr) (net.PacketConn, error) {
	return nil, errors.New("not implemented")
}

func (d *testConcurrentDialer) waitBarrier(address string) {
	if d.barrier.release == nil || !d.barrier.targets[address] {
		return
	}
	started := d.barrier.started.Add(1)
	if started == d.barrier.count {
		close(d.barrier.release)
	}
	<-d.barrier.release
}

func (d *testConcurrentDialer) snapshotConnections() []*trackedConn {
	d.connectionsMu.Lock()
	defer d.connectionsMu.Unlock()
	return append([]*trackedConn(nil), d.connections...)
}

type trackedConn struct {
	closed atomic.Bool
}

func (c *trackedConn) Read([]byte) (int, error)         { return 0, io.EOF }
func (c *trackedConn) Write(p []byte) (int, error)      { return len(p), nil }
func (c *trackedConn) Close() error                     { c.closed.Store(true); return nil }
func (c *trackedConn) LocalAddr() net.Addr              { return &net.TCPAddr{} }
func (c *trackedConn) RemoteAddr() net.Addr             { return &net.TCPAddr{} }
func (c *trackedConn) SetDeadline(time.Time) error      { return nil }
func (c *trackedConn) SetReadDeadline(time.Time) error  { return nil }
func (c *trackedConn) SetWriteDeadline(time.Time) error { return nil }

func TestDialTCPConcurrentCloseLoserConnections(t *testing.T) {
	dialer := &testConcurrentDialer{
		behaviors: map[string]dialBehavior{
			"198.18.0.1": {delay: 10 * time.Millisecond, ignoreCtx: true},
			"198.18.0.2": {delay: 120 * time.Millisecond, ignoreCtx: true},
			"198.18.0.3": {delay: 400 * time.Millisecond},
		},
		barrier: dialBarrier{
			targets: map[string]bool{
				"198.18.0.1": true,
				"198.18.0.2": true,
			},
			count:   2,
			release: make(chan struct{}),
		},
	}
	addresses := []netip.Addr{
		netip.MustParseAddr("198.18.0.1"),
		netip.MustParseAddr("198.18.0.2"),
		netip.MustParseAddr("198.18.0.3"),
	}
	destination := M.ParseSocksaddr("example.com:443")

	conn, err := DialTCPConcurrent(context.Background(), dialer, destination, addresses)
	if err != nil {
		t.Fatalf("DialTCPConcurrent returned error: %v", err)
	}

	winner, ok := conn.(*trackedConn)
	if !ok {
		t.Fatalf("unexpected winner conn type: %T", conn)
	}

	var connections []*trackedConn
	waitUntil(t, 700*time.Millisecond, func() bool {
		connections = dialer.snapshotConnections()
		return len(connections) >= 2
	})

	waitUntil(t, 500*time.Millisecond, func() bool {
		for _, connection := range connections {
			if connection == winner {
				continue
			}
			if !connection.closed.Load() {
				return false
			}
		}
		return true
	})

	if winner.closed.Load() {
		t.Fatal("winner connection should stay open")
	}
	_ = winner.Close()
}

func TestDialTCPConcurrentCancelInflightLoserDial(t *testing.T) {
	dialer := &testConcurrentDialer{
		behaviors: map[string]dialBehavior{
			"198.18.1.1": {delay: 10 * time.Millisecond, ignoreCtx: true},
			"198.18.1.2": {delay: 2 * time.Second},
		},
	}
	addresses := []netip.Addr{
		netip.MustParseAddr("198.18.1.1"),
		netip.MustParseAddr("198.18.1.2"),
	}
	destination := M.ParseSocksaddr("example.com:443")

	conn, err := DialTCPConcurrent(context.Background(), dialer, destination, addresses)
	if err != nil {
		t.Fatalf("DialTCPConcurrent returned error: %v", err)
	}
	_ = conn.Close()

	waitUntil(t, 700*time.Millisecond, func() bool {
		return dialer.cancelCount.Load() >= 1
	})
}

func TestDialTCPConcurrentAllFail(t *testing.T) {
	errA := errors.New("connect refused A")
	errB := errors.New("connect refused B")
	errC := errors.New("timeout C")
	dialer := &testConcurrentDialer{
		behaviors: map[string]dialBehavior{
			"198.18.2.1": {delay: 10 * time.Millisecond, err: errA},
			"198.18.2.2": {delay: 50 * time.Millisecond, err: errB},
			"198.18.2.3": {delay: 30 * time.Millisecond, err: errC},
		},
	}
	addresses := []netip.Addr{
		netip.MustParseAddr("198.18.2.1"),
		netip.MustParseAddr("198.18.2.2"),
		netip.MustParseAddr("198.18.2.3"),
	}
	destination := M.ParseSocksaddr("example.com:443")

	conn, err := DialTCPConcurrent(context.Background(), dialer, destination, addresses)
	if conn != nil {
		_ = conn.Close()
		t.Fatal("expected nil conn when all dials fail")
	}
	if err == nil {
		t.Fatal("expected error when all dials fail")
	}
	for _, target := range []error{errA, errB, errC} {
		if !errors.Is(err, target) {
			t.Fatalf("combined error should contain %v, got: %v", target, err)
		}
	}
	if connections := dialer.snapshotConnections(); len(connections) != 0 {
		t.Fatalf("expected no connections created, got %d", len(connections))
	}
}

func waitUntil(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}
