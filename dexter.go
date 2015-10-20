// Package dexter provides a thin wrapper around sync.WaitGroup and some
// convenience methods for tracking SIGINT and SIGTERM along with an Alive()
// method that returns state of application to the caller.
//
// Usage example:
//
//   package main
//
//   import "os"
//   import "github.com/ceocoder/dexter"
//
//   func foo(dex *dexter.Dexter, in <-chan string) {
//		 for dex.Alive() {
//			_ := <- in
//		 }
//   }
//
//   func main() {
//		 dex := NewDexter()
//
//		 in := make(chan string)
//		 go foo(&dex, in)
//
//		 f, err := os.Open("file.go")
//		 dex.TrackChannelsToKill(in)
//		 dex.TrackToKill(f)
//
//		 dex.WaitAndKill()
//   }
//
package dexter

import (
	"errors"
	"io"
	"log"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"
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
	wg        sync.WaitGroup
	waiter    chan os.Signal
	channels  []interface{}
	monitored []io.Closer
	keepAlive bool
}

// NewDexter returns a Dexter value.  One typically needs only single
// copy per app.  By default it listens for SIGINT and SIGTERM.
// When it receives either one - it will try to close all the io.Closer()s and
// channels it is currently monitoring.
func NewDexter() *Dexter {
	dex := &Dexter{
		waiter:    make(chan os.Signal),
		keepAlive: true,
		monitored: []io.Closer{},
	}
	signal.Notify(dex.waiter, syscall.SIGINT, syscall.SIGTERM)
	return dex
}

// Add is a really thin wrapper around sync.WorkGroup.Add
func (d *Dexter) Add(delta int) {
	d.wg.Add(delta)
}

// Done is a really thin wrapper around sync.WorkGroup.Done
func (d *Dexter) Done() {
	d.wg.Done()
}

// Alive returns value of keepAlive - it will be set to false once
// SIGINT or SIGTERM is received. Can be used as exit condition for a loop
//
//		dex := NewDexter()
//		for dex.Alive() {
//			...
//		}
func (d *Dexter) Alive() bool {
	return d.keepAlive
}

// TrackToKill keeps list of io.Closers to stop when we receive the shutdown signal
func (d *Dexter) TrackToKill(closable io.Closer) {
	d.monitored = append(d.monitored, closable)
}

// TrackChannelsToKill keeps a list of channels to be closed upon receiving
// SIGINT or SIGTERM
// Since there is no way to pass a chan interface{} for any channel type
// We are using *just* interface as the type of arg here.
// If passed value is NOT of type chan - an error will be returned.
func (d *Dexter) TrackChannelsToKill(channel interface{}) error {
	if reflect.TypeOf(channel).Kind() == reflect.Chan {
		d.channels = append(d.channels, channel)
		return nil
	}
	return errors.New("channel is not of type chan")
}

// WaitAndKill for SIGINT or SIGTERM upon intercepting either one
// * Set Run() value to false.
// * Close all closeable interfaces
// * Close all monitored channels
func (d *Dexter) WaitAndKill() {
	dlog.Println("Started Dexter - waiting for SIGINT or SIGTERM")
	dlog.Printf("Received %v signal, shutting down\n", <-d.waiter)
	dlog.Printf("Closing %d closers\n", len(d.monitored))
	for _, val := range d.monitored {
		val.Close()
	}

	dlog.Printf("Closing %d channels\n", len(d.channels))
	for _, channel := range d.channels {
		reflect.ValueOf(channel).Close()
	}

	// stop loops
	d.keepAlive = false
	dlog.Println("Waiting for go routines to die")
	d.wg.Wait()
}
