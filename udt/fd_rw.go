package udt

import (
	"io"
	"net"
	"unsafe"
)

// #cgo CFLAGS: -Wall
// #cgo LDFLAGS: libudt.dylib
//
// #include "udt_c.h"
// #include <errno.h>
// #include <arpa/inet.h>
// #include <string.h>
import "C"

func slice2cbuf(buf []byte) *C.char {
	return (*C.char)(unsafe.Pointer(&buf[0]))
}

func (fd *udtFD) Read(p []byte) (n int, err error) {
	if err = fd.lockAndIncref(); err != nil {
		return 0, err
	}
	fd.fdmuR.Lock()
	fd.unlock()

	defer func() {
		fd.fdmuR.Unlock()
		if fd.lock() == nil {
			fd.unlockAndDecref()
		}
	}()

	if getSocketStatus(fd.sock).inTeardown() {
		return 0, io.EOF
	}

	n = int(C.udt_recv(fd.sock, slice2cbuf(p), C.int(cap(p)), 0))
	if C.int(n) == C.ERROR {
		if getSocketStatus(fd.sock).inTeardown() {
			err = io.EOF
		} else {
			err = fd.lastErrorOp("read")
		}
		// TODO if UDT_DGRAM support is implemented, revisit this logic
	} else if n == 0 {
		err = io.EOF
	}
	if err != nil && err != io.EOF {
		err = &net.OpError{"read", fd.net, fd.laddr, err}
	}
	return n, err
}

func (fd *udtFD) Write(buf []byte) (nn int, err error) {
	if err = fd.lockAndIncref(); err != nil {
		return 0, err
	}
	fd.fdmuW.Lock()
	fd.unlock()

	defer func() {
		fd.fdmuW.Unlock()
		if fd.lock() == nil {
			fd.unlockAndDecref()
		}
	}()

	nn = 0
	n := 0
	for {
		n = int(C.udt_send(fd.sock, slice2cbuf(buf[n:]), C.int(len(buf[n:])), 0))
		if C.int(n) == C.ERROR {
			err = lastError()
			break
		}
		if n > 0 {
			nn += n
		}
		if nn == len(buf) {
			break
		}
		if err != nil {
			n = 0
			break
		}
		if n == 0 {
			err = io.ErrUnexpectedEOF
			break
		}
	}
	if err != nil {
		err = &net.OpError{"write", fd.net, fd.raddr, err}
	}
	return nn, err
}

type socketStatus C.enum_UDTSTATUS

func getSocketStatus(sock C.UDTSOCKET) socketStatus {
	return socketStatus(C.udt_getsockstate(sock))
}

func (s socketStatus) inSetup() bool {
	switch C.enum_UDTSTATUS(s) {
	case C.INIT, C.OPENED, C.LISTENING, C.CONNECTING:
		return true
	}
	return false
}

func (s socketStatus) inTeardown() bool {
	switch C.enum_UDTSTATUS(s) {
	case C.BROKEN, C.CLOSING, C.CLOSED, C.NONEXIST:
		return true
	}
	return false
}

func (s socketStatus) inConnected(sock C.UDTSOCKET) bool {
	return C.enum_UDTSTATUS(s) == C.CONNECTED
}
