package dexter

import (
	"os"
	"os/signal"
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
	stage1.Add(1)
	f1 := func(target *Target, in <-chan string) {
		defer target.Done()
		for _ = range in {
		}
	}
	c1In := make(chan string)
	stage1.TrackChannel(c1In)

	stage2 := NewTarget("stage2")
	stage2.Add(1)
	f2 := func(target *Target, in <-chan int) {
		defer target.Done()
		for _ = range in {
			_ = <-in
		}
	}
	c2In := make(chan int)
	stage2.TrackChannel(c2In)

	stage3 := NewTarget("stage3")
	stage3.Add(1)
	f3 := func(target *Target, in <-chan bool) {
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

func TestForceKillInterval(t *testing.T) {

	stage1Stuck := NewTarget("stage_stuck")
	stage1Stuck.Add(1)
	f1 := func(target *Target) {
		for {
			time.Sleep(1 * time.Second)
		}
	}
	go f1(stage1Stuck)

	pass := make(chan bool)
	fail := make(chan bool)
	dummyExitFunc := func(code int) {
		if code == 0 {
			fail <- true
		} else {
			pass <- true
		}
	}

	dex := &Dexter{
		waiter:          make(chan os.Signal),
		targets:         []*Target{},
		forceKillWindow: 1 * time.Second,
		exitFunc:        dummyExitFunc,
	}
	signal.Notify(dex.waiter, syscall.SIGINT, syscall.SIGTERM)
	dex.Track(stage1Stuck)

	go func() {
		// kill after it waiting for small amount of time
		time.Sleep(10 * time.Millisecond)
		syscall.Kill(os.Getpid(), syscall.SIGINT)
	}()

	go dex.WaitAndKill()
	// this is proxy dummyExitFunc above working
	select {
	case <-pass:
		// good - it worked
	case <-fail:
		// got unexpected code in return
		t.Fail()
	}

}
