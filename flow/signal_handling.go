// +build darwin freebsd netbsd openbsd, !plan9, !windows, !linux

package flow

import (
	"os"
	"os/signal"
	"syscall"
)

func OnInterrupt(fn func()) {
	// deal with control+c,etc
	signalChan := make(chan os.Signal, 1)
	// controlling terminal close, daemon not exit
	signal.Ignore(syscall.SIGHUP)
	signal.Notify(signalChan,
		os.Interrupt,
		os.Kill,
		syscall.SIGALRM,
		// syscall.SIGHUP,
		syscall.SIGINFO, // this causes windows to fail
		syscall.SIGINT,
		syscall.SIGTERM,
		// syscall.SIGQUIT, // Quit from keyboard, "kill -3"
	)
	go func() {
		for sig := range signalChan {
			fn()
			if sig != syscall.SIGINFO {
				os.Exit(0)
			}
		}
	}()
}