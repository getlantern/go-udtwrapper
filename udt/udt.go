package udt

// #cgo LDFLAGS: libudt.dylib
//
// #include "udt_c.h"
// #include <arpa/inet.h>
// #include <string.h>
import "C"

import (
// "fmt"
// "strconv"
// "strings"
// "syscall"
// "unsafe"

// sockaddr "github.com/jbenet/go-sockaddr"
// sockaddrnet "github.com/jbenet/go-sockaddr/net"
)

// type socket C.UDTSOCKET

// func (s *socket) readFrom(b []byte) (n int, sa syscall.Sockaddr, err error) {
// }

// func (s *socket) readMsg(b []byte, oob []byte) (n int, oobn int, flags int, sa syscall.Sockaddr, err error) {
// }

// func (s *socket) writeTo(b []byte) (n int, err error) {
// }

// func (s *socket) writeMsg(b []byte) (n int, oobn int, err error) {
// }

// func dial(laddr, raddr *UDTAddr) (*UDTConn, error) {

// 	sock, err := socket(laddr)
// 	if err != nil {
// 		return 0, err
// 	}

// 	af, sa, err := raddr.socketArgs()
// 	if err != nil {
// 		return 0, err
// 	}

// 	csa := &(C.struct_sockaddr_any(sa).addr)
// 	if udt_connect(sock, csa, C.int(sa.Len)) != 0 {
// 		return 0, fmt.Erorrf("failed to connect to: %s, %s", raddr, lastError())
// 	}

// 	return sock, nil
// }
