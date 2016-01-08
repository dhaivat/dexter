// Package dexter provides a thin wrapper around sync.WaitGroup and some
// convenience methods for tracking SIGINT and SIGTERM
//
// Each stage of application that needs to shutdown should have a correspondign Target
// this target will be killed in the order it was added to dexter.  This allows shutdown
// in stages.
//
// Usage example:
//
//	package main
//
//	import "os"
//	import "github.com/ceocoder/dexter"
//
//	func foo(dex *dexter.Target, in <-chan string) {
//		 for _ := range in {
//
//		 }
//	}
//
//	func main() {
//		dex := NewDexter()
//
//		foo := NewTarget("foo")
//		in := make(chan string)
//		foo.TrackChannel(in)
//
//		f, err := os.Open("file.go")
//		foo.TrackCloser(f)
//
//		go foo(foo, in)
//
//		bar := NewTarget("bar")
//		out := make(chan int)
//
//		bar.TrackChannel(out)
//
//		dex.Track(foo)
//		dex.Track(bar)
//
//		dex.WaitAndKill()
//   }
//
package dexter

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	dlog *log.Logger
)

// annotate our logs with [Dexter]
func init() {
	dlog = log.New(os.Stdout, "[Dexter] ", log.Ldate|log.Ltime)
}

// Dexter is a wrapper around sync.WaitGroup with convenience methods to intercept
// SIGINT and SIGTERM and provides a way of graceful shutdown
type Dexter struct {
	waiter          chan os.Signal
	targets         []*Target
	forceKillWindow time.Duration
	exitFunc        func(int)
}

// NewDexter returns a Dexter value.  One typically needs only single
// copy per app.  By default it listens for SIGINT and SIGTERM.
// When it receives either one - it will try to close all the io.Closer()s and
// channels it is currently monitoring.
func NewDexter() *Dexter {
	dex := &Dexter{
		waiter:          make(chan os.Signal),
		targets:         []*Target{},
		forceKillWindow: 5 * time.Second,
		exitFunc:        os.Exit,
	}
	signal.Notify(dex.waiter, syscall.SIGINT, syscall.SIGTERM)
	return dex
}

// SetForceKillInterval sets amount of time (in seconds) to wait before exiting with
// non-zero return code, this helps one avoid stuck processes
func (d *Dexter) SetForceKillInterval(interval time.Duration) {
	d.forceKillWindow = interval
}

// Track adds a new target to Dexter's kill list,
// this target will be killed in the order it was inserted in
func (d *Dexter) Track(target *Target) {
	d.targets = append(d.targets, target)
}

// WaitAndKill for SIGINT or SIGTERM upon intercepting either one
// * Close all closeable interfaces
// * Close all monitored channels
func (d *Dexter) WaitAndKill() {
	dlog.Println("Started Dexter - waiting for SIGINT or SIGTERM")
	dlog.Printf("Received %v signal, shutting down\n", <-d.waiter)
	dlog.Printf("Killing %d targets\n", len(d.targets))

	// starting a routine in the background to kill if process doesn't die
	// gracefully in set time
	timer := time.AfterFunc(1*time.Second, func() {
		dlog.Println("Timeout! - force exiting")
		d.exitFunc(1)
	})
	defer timer.Stop()

	for _, target := range d.targets {
		target.kill()
		target.Wait()
	}

	// stop loops
	dlog.Println("Killed all targets returning control")
}
