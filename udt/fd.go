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
var udtLock sync.Mutex

// udtFD (wraps udt.socket)
type udtFD struct {
	fdmu   sync.Mutex
	refcnt int32

	// immutable until Close
	sock        C.UDTSOCKET
	isClosing   bool
	isConnected bool
	net         string
	laddr       *UDTAddr
	raddr       *UDTAddr
}

// lastError returns the last error as a Go string.
// caller should be holding udtLock, or errors may be reported
// incorrectly...
func lastError() error {
	return errors.New(C.GoString(C.udt_getlasterror_desc()))
}

func newFD(sock C.UDTSOCKET, laddr *UDTAddr, net string) *udtFD {
	fd := &udtFD{sock: sock, laddr: laddr, net: net}
	runtime.SetFinalizer(fd, (*udtFD).Close)
	return fd
}

func (fd *udtFD) init() error {
	return nil
}

func (fd *udtFD) setAddr(laddr, raddr *UDTAddr) {
	fd.laddr = laddr
	fd.raddr = raddr
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

func socket(laddr *UDTAddr) (sock C.UDTSOCKET, reterr error) {
	af, sa, salen, err := laddr.socketArgs()
	if err != nil {
		return 0, err
	}

	// lock + teardown.
	udtLock.Lock()
	defer func() {
		if reterr != nil && sock != C.INVALID_SOCK {
			closeSocket(sock, true)
			sock = C.INVALID_SOCK
		}
		udtLock.Unlock()
	}()

	sock = C.udt_socket(C.int(af), C.SOCK_STREAM, 0)
	if sock == C.INVALID_SOCK {
		reterr = fmt.Errorf("invalid socket: %s", lastError())
		return
	}

	// set options
	trueint := C.int(1)
	falseint := C.int(0)

	// reduce maximum size
	if C.udt_setsockopt(sock, 0, C.UDP_RCVBUF, unsafe.Pointer(&UDP_RCVBUF_SIZE), C.sizeof_int) != 0 {
		reterr = fmt.Errorf("failed to set UDP_RCVBUF: %d, %s", UDP_RCVBUF_SIZE, lastError())
		return
	}

	// set sending to be async
	if C.udt_setsockopt(sock, 0, C.UDT_SNDSYN, unsafe.Pointer(&falseint), C.sizeof_int) != 0 {
		reterr = fmt.Errorf("failed to set UDT_SNDSYN: %s", lastError())
		return
	}

	// set receiving to be async
	if C.udt_setsockopt(sock, 0, C.UDT_RCVSYN, unsafe.Pointer(&falseint), C.sizeof_int) != 0 {
		reterr = fmt.Errorf("failed to set UDT_RCVSYN: %s", lastError())
		return
	}

	// set SO_REUSEADDR
	if C.udt_setsockopt(sock, 0, C.SO_REUSEADDR, unsafe.Pointer(&trueint), C.sizeof_int) != 0 {
		reterr = fmt.Errorf("failed to set SO_REUSEADDR: %s", lastError())
	}

	// cast sockaddr
	csa := (*C.struct_sockaddr)(unsafe.Pointer(sa))
	if C.udt_bind(sock, csa, C.int(salen)) != 0 {
		reterr = fmt.Errorf("failed to bind: %s, %s", laddr, lastError())
		return
	}

	return sock, nil
}

func (fd *udtFD) listen(backlog int) error {
	udtLock.Lock()
	defer udtLock.Unlock()

	if C.udt_listen(fd.sock, C.int(backlog)) == C.ERROR {
		return fmt.Errorf("error listening: %s", lastError())
	}
	return nil
}

// func (fd *udtFD) accept() (remotefd *udtFD, err error) {
// 	if err := fd.lockAndIncref(); err != nil {
// 		return nil, err
// 	}
// 	defer fd.unlockAndDecref()

// 	var s int
// 	var rsa syscall.Sockaddr

// 	for {
// 		s, rsa, err = accept(fd.sysfd)
// 		if err != nil {
// 			if err == syscall.EAGAIN {
// 				if err = fd.pd.WaitRead(); err == nil {
// 					continue
// 				}
// 			} else if err == syscall.ECONNABORTED {
// 				// This means that a socket on the listen queue was closed
// 				// before we Accept()ed it; it's a silly error, so try again.
// 				continue
// 			}
// 			return nil, &net.OpError{"accept", fd.net, fd.laddr, err}
// 		}
// 		break
// 	}

// 	if netfd, err = newFD(s, fd.family, fd.sotype, fd.net); err != nil {
// 		closesocket(s)
// 		return nil, err
// 	}
// 	if err = netfd.init(); err != nil {
// 		fd.Close()
// 		return nil, err
// 	}
// 	lsa, _ := syscall.Getsockname(netfd.sysfd)
// 	netfd.setAddr(toAddr(lsa), toAddr(rsa))
// 	return netfd, nil
// }

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
	udtLock.Unlock()

	for {
		// TODO: replace this with proper net waiting on a Write.
		// polling (EEEEW).
		<-time.After(time.Microsecond * 10)

		nerrlen := C.int(C.sizeof_int)
		nerr := C.int(0)

		udtLock.Lock()
		if C.udt_getsockopt(fd.sock, syscall.SOL_SOCKET, syscall.SO_ERROR, unsafe.Pointer(&nerr), &nerrlen) == C.ERROR {
			err := lastError()
			udtLock.Unlock()
			return err
		}
		udtLock.Unlock()

		switch err := syscall.Errno(nerr); err {
		case syscall.EINPROGRESS, syscall.EALREADY, syscall.EINTR:
		case syscall.Errno(0), syscall.EISCONN:
			return nil
		default:
			return err
		}
	}
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

func (fd *udtFD) Read(p []byte) (n int, err error) {
	panic("not yet implemented")
	// 	if err := fd.readLock(); err != nil {
	// 		return 0, err
	// 	}
	// 	defer fd.readUnlock()
	// 	if err := fd.pd.PrepareRead(); err != nil {
	// 		return 0, &net.OpError{"read", fd.net, fd.raddr, err}
	// 	}
	// 	for {
	// 		n, err = syscall.Read(int(fd.sysfd), p)
	// 		if err != nil {
	// 			n = 0
	// 			if err == syscall.EAGAIN {
	// 				if err = fd.pd.WaitRead(); err == nil {
	// 					continue
	// 				}
	// 			}
	// 		}
	// 		err = chkReadErr(n, err, fd)
	// 		break
	// 	}
	// 	if err != nil && err != io.EOF {
	// 		err = &net.OpError{"read", fd.net, fd.raddr, err}
	// 	}
	// 	return
}

// func (fd *udtFD) readFrom(p []byte) (n int, sa syscall.Sockaddr, err error) {
// 	if err := fd.readLock(); err != nil {
// 		return 0, nil, err
// 	}
// 	defer fd.readUnlock()
// 	if err := fd.pd.PrepareRead(); err != nil {
// 		return 0, nil, &net.OpError{"read", fd.net, fd.laddr, err}
// 	}
// 	for {
// 		n, sa, err = syscall.Recvfrom(fd.sysfd, p, 0)
// 		if err != nil {
// 			n = 0
// 			if err == syscall.EAGAIN {
// 				if err = fd.pd.WaitRead(); err == nil {
// 					continue
// 				}
// 			}
// 		}
// 		err = chkReadErr(n, err, fd)
// 		break
// 	}
// 	if err != nil && err != io.EOF {
// 		err = &net.OpError{"read", fd.net, fd.laddr, err}
// 	}
// 	return
// }

// func (fd *udtFD) readMsg(p []byte, oob []byte) (n, oobn, flags int, sa syscall.Sockaddr, err error) {
// 	if err := fd.readLock(); err != nil {
// 		return 0, 0, 0, nil, err
// 	}
// 	defer fd.readUnlock()
// 	if err := fd.pd.PrepareRead(); err != nil {
// 		return 0, 0, 0, nil, &net.OpError{"read", fd.net, fd.laddr, err}
// 	}
// 	for {
// 		n, oobn, flags, sa, err = syscall.Recvmsg(fd.sysfd, p, oob, 0)
// 		if err != nil {
// 			// TODO(dfc) should n and oobn be set to 0
// 			if err == syscall.EAGAIN {
// 				if err = fd.pd.WaitRead(); err == nil {
// 					continue
// 				}
// 			}
// 		}
// 		err = chkReadErr(n, err, fd)
// 		break
// 	}
// 	if err != nil && err != io.EOF {
// 		err = &net.OpError{"read", fd.net, fd.laddr, err}
// 	}
// 	return
// }

// func chkReadErr(n int, err error, fd *udtFD) error {
// 	if n == 0 && err == nil && fd.sotype != syscall.SOCK_DGRAM && fd.sotype != syscall.SOCK_RAW {
// 		return io.EOF
// 	}
// 	return err
// }

func (fd *udtFD) Write(p []byte) (nn int, err error) {
	panic("not yet implemented")
	// 	if err := fd.writeLock(); err != nil {
	// 		return 0, err
	// 	}
	// 	defer fd.writeUnlock()
	// 	if err := fd.pd.PrepareWrite(); err != nil {
	// 		return 0, &net.OpError{"write", fd.net, fd.raddr, err}
	// 	}
	// 	for {
	// 		var n int
	// 		n, err = syscall.Write(int(fd.sysfd), p[nn:])
	// 		if n > 0 {
	// 			nn += n
	// 		}
	// 		if nn == len(p) {
	// 			break
	// 		}
	// 		if err == syscall.EAGAIN {
	// 			if err = fd.pd.WaitWrite(); err == nil {
	// 				continue
	// 			}
	// 		}
	// 		if err != nil {
	// 			n = 0
	// 			break
	// 		}
	// 		if n == 0 {
	// 			err = io.ErrUnexpectedEOF
	// 			break
	// 		}
	// 	}
	// 	if err != nil {
	// 		err = &net.OpError{"write", fd.net, fd.raddr, err}
	// 	}
	// 	return nn, err
}

// func (fd *udtFD) writeTo(p []byte, sa syscall.Sockaddr) (n int, err error) {
// 	if err := fd.writeLock(); err != nil {
// 		return 0, err
// 	}
// 	defer fd.writeUnlock()
// 	if err := fd.pd.PrepareWrite(); err != nil {
// 		return 0, &net.OpError{"write", fd.net, fd.raddr, err}
// 	}
// 	for {
// 		err = syscall.Sendto(fd.sysfd, p, 0, sa)
// 		if err == syscall.EAGAIN {
// 			if err = fd.pd.WaitWrite(); err == nil {
// 				continue
// 			}
// 		}
// 		break
// 	}
// 	if err == nil {
// 		n = len(p)
// 	} else {
// 		err = &net.OpError{"write", fd.net, fd.raddr, err}
// 	}
// 	return
// }

// func (fd *udtFD) writeMsg(p []byte, oob []byte, sa syscall.Sockaddr) (n int, oobn int, err error) {
// 	if err := fd.writeLock(); err != nil {
// 		return 0, 0, err
// 	}
// 	defer fd.writeUnlock()
// 	if err := fd.pd.PrepareWrite(); err != nil {
// 		return 0, 0, &net.OpError{"write", fd.net, fd.raddr, err}
// 	}
// 	for {
// 		n, err = syscall.SendmsgN(fd.sysfd, p, oob, sa, 0)
// 		if err == syscall.EAGAIN {
// 			if err = fd.pd.WaitWrite(); err == nil {
// 				continue
// 			}
// 		}
// 		break
// 	}
// 	if err == nil {
// 		oobn = len(oob)
// 	} else {
// 		err = &net.OpError{"write", fd.net, fd.raddr, err}
// 	}
// 	return
// }

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
