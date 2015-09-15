// package udtcat provides an implementation of netcat using the go-udtwrapper package.
// It is meant to exercise the udt implementation.
// Usage:
//    ucat [<local address>] <remote address>
//    ucat -l <local address>
//
// Address format is: [host]:port
//
// Note that UDT's congestion control gives priority to tcp flows (web traffic),
// so you could use this ucat tool to transfer massive files without hogging
// all the bandwidth.
package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"

	udt "github.com/jbenet/go-udtwrapper/udt"
)

var verbose = false

// Usage prints out the usage of this module.
// Assumes flags use go stdlib flag pacakage.
var Usage = func() {
	text := `udtcat - UDT netcat in Go

Usage:

  listen: %s [<local address>] <remote address>
  dial:   %s -l <local address>

Address format is Go's: [host]:port
`

	fmt.Fprintf(os.Stderr, text, os.Args[0], os.Args[0])
	flag.PrintDefaults()
}

type args struct {
	listen     bool
	verbose    bool
	localAddr  string
	remoteAddr string
}

func parseArgs() args {
	var a args

	// setup + parse flags
	flag.BoolVar(&a.listen, "listen", false, "listen for connections")
	flag.BoolVar(&a.listen, "l", false, "listen for connections (short)")
	flag.BoolVar(&a.verbose, "v", false, "verbose debugging")
	flag.Usage = Usage
	flag.Parse()
	osArgs := flag.Args()

	if len(osArgs) < 1 {
		exit("")
	}

	if a.listen {
		a.localAddr = osArgs[0]
	} else {
		if len(osArgs) > 1 {
			a.localAddr = osArgs[0]
			a.remoteAddr = osArgs[1]
		} else {
			a.remoteAddr = osArgs[0]
		}
	}

	return a
}

func main() {
	args := parseArgs()
	verbose = args.verbose

	go func() {
		// wait until we exit.
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGABRT)
		<-sigc
		panic("ABORT! ABORT! ABORT!")
	}()

	var err error
	if args.listen {
		err = Listen(args.localAddr)
	} else {
		err = Dial(args.localAddr, args.remoteAddr)
	}

	if err != nil {
		exit("%s", err)
	}
}

func exit(format string, vals ...interface{}) {
	if format != "" {
		fmt.Fprintf(os.Stderr, "ucat error: "+format+"\n", vals...)
	}
	Usage()
	os.Exit(1)
}

func log(format string, vals ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stderr, "ucat log: "+format+"\n", vals...)
	}
}

// Listen listens and accepts one incoming UDT connection on a given port,
// and pipes all incoming data to os.Stdout.
func Listen(localAddr string) error {
	l, err := udt.Listen("udt", localAddr)
	if err != nil {
		return err
	}
	log("listening at %s", l.Addr())

	c, err := l.Accept()
	if err != nil {
		return err
	}
	log("accepted connection from %s", c.RemoteAddr())

	// done with listener
	l.Close()

	netcat(c)
	return c.Close()
}

// Dial connects to a remote address and pipes all os.Stdin to the remote end.
// If localAddr is set, uses it to Dial from.
func Dial(localAddr, remoteAddr string) error {

	var laddr net.Addr
	var err error
	if localAddr != "" {
		laddr, err = udt.ResolveUDTAddr("udt", localAddr)
		if err != nil {
			return fmt.Errorf("failed to resolve address %s", localAddr)
		}
	}

	if laddr != nil {
		log("dialing %s from %s", remoteAddr, laddr)
	} else {
		log("dialing %s", remoteAddr)
	}

	d := udt.Dialer{LocalAddr: laddr}
	c, err := d.Dial("udt", remoteAddr)
	if err != nil {
		return err
	}
	log("connected to %s", c.RemoteAddr())

	netcat(c)
	return c.Close()
}

func netcat(c net.Conn) {
	log("piping stdio to connection")

	done := make(chan struct{}, 2)

	go func() {
		n, _ := io.Copy(c, os.Stdin)
		log("sent %d bytes", n)
		done <- struct{}{}
	}()
	go func() {
		n, _ := io.Copy(os.Stdout, c)
		log("received %d bytes", n)
		done <- struct{}{}
	}()

	// wait until we exit.
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT,
		syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-done:
	case <-sigc:
		return
	}

	c.Close()
}
