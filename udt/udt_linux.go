package udt

import "syscall"

func maxRcvBufSize() (uint32, error) {
  return syscall.SysctlUint32("net.core.rmem_max")
}
