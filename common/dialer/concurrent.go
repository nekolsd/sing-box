package dialer

import (
	"context"
	"net"
	"net/netip"

	"github.com/sagernet/sing/common"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

// DialTCPConcurrent races all resolved addresses and returns the first
// successful TCP connection. Successful loser connections are closed.
func DialTCPConcurrent(ctx context.Context, dialer N.Dialer, destination M.Socksaddr, destinationAddresses []netip.Addr) (net.Conn, error) {
	return dialTCPConcurrent(ctx, dialer.DialContext, destination, destinationAddresses)
}

func dialTCPConcurrent(ctx context.Context, dial func(context.Context, string, M.Socksaddr) (net.Conn, error), destination M.Socksaddr, destinationAddresses []netip.Addr) (net.Conn, error) {
	if len(destinationAddresses) == 0 {
		return nil, E.New("no addresses")
	}
	if len(destinationAddresses) == 1 {
		return dial(ctx, N.NetworkTCP, M.SocksaddrFrom(destinationAddresses[0], destination.Port))
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	returned := make(chan struct{})
	defer close(returned)
	type dialResult struct {
		net.Conn
		error
	}
	results := make(chan dialResult)
	for _, address := range destinationAddresses {
		go func() {
			conn, err := dial(ctx, N.NetworkTCP, M.SocksaddrFrom(address, destination.Port))
			if err == nil {
				// TFO connections normally connect on their first Write. Complete
				// that handshake before choosing a winner and canceling the racers.
				if slowConn, loaded := common.Cast[*slowOpenConn](conn); loaded {
					_, err = slowConn.Write(nil)
					if err != nil {
						_ = conn.Close()
					}
				}
			}
			if err != nil {
				select {
				case results <- dialResult{error: err}:
				case <-returned:
				}
				return
			}
			select {
			case <-returned:
				_ = conn.Close()
			case results <- dialResult{Conn: conn}:
			}
		}()
	}
	var connErrors []error
	for range destinationAddresses {
		result := <-results
		if result.error == nil {
			cancel()
			return result.Conn, nil
		}
		connErrors = append(connErrors, result.error)
	}
	return nil, E.Errors(connErrors...)
}
