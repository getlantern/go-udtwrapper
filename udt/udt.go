package udt

// #cgo LDFLAGS: libudt.dylib
//
// #include "udt_c.h"
// #include <arpa/inet.h>
// #include <string.h>
import "C"

import (
	"fmt"
	"strconv"
	"strings"
	"unsafe"
)

type Socket struct {
	sock C.UDTSOCKET
}

func Dial(network string, address string) (socket *Socket, err error) {
	var n C.int
	if network == "ip4" {
		n = C.AF_INET
	} else if network == "ip6" {
		n = C.AF_INET6
	} else {
		return nil, fmt.Errorf("network must be either ip4 or ip6")
	}
	sock := C.udt_socket(n, C.SOCK_STREAM, 0)
	if sock == C.INVALID_SOCK {
		return nil, fmt.Errorf("Invalid socket: %s", C.GoString(C.udt_getlasterror_desc()))
	}

	splitAddr := strings.Split(address, ":")
	if len(splitAddr) != 2 {
		return nil, fmt.Errorf("Please specify an address as host:port")
	}
	host, _port := splitAddr[0], splitAddr[1]
	port, err := strconv.Atoi(_port)
	if err != nil {
		return nil, fmt.Errorf("Invalid port: %s", _port)
	}
	var serv_addr C.struct_sockaddr_in
	serv_addr.sin_family = C.sa_family_t(n)
	serv_addr.sin_port = C.in_port_t(C._htons(C.uint16_t(port)))
	if _, err := C.inet_pton(C.int(n), C.CString(host), unsafe.Pointer(&serv_addr.sin_addr)); err != nil {
		return nil, fmt.Errorf("Unable to convert IP address: %s", err)
	}
	if _, err := C.memset(unsafe.Pointer(&(serv_addr.sin_zero)), 0, 8); err != nil {
		return nil, fmt.Errorf("Unable to zero sin_zero")
	}

	socket = &Socket{
		sock: sock,
	}
	return
}
