package limiter

import (
	"context"
	"net"
	"time"

	"github.com/sagernet/sing/common/buf"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"golang.org/x/time/rate"
)

type limiter struct {
	downloadLimiter *rate.Limiter
	uploadLimiter   *rate.Limiter
	timeout         time.Duration
}

func newLimiter(download, upload uint64, timeout time.Duration) *limiter {
	var downloadLimiter, uploadLimiter *rate.Limiter
	if download > 0 {
		downloadLimiter = rate.NewLimiter(rate.Limit(float64(download)), int(download))
	}
	if upload > 0 {
		uploadLimiter = rate.NewLimiter(rate.Limit(float64(upload)), int(upload))
	}
	return &limiter{downloadLimiter: downloadLimiter, uploadLimiter: uploadLimiter, timeout: timeout}
}

type connWithLimiter struct {
	net.Conn
	limiter *limiter
	ctx     context.Context
}

func (conn *connWithLimiter) Upstream() any {
	return conn.Conn
}

func (conn *connWithLimiter) Read(p []byte) (n int, err error) {
	if conn.limiter != nil {
		if conn.limiter.timeout > 0 {
			err = conn.Conn.SetDeadline(time.Now().Add(conn.limiter.timeout))
			if err != nil {
				return
			}
		}
		if conn.limiter.uploadLimiter != nil {
			return conn.readWithLimiter(p)
		}
	}
	return conn.Conn.Read(p)
}

func (conn *connWithLimiter) readWithLimiter(p []byte) (n int, err error) {
	if conn.limiter == nil || conn.limiter.uploadLimiter == nil {
		return conn.Conn.Read(p)
	}
	b := conn.limiter.uploadLimiter.Burst()
	if b < len(p) {
		p = p[:b]
	}
	n, err = conn.Conn.Read(p)
	if err != nil {
		return
	}
	err = conn.limiter.uploadLimiter.WaitN(conn.ctx, n)
	if err != nil {
		return
	}
	return
}

func (conn *connWithLimiter) Write(p []byte) (n int, err error) {
	if conn.limiter != nil {
		if conn.limiter.timeout > 0 {
			err = conn.Conn.SetDeadline(time.Now().Add(conn.limiter.timeout))
			if err != nil {
				return
			}
		}
		if conn.limiter.downloadLimiter != nil {
			return conn.writeWithLimiter(p)
		}
	}
	return conn.Conn.Write(p)
}

func (conn *connWithLimiter) writeWithLimiter(p []byte) (n int, err error) {
	var nn int
	b := conn.limiter.downloadLimiter.Burst()
	for {
		end := len(p)
		if end == 0 {
			break
		}
		if b < len(p) {
			end = b
		}
		err = conn.limiter.downloadLimiter.WaitN(conn.ctx, end)
		if err != nil {
			return
		}
		nn, err = conn.Conn.Write(p[:end])
		n += nn
		if err != nil {
			return
		}
		p = p[end:]
	}
	return
}

type packetConnWithLimiter struct {
	N.PacketConn
	limiter *limiter
	ctx     context.Context
}

func (conn *packetConnWithLimiter) Upstream() any {
	return conn.PacketConn
}
func (conn *packetConnWithLimiter) ReadPacket(buffer *buf.Buffer) (destination M.Socksaddr, err error) {
	if conn.limiter != nil {
		if conn.limiter.timeout > 0 {
			err = conn.PacketConn.SetDeadline(time.Now().Add(conn.limiter.timeout))
			if err != nil {
				return
			}
		}
		if conn.limiter.uploadLimiter != nil {
			return conn.readWithLimiter(buffer)
		}
	}
	return conn.PacketConn.ReadPacket(buffer)
}
func (conn *packetConnWithLimiter) readWithLimiter(buffer *buf.Buffer) (destination M.Socksaddr, err error) {
	destination, err = conn.PacketConn.ReadPacket(buffer)
	if err != nil {
		return
	}
	err = conn.limiter.uploadLimiter.WaitN(conn.ctx, buffer.Len())
	if err != nil {
		return
	}
	return
}
func (conn *packetConnWithLimiter) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) (err error) {
	if conn.limiter != nil {
		if conn.limiter.timeout > 0 {
			err = conn.PacketConn.SetDeadline(time.Now().Add(conn.limiter.timeout))
			if err != nil {
				return
			}
		}
		if conn.limiter.downloadLimiter != nil {
			return conn.writePacketWithLimiter(buffer, destination)
		}
	}
	return conn.PacketConn.WritePacket(buffer, destination)
}
func (conn *packetConnWithLimiter) writePacketWithLimiter(buffer *buf.Buffer, destination M.Socksaddr) (err error) {
	err = conn.limiter.downloadLimiter.WaitN(conn.ctx, buffer.Len())
	if err != nil {
		return
	}
	err = conn.PacketConn.WritePacket(buffer, destination)
	if err != nil {
		return
	}
	return
}
