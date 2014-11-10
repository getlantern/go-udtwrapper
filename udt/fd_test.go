package udt

import (
	"testing"
)

func TestResolevUDTAddr(t *testing.T) {
	a, err := ResolveUDTAddr("udt", ":1234")
	if err != nil {
		t.Fatal(err)
	}

	if a.Network() != "udt" {
		t.Fatal("addr resolved incorrectly: %s", a.Network())
	}

	if a.String() != ":1234" {
		t.Fatal("addr resolved incorrectly: %s", a)
	}
}

func TestSocketConstruct(t *testing.T) {
	a, err := ResolveUDTAddr("udt", ":1234")
	assert(t, nil == err)

	if _, err := socket(a); err != nil {
		t.Fatal(err)
	}
}

func TestSocketClose(t *testing.T) {
	a, err := ResolveUDTAddr("udt", ":1234")
	assert(t, nil == err)
	s, err := socket(a)
	assert(t, nil == err)

	if int(s) <= 0 {
		t.Fatal("socket num invalid")
	}

	if err := closeSocket(s); err != nil {
		t.Fatal(err)
	}

	if err := closeSocket(s); err == nil {
		t.Fatal("closing again did not produce error")
	}
}

func TestUdtFDConstruct(t *testing.T) {
	a, err := ResolveUDTAddr("udt", ":1234")
	assert(t, nil == err)
	s, err := socket(a)
	assert(t, nil == err)

	if int(s) <= 0 {
		t.Fatal("socket num invalid")
	}

	fd := newFD(s, a, "udt")

	if fd.name() != "udt::1234->" {
		t.Fatal("incorrect name:", fd.name())
	}

	if err := fd.Close(); err != nil {
		t.Fatal(err)
	}

	if int(fd.sock) != -1 {
		t.Fatal("sock should now be -1")
	}

	if err := fd.Close(); err == nil {
		t.Fatal("closing twice should be an error")
	}
}

func TestUdtFDLocking(t *testing.T) {
	a, err := ResolveUDTAddr("udt", ":1234")
	assert(t, nil == err)
	s, err := socket(a)
	assert(t, nil == err)
	fd := newFD(s, a, "udt")

	if err := fd.lockAndIncref(); err != nil {
		t.Fatal(err)
	}

	if fd.refcnt != 1 {
		t.Fatal("fd.refcnt != 1", fd.refcnt)
	}

	fd.unlockAndDecref()

	if fd.refcnt != 0 {
		t.Fatal("fd.refcnt != 0", fd.refcnt)
	}

	fd.incref()
	fd.incref()
	fd.incref()

	if fd.refcnt != 3 {
		t.Fatal("fd.refcnt != 3", fd.refcnt)
	}

	if err := fd.Close(); err != nil {
		t.Fatal(err)
	}

	if int(fd.sock) == -1 {
		t.Fatal("sock should not yet be -1")
	}

	fd.decref()
	fd.decref()

	if fd.refcnt != 1 {
		t.Fatal("fd.refcnt != 1", fd.refcnt)
	}

	if err := fd.Close(); err == nil {
		t.Fatal("closing twice should still be an error")
	}

	fd.decref()

	if fd.refcnt != 0 {
		t.Fatal("fd.refcnt != 0", fd.refcnt)
	}

	if int(fd.sock) != -1 {
		t.Fatal("sock should now be -1")
	}
}

func TestUdtFDListenOnly(t *testing.T) {
	la, err := ResolveUDTAddr("udt", ":1234")
	assert(t, nil == err)
	s, err := socket(la)
	assert(t, nil == err)
	fd := newFD(s, la, "udt")

	if err := fd.listen(10); err != nil {
		t.Fatal(err)
	}

	if err := fd.Close(); err != nil {
		t.Fatal(err)
	}

	if err := fd.listen(10); err == nil {
		t.Fatal("should not be able to listen after closing")
	}
}

func TestUdtFDConnect(t *testing.T) {
	la, err := ResolveUDTAddr("udt", ":1234")
	assert(t, nil == err)
	// ra, err := ResolveUDTAddr("udt", "127.0.0.1:2222")
	// assert(t, nil == err)
	s, err := socket(la)
	assert(t, nil == err)
	fd := newFD(s, la, "udt")

	// if err := fd.connect(ra); err != nil {
	// 	t.Fatal(err)
	// }

	if err := fd.Close(); err != nil {
		t.Fatal(err)
	}
}

func assert(t *testing.T, cond bool, vals ...interface{}) {
	if !cond {
		t.Fatal(vals...)
	}
}
