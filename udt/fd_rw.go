package udt

import ()

// #cgo CFLAGS: -Wall
// #cgo LDFLAGS: libudt.dylib
//
// #include "udt_c.h"
// #include <errno.h>
// #include <arpa/inet.h>
// #include <string.h>
import "C"

func (fd *udtFD) Read(p []byte) (n int, err error) {
	panic("not yet implemented")
	//  if err := fd.readLock(); err != nil {
	//    return 0, err
	//  }
	//  defer fd.readUnlock()
	//  if err := fd.pd.PrepareRead(); err != nil {
	//    return 0, &net.OpError{"read", fd.net, fd.raddr, err}
	//  }
	//  for {
	//    n, err = syscall.Read(int(fd.sysfd), p)
	//    if err != nil {
	//      n = 0
	//      if err == syscall.EAGAIN {
	//        if err = fd.pd.WaitRead(); err == nil {
	//          continue
	//        }
	//      }
	//    }
	//    err = chkReadErr(n, err, fd)
	//    break
	//  }
	//  if err != nil && err != io.EOF {
	//    err = &net.OpError{"read", fd.net, fd.raddr, err}
	//  }
	//  return
}

// func (fd *udtFD) readFrom(p []byte) (n int, sa syscall.Sockaddr, err error) {
//  if err := fd.readLock(); err != nil {
//    return 0, nil, err
//  }
//  defer fd.readUnlock()
//  if err := fd.pd.PrepareRead(); err != nil {
//    return 0, nil, &net.OpError{"read", fd.net, fd.laddr, err}
//  }
//  for {
//    n, sa, err = syscall.Recvfrom(fd.sysfd, p, 0)
//    if err != nil {
//      n = 0
//      if err == syscall.EAGAIN {
//        if err = fd.pd.WaitRead(); err == nil {
//          continue
//        }
//      }
//    }
//    err = chkReadErr(n, err, fd)
//    break
//  }
//  if err != nil && err != io.EOF {
//    err = &net.OpError{"read", fd.net, fd.laddr, err}
//  }
//  return
// }

// func (fd *udtFD) readMsg(p []byte, oob []byte) (n, oobn, flags int, sa syscall.Sockaddr, err error) {
//  if err := fd.readLock(); err != nil {
//    return 0, 0, 0, nil, err
//  }
//  defer fd.readUnlock()
//  if err := fd.pd.PrepareRead(); err != nil {
//    return 0, 0, 0, nil, &net.OpError{"read", fd.net, fd.laddr, err}
//  }
//  for {
//    n, oobn, flags, sa, err = syscall.Recvmsg(fd.sysfd, p, oob, 0)
//    if err != nil {
//      // TODO(dfc) should n and oobn be set to 0
//      if err == syscall.EAGAIN {
//        if err = fd.pd.WaitRead(); err == nil {
//          continue
//        }
//      }
//    }
//    err = chkReadErr(n, err, fd)
//    break
//  }
//  if err != nil && err != io.EOF {
//    err = &net.OpError{"read", fd.net, fd.laddr, err}
//  }
//  return
// }

// func chkReadErr(n int, err error, fd *udtFD) error {
//  if n == 0 && err == nil && fd.sotype != syscall.SOCK_DGRAM && fd.sotype != syscall.SOCK_RAW {
//    return io.EOF
//  }
//  return err
// }

func (fd *udtFD) Write(p []byte) (nn int, err error) {
	panic("not yet implemented")
	//  if err := fd.writeLock(); err != nil {
	//    return 0, err
	//  }
	//  defer fd.writeUnlock()
	//  if err := fd.pd.PrepareWrite(); err != nil {
	//    return 0, &net.OpError{"write", fd.net, fd.raddr, err}
	//  }
	//  for {
	//    var n int
	//    n, err = syscall.Write(int(fd.sysfd), p[nn:])
	//    if n > 0 {
	//      nn += n
	//    }
	//    if nn == len(p) {
	//      break
	//    }
	//    if err == syscall.EAGAIN {
	//      if err = fd.pd.WaitWrite(); err == nil {
	//        continue
	//      }
	//    }
	//    if err != nil {
	//      n = 0
	//      break
	//    }
	//    if n == 0 {
	//      err = io.ErrUnexpectedEOF
	//      break
	//    }
	//  }
	//  if err != nil {
	//    err = &net.OpError{"write", fd.net, fd.raddr, err}
	//  }
	//  return nn, err
}

// func (fd *udtFD) writeTo(p []byte, sa syscall.Sockaddr) (n int, err error) {
//  if err := fd.writeLock(); err != nil {
//    return 0, err
//  }
//  defer fd.writeUnlock()
//  if err := fd.pd.PrepareWrite(); err != nil {
//    return 0, &net.OpError{"write", fd.net, fd.raddr, err}
//  }
//  for {
//    err = syscall.Sendto(fd.sysfd, p, 0, sa)
//    if err == syscall.EAGAIN {
//      if err = fd.pd.WaitWrite(); err == nil {
//        continue
//      }
//    }
//    break
//  }
//  if err == nil {
//    n = len(p)
//  } else {
//    err = &net.OpError{"write", fd.net, fd.raddr, err}
//  }
//  return
// }

// func (fd *udtFD) writeMsg(p []byte, oob []byte, sa syscall.Sockaddr) (n int, oobn int, err error) {
//  if err := fd.writeLock(); err != nil {
//    return 0, 0, err
//  }
//  defer fd.writeUnlock()
//  if err := fd.pd.PrepareWrite(); err != nil {
//    return 0, 0, &net.OpError{"write", fd.net, fd.raddr, err}
//  }
//  for {
//    n, err = syscall.SendmsgN(fd.sysfd, p, oob, sa, 0)
//    if err == syscall.EAGAIN {
//      if err = fd.pd.WaitWrite(); err == nil {
//        continue
//      }
//    }
//    break
//  }
//  if err == nil {
//    oobn = len(oob)
//  } else {
//    err = &net.OpError{"write", fd.net, fd.raddr, err}
//  }
//  return
// }
