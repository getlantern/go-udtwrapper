package udt

import (
	"net"

	// sockaddr "github.com/jbenet/go-sockaddr"
	// sockaddrnet "github.com/jbenet/go-sockaddr/net"
)

// ListenUDT listens for incoming UDT packets addressed to the local
// address laddr.  Net must be "udt", "udt4", or "udt6".  If laddr has
// a port of 0, ListenUDT will choose an available port.
// The LocalAddr method of the returned UDTConn can be used to
// discover the port.  The returned connection's ReadFrom and WriteTo
// methods can be used to receive and send UDT packets with per-packet
// addressing.
func ListenUDT(network string, laddr *UDTAddr) (*UDTConn, error) {
	switch network {
	case "udt", "udt4", "udt6":
	default:
		return nil, &net.OpError{Op: "listen", Net: network, Addr: laddr, Err: net.UnknownNetworkError(network)}
	}
	if laddr == nil {
		laddr = &UDTAddr{addr: &net.UDPAddr{}}
	}
	panic("not implemented yet")
	// c, err := listen(laddr)
	// if err != nil {
	// 	return nil, err
	// }
	// c.net = network
	// return c
}
