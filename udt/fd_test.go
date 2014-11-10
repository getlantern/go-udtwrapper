package udt

import (
	"fmt"
	"syscall"
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
	if _, err := socket(syscall.AF_INET); err != nil {
		t.Fatal(err)
	}
}

func TestSocketClose(t *testing.T) {
	s, err := socket(syscall.AF_INET)
	assert(t, nil == err, err)

	if int(s) <= 0 {
		t.Fatal("socket num invalid")
	}

	if err := closeSocket(s, false); err != nil {
		t.Fatal(err)
	}

	if err := closeSocket(s, false); err == nil {
		t.Fatal("closing again did not produce error")
	}
}

func TestUdtFDConstruct(t *testing.T) {
	a, err := ResolveUDTAddr("udt", ":1234")
	assert(t, nil == err, err)
	s, err := socket(a.AF())
	assert(t, nil == err, err)

	if int(s) <= 0 {
		t.Fatal("socket num invalid")
	}

	fd, err := newFD(s, a, nil, "udt")
	if err != nil {
		t.Fatal(err)
	}

	if err := fd.setDefaultOpts(); err != nil {
		t.Fatal(err)
	}

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
	assert(t, nil == err, err)
	s, err := socket(a.AF())
	assert(t, nil == err, err)
	fd, err := newFD(s, a, nil, "udt")
	assert(t, nil == err, err)
	err = fd.setDefaultOpts()
	assert(t, nil == err, err)

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
	la, err := ResolveUDTAddr("udt", ":1235")
	assert(t, nil == err, err)
	s, err := socket(la.AF())
	assert(t, nil == err, err)
	fd, err := newFD(s, la, nil, "udt")
	assert(t, nil == err, err)
	err = fd.setDefaultOpts()
	assert(t, nil == err, err)

	if err := fd.listen(10); err == nil {
		t.Fatal("should fail. must bind first")
	}

	if err := fd.bind(); err != nil {
		t.Fatal(err)
	}

	if err := fd.listen(10); err != nil {
		t.Fatal(err)
	}

	if err := fd.Close(); err != nil {
		t.Fatal(err)
	}

	assert(t, fd.sock == -1, "sock should now be -1", fd.sock)
	assert(t, fd.Close() != nil, "closing twice should be an error")
}

func TestUdtFDAcceptAndConnect(t *testing.T) {
	al, err := ResolveUDTAddr("udt", "127.0.0.1:1234")
	assert(t, nil == err, err)
	sl, err := socket(al.AF())
	assert(t, nil == err, err)
	sc, err := socket(al.AF())
	assert(t, nil == err, err)
	fdl, err := newFD(sl, al, nil, "udt")
	assert(t, nil == err, err)
	fdc, err := newFD(sc, nil, nil, "udt")
	assert(t, nil == err, err)
	err = fdl.setDefaultOpts()
	assert(t, nil == err, err)
	err = fdc.setDefaultOpts()
	assert(t, nil == err, err)
	err = fdl.bind()
	assert(t, nil == err, err)
	err = fdl.listen(10)
	assert(t, nil == err, err)

	cerrs := make(chan error, 10)
	go func() {
		err := fdc.connect(al)
		if err != nil {
			cerrs <- err
			return
		}

		if fdc.raddr != al {
			cerrs <- fmt.Errorf("addr should be set (todo change)")
		}

		cerrs <- fdc.Close()

		if err := fdc.connect(al); err == nil {
			cerrs <- fmt.Errorf("should not be able to connect after closing")
		}

		assert(t, fdc.sock == -1, "sock should now be -1", fdc.sock)
		assert(t, fdc.Close() != nil, "closing twice should be an error")
		close(cerrs)
	}()

	connl, err := fdl.accept()
	if err != nil {
		t.Fatal(err)
	}

	if connl.sock <= 0 {
		t.Fatal("sock <= 0", connl.sock)
	}

	if err := fdl.Close(); err != nil {
		t.Fatal(err)
	}

	assert(t, fdl.listen(10) != nil, "should not be able to listen after closing")
	assert(t, fdl.sock == -1, "sock should now be -1", fdl.sock)
	assert(t, fdl.Close() != nil, "closing twice should be an error")

	// drain connector errs
	for err := range cerrs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestUdtFDAcceptAndDialFD(t *testing.T) {
	al, err := ResolveUDTAddr("udt", "127.0.0.1:1334")
	assert(t, nil == err, err)
	sl, err := socket(al.AF())
	assert(t, nil == err, err)
	fdl, err := newFD(sl, al, nil, "udt")
	assert(t, nil == err, err)
	err = fdl.setDefaultOpts()
	assert(t, nil == err, err)
	err = fdl.bind()
	assert(t, nil == err, err)
	err = fdl.listen(10)
	assert(t, nil == err, err)

	cerrs := make(chan error, 10)
	go func() {
		fdc, err := dialFD(nil, al)
		if err != nil {
			fmt.Printf("failed to dial %s", err)
			cerrs <- err
			return
		}

		if fdc.raddr != al {
			cerrs <- fmt.Errorf("addr should be set (todo change)")
		}

		cerrs <- fdc.Close()

		if err := fdc.connect(al); err == nil {
			cerrs <- fmt.Errorf("should not be able to connect after closing")
		}

		assert(t, fdc.sock == -1, "sock should now be -1", fdc.sock)
		assert(t, fdc.Close() != nil, "closing twice should be an error")
		close(cerrs)
	}()

	connl, err := fdl.accept()
	if err != nil {
		t.Fatal(err)
	}

	if connl.sock <= 0 {
		t.Fatal("sock <= 0", connl.sock)
	}

	if err := fdl.Close(); err != nil {
		t.Fatal(err)
	}

	assert(t, fdl.listen(10) != nil, "should not be able to listen after closing")
	assert(t, fdl.sock == -1, "sock should now be -1", fdl.sock)
	assert(t, fdl.Close() != nil, "closing twice should be an error")

	// drain connector errs
	for err := range cerrs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestUdtDialFDAndListenFD(t *testing.T) {
	al, err := ResolveUDTAddr("udt", "127.0.0.1:1434")
	assert(t, nil == err, err)

	cerrs := make(chan error, 10)
	go func() {
		fdc, err := dialFD(nil, al)
		if err != nil {
			cerrs <- err
			return
		}

		if fdc.raddr != al {
			cerrs <- fmt.Errorf("addr should be set (todo change)")
		}

		cerrs <- fdc.Close()

		if err := fdc.connect(al); err == nil {
			cerrs <- fmt.Errorf("should not be able to connect after closing")
		}

		assert(t, fdc.sock == -1, "sock should now be -1", fdc.sock)
		assert(t, fdc.Close() != nil, "closing twice should be an error")
		close(cerrs)
	}()

	fdl, err := listenFD(al)
	if err != nil {
		t.Fatal(err)
	}

	connl, err := fdl.accept()
	if err != nil {
		t.Fatal(err)
	}

	if connl.sock <= 0 {
		t.Fatal("sock <= 0", connl.sock)
	}

	if err := fdl.Close(); err != nil {
		t.Fatal(err)
	}

	assert(t, fdl.listen(10) != nil, "should not be able to listen after closing")
	assert(t, fdl.sock == -1, "sock should now be -1", fdl.sock)
	assert(t, fdl.Close() != nil, "closing twice should be an error")

	// drain connector errs
	for err := range cerrs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func assert(t *testing.T, cond bool, vals ...interface{}) {
	if !cond {
		t.Fatal(vals...)
	}
}
