package dexter

import (
	"errors"
	"io"
	"reflect"
	"sync"
)

// Target hold a wait group, channels and io.Closers
// each target should hold resources that need to be shutdown
// stopped at once as in stage before moving on to next logical
// group of targets
type Target struct {
	name      string
	wg        sync.WaitGroup
	channels  []interface{}
	monitored []io.Closer
}

// NewTarget builds a new target to be tracked and killed by dexter
func NewTarget(name string) *Target {
	target := &Target{
		name:      name,
		monitored: []io.Closer{},
	}

	return target
}

// TrackCloser keeps list of io.Closers to stop when we receive the shutdown signal
func (t *Target) TrackCloser(closer io.Closer) {
	t.monitored = append(t.monitored, closer)
}

// TrackChannel keeps a list of channels to be closed upon receiving
// SIGINT or SIGTERM
// Since there is no way to pass a chan interface{} for any channel type
// We are using *just* interface as the type of arg here.
// If passed value is NOT of type chan - an error will be returned.
func (t *Target) TrackChannel(channel interface{}) error {
	if reflect.TypeOf(channel).Kind() == reflect.Chan {
		t.channels = append(t.channels, channel)
		return nil
	}
	return errors.New("channel is not of type chan")
}

// Add is a really thin wrapper around sync.WorkGroup.Add
func (t *Target) Add(delta int) {
	t.wg.Add(delta)
}

// Done is a really thin wrapper around sync.WorkGroup.Done
func (t *Target) Done() {
	t.wg.Done()
}

// Wait is a really thin wrapper around sync.WorGroup.Wait
func (t *Target) Wait() {
	t.wg.Wait()
}

func (t *Target) kill() {
	dlog.Printf("Killing target %s\n", t.name)
	for _, val := range t.monitored {
		val.Close()
	}

	dlog.Printf("Closing %d channels\n", len(t.channels))
	for _, channel := range t.channels {
		reflect.ValueOf(channel).Close()
	}
}
