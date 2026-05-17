package transport

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sagernet/sing/common/logger"

	mDNS "github.com/miekg/dns"
)

type pipelinePool struct {
	logger                logger.ContextLogger
	enablePipeline        bool
	idleTimeout           time.Duration
	disableKeepAlive      bool
	maxQueries            int
	connections           *ConnPool[*reuseableDNSConn]
	activeConns           []*reuseableDNSConn
	activeAccess          sync.Mutex
	pipelineDetected      atomic.Int32
	consecutiveOutOfOrder int32
	outOfOrderCount       int32
	totalResponses        int32
}

func newReuseableDNSConnPool(mode ConnPoolMode, maxInflight int) *ConnPool[*reuseableDNSConn] {
	return NewConnPool(ConnPoolOptions[*reuseableDNSConn]{
		Mode:        mode,
		MaxInflight: maxInflight,
		IsAlive: func(conn *reuseableDNSConn) bool {
			select {
			case <-conn.done:
				return false
			default:
				return true
			}
		},
		Close: func(conn *reuseableDNSConn, _ error) {
			conn.Close()
		},
	})
}

func (p *pipelinePool) exchange(ctx context.Context, message *mDNS.Msg, createNewConn func(context.Context, *mDNS.Msg) (*mDNS.Msg, error)) (*mDNS.Msg, error) {
	if p.enablePipeline {
		conn := p.findAndReserveActiveConn()
		if conn != nil {
			response, err := conn.exchangeWithoutIncrement(ctx, message)
			if err == nil {
				return response, nil
			}
			if ctx.Err() != nil {
				return nil, err
			}
			p.logger.DebugContext(ctx, "retrying query after active conn failure: ", err)
		}
		conn = p.getValidConnFromPool()
		if conn != nil {
			p.reserveActiveConn(conn)
			response, err := conn.exchangeWithoutIncrement(ctx, message)
			if err == nil {
				return response, nil
			}
			if ctx.Err() != nil {
				return nil, err
			}
			p.logger.DebugContext(ctx, "retrying query on new connection after pooled conn failure: ", err)
		}
		return createNewConn(ctx, message)
	} else {
		conn := p.getValidConnFromPool()
		if conn != nil {
			response, err := conn.Exchange(ctx, message)
			if err == nil {
				return response, nil
			}
			if ctx.Err() != nil {
				return nil, err
			}
			p.logger.DebugContext(ctx, "retrying query on new connection after reused conn failure: ", err)
		}
		return createNewConn(ctx, message)
	}
}

func (p *pipelinePool) closePool() error {
	if p.connections != nil {
		return p.connections.Close()
	}
	return nil
}

func (p *pipelinePool) resetPool() {
	if p.connections != nil {
		p.connections.Reset()
	}
	p.activeAccess.Lock()
	activeConns := p.activeConns
	p.activeConns = nil
	p.activeAccess.Unlock()
	for _, conn := range activeConns {
		conn.Close()
	}
	p.pipelineDetected.Store(0)
	atomic.StoreInt32(&p.consecutiveOutOfOrder, 0)
	atomic.StoreInt32(&p.outOfOrderCount, 0)
	atomic.StoreInt32(&p.totalResponses, 0)
}

func (p *pipelinePool) getValidConnFromPool() *reuseableDNSConn {
	conn, loaded := p.connections.AcquireIdle()
	if !loaded {
		return nil
	}
	return conn
}

func (p *pipelinePool) findAndReserveActiveConn() *reuseableDNSConn {
	p.activeAccess.Lock()
	defer p.activeAccess.Unlock()

	var bestConn *reuseableDNSConn
	var minQueries int32 = -1
	var closedCount int

	for _, conn := range p.activeConns {
		select {
		case <-conn.done:
			closedCount++
		default:
			if conn.maxQueries <= 0 || conn.activeQueries.Load() < int32(conn.maxQueries) {
				current := conn.activeQueries.Load()
				if minQueries == -1 || current < minQueries {
					minQueries = current
					bestConn = conn
				}
			}
		}
	}

	if closedCount > 0 {
		validConns := make([]*reuseableDNSConn, 0, len(p.activeConns)-closedCount)
		for _, conn := range p.activeConns {
			select {
			case <-conn.done:
			default:
				validConns = append(validConns, conn)
			}
		}
		p.activeConns = validConns
	}

	if bestConn != nil {
		bestConn.activeQueries.Add(1)
	}

	return bestConn
}

func (p *pipelinePool) reserveActiveConn(conn *reuseableDNSConn) {
	p.activeAccess.Lock()
	defer p.activeAccess.Unlock()

	select {
	case <-conn.done:
	default:
		if !slices.Contains(p.activeConns, conn) {
			p.activeConns = append(p.activeConns, conn)
		}
	}
	conn.activeQueries.Add(1)
}

func (p *pipelinePool) releaseActiveConn(conn *reuseableDNSConn) int32 {
	p.activeAccess.Lock()
	defer p.activeAccess.Unlock()

	newCount := conn.activeQueries.Add(-1)
	if newCount == 0 {
		for i, c := range p.activeConns {
			if c == conn {
				last := len(p.activeConns) - 1
				p.activeConns[i] = p.activeConns[last]
				p.activeConns = p.activeConns[:last]
				break
			}
		}
	}
	return newCount
}

func (p *pipelinePool) markPipelineDetected() bool {
	return p.pipelineDetected.CompareAndSwap(0, 1)
}

func (p *pipelinePool) isPipelineDetected() bool {
	return p.pipelineDetected.Load() != 0
}

func (p *pipelinePool) getDetectionCounters() (*int32, *int32, *int32) {
	return &p.consecutiveOutOfOrder, &p.outOfOrderCount, &p.totalResponses
}
