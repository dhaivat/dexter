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

	stage1 := NewTarget("stage1")
	f1 := func(target *Target, in <-chan string) {
		target.Add(1)
		defer target.Done()
		for _ = range in {
		}
	}
	c1In := make(chan string)
	stage1.TrackChannel(c1In)

	stage2 := NewTarget("stage2")
	f2 := func(target *Target, in <-chan int) {
		target.Add(1)
		defer target.Done()
		for _ = range in {
			_ = <-in
		}
	}
	c2In := make(chan int)
	stage2.TrackChannel(c2In)

	stage3 := NewTarget("stage3")
	f3 := func(target *Target, in <-chan bool) {
		target.Add(1)
		defer target.Done()
		for _ = range in {
			<-in
		}
	}
	c3In := make(chan bool)
	stage3.TrackChannel(c3In)
	stage3.TrackCloser(dcloser{})

	go f1(stage1, c1In)
	go f2(stage2, c2In)
	go f3(stage3, c3In)

	dex := NewDexter()
	dex.Track(stage1)
	dex.Track(stage2)
	dex.Track(stage3)

	go func() {
		// kill after it waiting for small amount of time
		time.Sleep(10 * time.Millisecond)
		syscall.Kill(os.Getpid(), syscall.SIGINT)
	}()
	dex.WaitAndKill()
}
