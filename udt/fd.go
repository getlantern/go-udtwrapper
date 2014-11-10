package udt

import (
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync"
	"syscall"
	"time"
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

// returned when calling functions on a fd that's closing.
var errClosing = errors.New("file descriptor closing")

// UDP_RCVBUF_SIZE is the default UDP_RCVBUF size.
var UDP_RCVBUF_SIZE = uint32(20971520) // 20MB

func init() {
	// adjust the rcvbuf to our max.
	max, err := maxRcvBufSize()
	if err == nil {
		max = uint32(float32(max) * 0.5) // 0.5 of max.
		if max < UDP_RCVBUF_SIZE {
			UDP_RCVBUF_SIZE = max
		}
	}
}

// udtLock is a lock on the entire udt API. WHAT!? you might say,
// and you'd be right to scream. The udt API is not re-entrant,
// and in particular it _sets a global error and has the user
// fetch it with a function!!! (errno style. wtf)_
//
// Since we're probably paying the lovely cost of a syscall on
// such calls this isn't sooo bad. But it's still bad.
var udtLock polyamorous

type polyamorous struct {
	sync.Locker
}

func (*polyamorous) Lock()   {}
func (*polyamorous) Unlock() {}

// udtFD (wraps udt.socket)
type udtFD struct {
	fdmu   sync.Mutex
	fdmuR  sync.Mutex
	fdmuW  sync.Mutex
	refcnt int32
	bound  bool

	// immutable until Close
	sock        C.UDTSOCKET
	isClosing   bool
	isConnected bool
	net         string
	laddr       *UDTAddr
	raddr       *UDTAddr
}

func newFD(sock C.UDTSOCKET, laddr, raddr *UDTAddr, net string) (*udtFD, error) {
	lac := laddr.copy()
	rac := raddr.copy()
	fd := &udtFD{sock: sock, laddr: lac, raddr: rac, net: net}
	runtime.SetFinalizer(fd, (*udtFD).Close)
	return fd, nil
}

// lastErrorOp returns the last error as a net.OpError.
// caller should be holding udtLock, or errors may be reported
// incorrectly...
func (fd *udtFD) lastErrorOp(op string) *net.OpError {
	return &net.OpError{op, fd.net, fd.laddr, lastError()}
}

func (fd *udtFD) name() string {
	var ls, rs string
	if fd.laddr != nil {
		ls = fd.laddr.String()
	}
	if fd.raddr != nil {
		rs = fd.raddr.String()
	}
	return fd.net + ":" + ls + "->" + rs
}

func (fd *udtFD) setDefaultOpts() error {

	// lock + teardown.
	udtLock.Lock()
	defer udtLock.Unlock()

	// options
	trueint := C.int(1)
	falseint := C.int(0)

	// reduce maximum size
	if C.udt_setsockopt(fd.sock, 0, C.UDP_RCVBUF, unsafe.Pointer(&UDP_RCVBUF_SIZE), C.sizeof_int) != 0 {
		return fmt.Errorf("failed to set UDP_RCVBUF: %d, %s", UDP_RCVBUF_SIZE, lastError())
	}

	// set UDT_REUSEADDR
	if C.udt_setsockopt(fd.sock, 0, C.UDT_REUSEADDR, unsafe.Pointer(&trueint), C.sizeof_int) != 0 {
		return fmt.Errorf("failed to set UDT_REUSEADDR: %s", lastError())
	}

	// set UDT_LINGER
	if C.udt_setsockopt(fd.sock, 0, C.UDT_LINGER, unsafe.Pointer(&falseint), C.sizeof_int) != 0 {
		return fmt.Errorf("failed to set UDT_LINGER: %s", lastError())
	}

	return nil
}

func (fd *udtFD) setAsyncOpts() error {
	return fmt.Errorf("async disabled")

	// lock + teardown.
	udtLock.Lock()
	defer udtLock.Unlock()

	// options
	falseint := C.int(0)

	// set sending to be async
	if C.udt_setsockopt(fd.sock, 0, C.UDT_SNDSYN, unsafe.Pointer(&falseint), C.sizeof_int) != 0 {
		return fmt.Errorf("failed to set UDT_SNDSYN: %s", lastError())
	}

	// set receiving to be async
	if C.udt_setsockopt(fd.sock, 0, C.UDT_RCVSYN, unsafe.Pointer(&falseint), C.sizeof_int) != 0 {
		return fmt.Errorf("failed to set UDT_RCVSYN: %s", lastError())
	}

	return nil
}

func (fd *udtFD) bind() error {
	_, sa, salen, err := fd.laddr.socketArgs()
	if err != nil {
		return err
	}

	udtLock.Lock()
	defer udtLock.Unlock()

	// cast sockaddr
	csa := (*C.struct_sockaddr)(unsafe.Pointer(sa))
	if C.udt_bind(fd.sock, csa, C.int(salen)) != 0 {
		return fd.lastErrorOp("bind")
	}

	return nil
}

func (fd *udtFD) listen(backlog int) error {

	udtLock.Lock()
	defer udtLock.Unlock()

	if C.udt_listen(fd.sock, C.int(backlog)) == C.ERROR {
		return fd.lastErrorOp("listen")
	}
	return nil
}

func (fd *udtFD) accept() (*udtFD, error) {
	if err := fd.lockAndIncref(); err != nil {
		return nil, err
	}
	defer fd.unlockAndDecref()

	var sa syscall.RawSockaddrAny
	var salen C.int

	udtLock.Lock()
	sock2 := C.udt_accept(fd.sock, (*C.struct_sockaddr)(unsafe.Pointer(&sa)), &salen)
	if sock2 == C.INVALID_SOCK {
		err := fd.lastErrorOp("accept")
		udtLock.Unlock()
		return nil, err
	}
	udtLock.Unlock()

	raddr, err := addrWithSockaddr(&sa)
	if err != nil {
		closeSocket(sock2, false)
		return nil, fmt.Errorf("error converting address: %s", err)
	}

	remotefd, err := newFD(sock2, fd.laddr, raddr, fd.net)
	if err != nil {
		closeSocket(sock2, false)
		return nil, err
	}

	// if err = remotefd.setAsyncOpts(); err != nil {
	// 	remotefd.Close()
	// 	return nil, err
	// }

	return remotefd, nil
}

func (fd *udtFD) connect(raddr *UDTAddr) error {

	_, sa, salen, err := raddr.socketArgs()
	if err != nil {
		return err
	}
	csa := (*C.struct_sockaddr)(unsafe.Pointer(sa))

	udtLock.Lock()
	if C.udt_connect(fd.sock, csa, C.int(salen)) == C.ERROR {
		err := lastError()
		udtLock.Unlock()
		return fmt.Errorf("error connecting: %s", err)
	}

	fd.raddr = raddr
	udtLock.Unlock()
	// return fd.setAsyncOpts()
	return nil

	// for {
	// 	// TODO: replace this with proper net waiting on a Write.
	// 	// polling (EEEEW).
	// 	<-time.After(time.Microsecond * 10)

	// 	nerrlen := C.int(C.sizeof_int)
	// 	nerr := C.int(0)

	// 	udtLock.Lock()
	// 	if C.udt_getsockopt(fd.sock, syscall.SOL_SOCKET, syscall.SO_ERROR, unsafe.Pointer(&nerr), &nerrlen) == C.ERROR {
	// 		err := lastError()
	// 		udtLock.Unlock()
	// 		return err
	// 	}
	// 	udtLock.Unlock()

	// 	switch err := syscall.Errno(nerr); err {
	// 	case syscall.EINPROGRESS, syscall.EALREADY, syscall.EINTR:
	// 	case syscall.Errno(0), syscall.EISCONN:
	// 		return nil
	// 	default:
	// 		return err
	// 	}
	// }
}

func (fd *udtFD) destroy() {
	closeSocket(fd.sock, false)
	fd.sock = -1
	runtime.SetFinalizer(fd, nil)
}

// Add a reference to this fd.
// Returns an error if the fd cannot be used.
func (fd *udtFD) incref() {
	fd.refcnt++
}

// Remove a reference to this FD and close if we've been asked to do so
// (and there are no references left).
func (fd *udtFD) decref() {
	fd.refcnt--
	if fd.isClosing && fd.refcnt == 0 {
		fd.destroy()
	}
}

// Lock
// Returns an error if the fd cannot be used.
func (fd *udtFD) lock() error {
	fd.fdmu.Lock()
	if fd.isClosing {
		fd.fdmu.Unlock()
		return errClosing
	}
	return nil
}

// Unlock
func (fd *udtFD) unlock() {
	fd.fdmu.Unlock()
}

// Locks, and adds a reference to this fd
// Returns an error if the fd cannot be used.
func (fd *udtFD) lockAndIncref() error {
	if err := fd.lock(); err != nil {
		return err
	}
	fd.incref()
	return nil
}

// Removes a reference and unlocks
func (fd *udtFD) unlockAndDecref() {
	fd.decref()
	fd.unlock()
}

func (fd *udtFD) Close() error {
	if err := fd.lockAndIncref(); err != nil {
		return err
	}

	// Unblock any I/O.  Once it all unblocks and returns,
	// so that it cannot be referring to fd.sysfd anymore,
	// the final decref will close fd.sysfd.  This should happen
	// fairly quickly, since all the I/O is non-blocking, and any
	// attempts to block in the pollDesc will return errClosing.

	// TODO
	fd.isClosing = true
	fd.unlockAndDecref()
	return nil
}

// func (fd *udtFD) shutdown(how int) error {
// 	if err := fd.incref(); err != nil {
// 		return err
// 	}
// 	defer fd.decref()

// 	if err := fd.closeSocket(); err != nil {
// 		return &net.OpError{"shutdown", fd.net, fd.laddr, err}
// 	}
// 	return nil
// }

// net.Conn functions

func (fd *udtFD) LocalAddr() net.Addr {
	return fd.laddr
}

func (fd *udtFD) RemoteAddr() net.Addr {
	return fd.raddr
}

func (fd *udtFD) SetDeadline(t time.Time) error {
	panic("not yet implemented")
}

func (fd *udtFD) SetReadDeadline(t time.Time) error {
	panic("not yet implemented")
}

func (fd *udtFD) SetWriteDeadline(t time.Time) error {
	panic("not yet implemented")
}

// lastError returns the last error as a Go string.
// caller should be holding udtLock, or errors may be reported
// incorrectly...
func lastError() error {
	return errors.New(C.GoString(C.udt_getlasterror_desc()))
}

func socket(addrfamily int) (sock C.UDTSOCKET, reterr error) {

	// lock + teardown.
	udtLock.Lock()
	defer udtLock.Unlock()

	sock = C.udt_socket(C.int(addrfamily), C.SOCK_STREAM, 0)
	if sock == C.INVALID_SOCK {
		return C.INVALID_SOCK, fmt.Errorf("invalid socket: %s", lastError())
	}

	return sock, nil
}

func closeSocket(sock C.UDTSOCKET, locked bool) error {
	if !locked {
		udtLock.Lock()
		defer udtLock.Unlock()
	}

	if C.udt_close(sock) == C.ERROR {
		return lastError()
	}
	return nil
}

// dialFD sets up a udtFD
func dialFD(laddr, raddr *UDTAddr) (*udtFD, error) {

	if raddr == nil {
		return nil, &net.OpError{"dial", "udt", raddr, errors.New("invalid remote address")}
	}

	if laddr != nil && laddr.AF() != raddr.AF() {
		return nil, &net.OpError{"dial", "udt", raddr, errors.New("differing remote address network")}
	}

	sock, err := socket(raddr.AF())
	if err != nil {
		return nil, err
	}

	fd, err := newFD(sock, laddr, raddr, "udt")
	if err != nil {
		closeSocket(sock, false)
		return nil, err
	}

	if err := fd.setDefaultOpts(); err != nil {
		fd.Close()
		return nil, err
	}

	if laddr != nil {
		if err := fd.bind(); err != nil {
			fd.Close()
			return nil, err
		}
	}

	if err := fd.connect(raddr); err != nil {
		fd.Close()
		return nil, err
	}

	return fd, nil
}

// listenFD sets up a udtFD
func listenFD(laddr *UDTAddr) (*udtFD, error) {

	if laddr == nil {
		return nil, &net.OpError{"dial", "udt", nil, errors.New("invalid address")}
	}

	sock, err := socket(laddr.AF())
	if err != nil {
		return nil, err
	}

	fd, err := newFD(sock, laddr, nil, "udt")
	if err != nil {
		closeSocket(sock, false)
		return nil, err
	}

	if err := fd.setDefaultOpts(); err != nil {
		fd.Close()
		return nil, err
	}

	if err := fd.bind(); err != nil {
		fd.Close()
		return nil, err
	}

	// TODO: use maxListenerBacklog like golang.org/net/
	if err := fd.listen(syscall.SOMAXCONN); err != nil {
		fd.Close()
		return nil, err
	}

	return fd, nil
}
