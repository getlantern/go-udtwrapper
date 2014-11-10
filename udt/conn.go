package udt

import (
// "net"

// sockaddr "github.com/jbenet/go-sockaddr"
// sockaddrnet "github.com/jbenet/go-sockaddr/net"
)

// UDTConn is the implementation of the Conn and PacketConn interfaces
// for UDT network connections.
type UDTConn struct {
	*udtFD
	net string
}

func newUDTConn(fd *udtFD, net string) *UDTConn {
	return &UDTConn{udtFD: fd, net: net}
}

// // ReadFromUDT reads a UDT packet from c, copying the payload into b.
// // It returns the number of bytes copied into b and the return address
// // that was on the packet.
// func (c *UDTConn) ReadFromUDT(b []byte) (n int, addr *UDTAddr, err error) {
// 	if !c.ok() {
// 		return 0, nil, syscall.EINVAL
// 	}
// 	n, sa, err := c.sock.readFrom(b)
// 	if sa != nil {
// 		udpaddr := sockaddrnet.SockaddrToNetAddr(sa)
// 		addr = &UDTAddr{addr: udpaddr}
// 	}
// 	return
// }

// // ReadFrom implements the PacketConn ReadFrom method.
// func (c *UDTConn) ReadFrom(b []byte) (int, Addr, error) {
// 	if !c.ok() {
// 		return 0, nil, syscall.EINVAL
// 	}
// 	n, addr, err := c.ReadFromUDT(b)
// 	return n, addr.toAddr(), err
// }

// // ReadMsgUDT reads a packet from c, copying the payload into b and
// // the associated out-of-band data into oob.  It returns the number
// // of bytes copied into b, the number of bytes copied into oob, the
// // flags that were set on the packet and the source address of the
// // packet.
// func (c *UDTConn) ReadMsgUDT(b, oob []byte) (n, oobn, flags int, addr *UDTAddr, err error) {
// 	if !c.ok() {
// 		return 0, 0, 0, nil, syscall.EINVAL
// 	}
// 	var sa syscall.Sockaddr
// 	n, oobn, flags, sa, err = c.sock.readMsg(b, oob)
// 	if sa != nil {
// 		udpaddr := sockaddrnet.SockaddrToNetAddr(sa)
// 		addr = &UDTAddr{addr: udpaddr}
// 	}
// 	return
// }

// // WriteToUDT writes a UDT packet to addr via c, copying the payload
// // from b.
// func (c *UDTConn) WriteToUDT(b []byte, addr *UDTAddr) (int, error) {
// 	if !c.ok() {
// 		return 0, syscall.EINVAL
// 	}
// 	if c.sock != 0 {
// 		return 0, &net.OpError{"write", c.net, addr, ErrWriteToConnected}
// 	}
// 	if addr == nil {
// 		return 0, &net.OpError{Op: "write", Net: c.net, Addr: nil, Err: errMissingAddress}
// 	}
// 	sa, err := sockaddrnet.NetAddrToSockaddr(addr)
// 	if err != nil {
// 		return 0, &net.OpError{"write", c.net, addr, err}
// 	}
// 	return c.sock.writeTo(b, sa)
// }

// // WriteTo implements the PacketConn WriteTo method.
// func (c *UDTConn) WriteTo(b []byte, addr Addr) (int, error) {
// 	if !c.ok() {
// 		return 0, syscall.EINVAL
// 	}
// 	a, ok := addr.(*UDTAddr)
// 	if !ok {
// 		return 0, &net.OpError{"write", c.net, addr, syscall.EINVAL}
// 	}
// 	return c.WriteToUDT(b, a)
// }

// // WriteMsgUDT writes a packet to addr via c, copying the payload from
// // b and the associated out-of-band data from oob.  It returns the
// // number of payload and out-of-band bytes written.
// func (c *UDTConn) WriteMsgUDT(b, oob []byte, addr *UDTAddr) (n, oobn int, err error) {
// 	if !c.ok() {
// 		return 0, 0, syscall.EINVAL
// 	}
// 	if c.sock != 0 {
// 		return 0, 0, &net.OpError{"write", c.net, addr, ErrWriteToConnected}
// 	}
// 	if addr == nil {
// 		return 0, 0, &net.OpError{Op: "write", Net: c.net, Addr: nil, Err: errMissingAddress}
// 	}
// 	sa, err := sockaddrnet.NetAddrToSockaddr(addr)
// 	if err != nil {
// 		return 0, 0, &net.OpError{"write", c.net, addr, err}
// 	}
// 	return c.sock.writeMsg(b, oob, sa)
// }
