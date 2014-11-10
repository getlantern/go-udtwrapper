package udt

import (
	"io"
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
		fd.lock()
		fd.unlockAndDecref()
	}()

	if getSocketStatus(fd.sock).inTeardown() {
		return 0, io.EOF
	}

	n = int(C.udt_recv(fd.sock, slice2cbuf(p), C.int(cap(p)), 0))
	if C.int(n) == C.ERROR {
		if getSocketStatus(fd.sock).inTeardown() {
			return 0, io.EOF
		}
		return 0, fd.lastErrorOp("read")
	}
	return n, nil
}

func (fd *udtFD) Write(buf []byte) (n int, err error) {
	if err = fd.lockAndIncref(); err != nil {
		return 0, err
	}
	fd.fdmuW.Lock()
	fd.unlock()

	defer func() {
		fd.fdmuW.Unlock()
		fd.lock()
		fd.unlockAndDecref()
	}()

	var nn int
	for nn, n = 0, 0; n < len(buf); n += nn {

		// if getSocketStatus(fd.sock).inTeardown() {
		// 	return n, errClosing
		// }

		nn = int(C.udt_send(fd.sock, slice2cbuf(buf[n:]), C.int(len(buf[n:])), 0))
		if C.int(nn) == C.ERROR {
			return n, fd.lastErrorOp("write")
		}
	}

	return n, nil
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
