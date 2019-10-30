package retry_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/travisjeffery/retry"
)

func TestRunWithFail(t *testing.T) {
	r := &retry.Timer{
		Timeout: 1 * time.Second,
		Wait:    100 * time.Millisecond,
	}
	run(t, r.Timeout, r.Wait, int(r.Timeout/r.Wait)+1, true, func(ft *fakeT, calls *int) {
		retry.RunWith(ft, r, func(r *retry.R) {
			*calls++
			r.Fatalf("fail")
		})
	})
}

func TestRunWithPass(t *testing.T) {
	r := &retry.Timer{
		Timeout: 1 * time.Second,
		Wait:    100 * time.Millisecond,
	}
	run(t, r.Timeout, r.Wait, 1, false, func(ft *fakeT, calls *int) {
		retry.RunWith(ft, r, func(r *retry.R) {
			*calls++
		})
	})
}

func TestRunFail(t *testing.T) {
	timeout := 2 * time.Second
	wait := 25 * time.Millisecond
	run(t, timeout, wait, int(timeout/wait), true, func(ft *fakeT, calls *int) {
		retry.Run(ft, func(r *retry.R) {
			*calls++
			r.Fatalf("fail")
		})
	})
}

func TestRunPass(t *testing.T) {
	run(t, 2*time.Second, 25*time.Millisecond, 1, false, func(ft *fakeT, calls *int) {
		retry.Run(ft, func(r *retry.R) {
			*calls++
		})
	})

}

type doFunc func(ft *fakeT, calls *int)

func run(t *testing.T, timeout, wait time.Duration, wantCalls int, wantFailed bool, do doFunc) {
	ft := &fakeT{}
	var slow uint64
	time.AfterFunc(timeout+wait, func() {
		atomic.StoreUint64(&slow, 1)
	})
	var gotCalls int

	do(ft, &gotCalls)

	if gotCalls != wantCalls {
		t.Fatalf("wanted func %d calls, got %d", wantCalls, gotCalls)
	}
	if ft.failed != wantFailed {
		t.Fatalf("wanted t to not have marked as failed")
	}
	// compare-and-swap returns true if value == old/0
	if !atomic.CompareAndSwapUint64(&slow, 0, 1) {
		t.Fatalf("wanted test to have finish before setting slow")
	}
}

type fakeT struct {
	failed bool
}

func (t *fakeT) Log(args ...interface{}) {}

func (t *fakeT) FailNow() {
	t.failed = true
}
