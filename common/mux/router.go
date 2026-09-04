package mux

import (
	"context"
	"net"

	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-mux"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/logger"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

func IsMuxDestination(destination M.Socksaddr) bool {
	return destination == mux.Destination
}

type parentIDKey struct{}

func contextWithParentID(ctx context.Context, id log.ID) context.Context {
	return context.WithValue(ctx, (*parentIDKey)(nil), id)
}

func parentIDFromContext(ctx context.Context) (log.ID, bool) {
	id, loaded := ctx.Value((*parentIDKey)(nil)).(log.ID)
	return id, loaded
}

var _ logger.ContextLogger = (*muxContextLogger)(nil)

type muxContextLogger struct {
	logger.ContextLogger
}

func (l *muxContextLogger) prependContext(ctx context.Context, args []any) []any {
	var prefix []any
	if parentID, loaded := parentIDFromContext(ctx); loaded {
		prefix = append(prefix, "[parent=", parentID.ID, "] ")
	}
	if inboundCtx := adapter.ContextFrom(ctx); inboundCtx != nil && inboundCtx.User != "" {
		prefix = append(prefix, "[", inboundCtx.User, "] ")
	}
	if len(prefix) > 0 {
		return append(prefix, args...)
	}
	return args
}

func (l *muxContextLogger) TraceContext(ctx context.Context, args ...any) {
	l.ContextLogger.TraceContext(ctx, l.prependContext(ctx, args)...)
}

func (l *muxContextLogger) DebugContext(ctx context.Context, args ...any) {
	l.ContextLogger.DebugContext(ctx, l.prependContext(ctx, args)...)
}

func (l *muxContextLogger) InfoContext(ctx context.Context, args ...any) {
	l.ContextLogger.InfoContext(ctx, l.prependContext(ctx, args)...)
}

func (l *muxContextLogger) WarnContext(ctx context.Context, args ...any) {
	l.ContextLogger.WarnContext(ctx, l.prependContext(ctx, args)...)
}

func (l *muxContextLogger) ErrorContext(ctx context.Context, args ...any) {
	l.ContextLogger.ErrorContext(ctx, l.prependContext(ctx, args)...)
}

func (l *muxContextLogger) FatalContext(ctx context.Context, args ...any) {
	l.ContextLogger.FatalContext(ctx, l.prependContext(ctx, args)...)
}

func (l *muxContextLogger) PanicContext(ctx context.Context, args ...any) {
	l.ContextLogger.PanicContext(ctx, l.prependContext(ctx, args)...)
}

type Router struct {
	router  adapter.ConnectionRouterEx
	service *mux.Service
}

func NewRouterWithOptions(router adapter.ConnectionRouterEx, logger logger.ContextLogger, options option.InboundMultiplexOptions) (adapter.ConnectionRouterEx, error) {
	if !options.Enabled {
		return router, nil
	}
	var brutalOptions mux.BrutalOptions
	if options.Brutal != nil && options.Brutal.Enabled {
		brutalOptions = mux.BrutalOptions{
			Enabled:    true,
			SendBPS:    uint64(options.Brutal.UpMbps * C.MbpsToBps),
			ReceiveBPS: uint64(options.Brutal.DownMbps * C.MbpsToBps),
		}
		if brutalOptions.SendBPS < mux.BrutalMinSpeedBPS {
			return nil, E.New("brutal: invalid upload speed")
		}
		if brutalOptions.ReceiveBPS < mux.BrutalMinSpeedBPS {
			return nil, E.New("brutal: invalid download speed")
		}
	}
	wrappedLogger := &muxContextLogger{logger}
	service, err := mux.NewService(mux.ServiceOptions{
		NewStreamContext: func(ctx context.Context, conn net.Conn) context.Context {
			if parentID, loaded := log.IDFromContext(ctx); loaded {
				ctx = contextWithParentID(ctx, parentID)
			}
			return log.ContextWithNewID(ctx)
		},
		Logger:    wrappedLogger,
		HandlerEx: adapter.NewRouteContextHandler(router),
		Padding:   options.Padding,
		Brutal:    brutalOptions,
	})
	if err != nil {
		return nil, err
	}
	return &Router{router, service}, nil
}

// Deprecated: Use RouteConnectionEx instead.
func (r *Router) RouteConnection(ctx context.Context, conn net.Conn, metadata adapter.InboundContext) error {
	if metadata.Destination == mux.Destination {
		// TODO: check if WithContext is necessary
		return r.service.NewConnection(adapter.WithContext(ctx, &metadata), conn, adapter.UpstreamMetadata(metadata))
	} else {
		return r.router.RouteConnection(ctx, conn, metadata)
	}
}

// Deprecated: Use RoutePacketConnectionEx instead.
func (r *Router) RoutePacketConnection(ctx context.Context, conn N.PacketConn, metadata adapter.InboundContext) error {
	return r.router.RoutePacketConnection(ctx, conn, metadata)
}

func (r *Router) RouteConnectionEx(ctx context.Context, conn net.Conn, metadata adapter.InboundContext, onClose N.CloseHandlerFunc) {
	if metadata.Destination == mux.Destination {
		r.service.NewConnectionEx(adapter.WithContext(ctx, &metadata), conn, metadata.Source, metadata.Destination, onClose)
		return
	}
	r.router.RouteConnectionEx(ctx, conn, metadata, onClose)
}

func (r *Router) RoutePacketConnectionEx(ctx context.Context, conn N.PacketConn, metadata adapter.InboundContext, onClose N.CloseHandlerFunc) {
	r.router.RoutePacketConnectionEx(ctx, conn, metadata, onClose)
}
