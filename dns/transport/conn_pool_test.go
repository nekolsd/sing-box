package transport

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	mDNS "github.com/miekg/dns"
)

func TestConnPoolAcquireIdleDoesNotWaitForDialSlot(t *testing.T) {
	pool := NewConnPool(ConnPoolOptions[int]{
		Mode:        ConnPoolOrdered,
		MaxInflight: 1,
		IsAlive: func(conn int) bool {
			return true
		},
		Close: func(conn int, err error) {},
	})
	dialStarted := make(chan struct{})
	releaseDial := make(chan struct{})
	dialResult := make(chan struct {
		conn int
		err  error
	}, 1)
	go func() {
		conn, _, err := pool.Acquire(context.Background(), func(ctx context.Context) (int, error) {
			close(dialStarted)
			<-releaseDial
			return 1, nil
		})
		dialResult <- struct {
			conn int
			err  error
		}{conn, err}
	}()

	<-dialStarted
	idleResult := make(chan bool, 1)
	go func() {
		_, loaded := pool.AcquireIdle()
		idleResult <- loaded
	}()
	select {
	case loaded := <-idleResult:
		if loaded {
			t.Fatal("unexpected idle connection while the only slot is dialing")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("AcquireIdle blocked behind an in-flight dial")
	}

	close(releaseDial)
	result := <-dialResult
	if result.err != nil {
		t.Fatal(result.err)
	}
	pool.Release(result.conn, false)
}

func TestConnPoolSingleSharesConcurrentDial(t *testing.T) {
	pool := NewConnPool(ConnPoolOptions[int]{
		Mode: ConnPoolSingle,
		IsAlive: func(conn int) bool {
			return true
		},
		Close: func(conn int, err error) {},
	})
	var dialCount atomic.Int32
	dialStarted := make(chan struct{})
	releaseDial := make(chan struct{})
	dial := func(ctx context.Context) (int, error) {
		if dialCount.Add(1) == 1 {
			close(dialStarted)
		}
		<-releaseDial
		return 7, nil
	}

	const concurrency = 8
	start := make(chan struct{})
	results := make(chan int, concurrency)
	var wg sync.WaitGroup
	for range concurrency {
		wg.Go(func() {
			<-start
			conn, _, err := pool.Acquire(context.Background(), dial)
			if err != nil {
				t.Error(err)
				return
			}
			results <- conn
		})
	}
	close(start)
	<-dialStarted
	time.Sleep(10 * time.Millisecond)
	if count := dialCount.Load(); count != 1 {
		t.Fatalf("expected one shared dial before release, got %d", count)
	}
	close(releaseDial)
	wg.Wait()
	close(results)
	for conn := range results {
		if conn != 7 {
			t.Fatalf("unexpected shared connection value: %d", conn)
		}
	}
	if count := dialCount.Load(); count != 1 {
		t.Fatalf("expected one shared dial, got %d", count)
	}
}

func TestPipelinePoolReservesUnlimitedActiveConnection(t *testing.T) {
	pool := &pipelinePool{}
	conn := &reuseableDNSConn{
		done: make(chan struct{}),
	}
	pool.reserveActiveConn(conn)
	if count := conn.activeQueries.Load(); count != 1 {
		t.Fatalf("expected initial active query reservation, got %d", count)
	}
	if reserved := pool.findAndReserveActiveConn(); reserved != conn {
		t.Fatal("expected unlimited pipeline connection to be reused while active")
	}
	if count := conn.activeQueries.Load(); count != 2 {
		t.Fatalf("expected second active query reservation, got %d", count)
	}
}

func TestPipelinePoolDoesNotReserveOverMaxQueries(t *testing.T) {
	pool := &pipelinePool{}
	conn := &reuseableDNSConn{
		done:       make(chan struct{}),
		maxQueries: 1,
	}
	pool.reserveActiveConn(conn)
	if reserved := pool.findAndReserveActiveConn(); reserved != nil {
		t.Fatal("reserved a pipeline connection that was already at max_queries")
	}
	if count := conn.activeQueries.Load(); count != 1 {
		t.Fatalf("active query count changed unexpectedly: %d", count)
	}
}

func TestPipelinePoolReleaseKeepsStillActiveConnection(t *testing.T) {
	pool := &pipelinePool{}
	conn := &reuseableDNSConn{
		done: make(chan struct{}),
	}
	pool.reserveActiveConn(conn)
	pool.reserveActiveConn(conn)
	if count := pool.releaseActiveConn(conn); count != 1 {
		t.Fatalf("expected one query to remain active, got %d", count)
	}
	if reserved := pool.findAndReserveActiveConn(); reserved != conn {
		t.Fatal("connection was removed from active set before the last query completed")
	}
	if count := conn.activeQueries.Load(); count != 2 {
		t.Fatalf("expected connection to be reserved again, got %d active queries", count)
	}
}

func TestReuseableDNSConnSkipsInFlightQueryIDs(t *testing.T) {
	conn := &reuseableDNSConn{
		queryId: 0,
		callbacks: map[uint16]*dnsCallback{
			1: {},
			2: {},
		},
	}
	queryID, err := conn.nextAvailableQueryId()
	if err != nil {
		t.Fatal(err)
	}
	if queryID != 3 {
		t.Fatalf("expected query ID 3, got %d", queryID)
	}
}

func TestTCPTransportRejectsExchangeAfterClose(t *testing.T) {
	transport := &TCPTransport{}
	if err := transport.Close(); err != nil {
		t.Fatal(err)
	}
	message := new(mDNS.Msg)
	message.SetQuestion("example.com.", mDNS.TypeA)
	_, err := transport.Exchange(context.Background(), message)
	if !errors.Is(err, net.ErrClosed) {
		t.Fatalf("expected net.ErrClosed after Close, got %v", err)
	}
}
