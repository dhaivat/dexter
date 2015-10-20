package dexter

import (
	"os"
	"syscall"
	"testing"
	"time"
)

type dcloser struct {
}

func (d dcloser) Close() error {
	return nil
}

func TestDexter(t *testing.T) {

	f1 := func(dex *Dexter, in <-chan string) {
		dex.Add(1)
		defer dex.Done()
		for dex.Alive() {
			_, _ = <-in
		}
	}

	f2 := func(dex *Dexter, in <-chan int) {
		dex.Add(1)
		defer dex.Done()
		for dex.Alive() {
			_ = <-in
		}
	}

	f3 := func(dex *Dexter, in <-chan bool) {
		dex.Add(1)
		defer dex.Done()
		for dex.Alive() {
			<-in
		}
	}

	c1In := make(chan string)
	c2In := make(chan int)
	c3In := make(chan bool)

	dex := NewDexter()
	dex.TrackChannelsToKill(c1In)
	dex.TrackChannelsToKill(c2In)
	dex.TrackChannelsToKill(c3In)
	dex.TrackToKill(dcloser{})

	go f1(dex, c1In)
	go f2(dex, c2In)
	go f3(dex, c3In)

	go func() {
		// kill after it waiting for small amount of time
		time.Sleep(1 * time.Second)
		syscall.Kill(os.Getpid(), syscall.SIGINT)
	}()
	dex.WaitAndKill()
}
