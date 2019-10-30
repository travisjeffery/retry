package retry

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Run retries the function on a 25ms interval for 2s stopping if it succeeds.
func Run(t T, f func(r *R)) {
	RunWith(t, &Timer{Timeout: 2 * time.Second, Wait: 25 * time.Millisecond}, f)
}

func RunWith(t T, r Retryer, f func(r *R)) {
	run(t, r, f)
}

// T is an interface compatible with testing.T.
type T interface {
	// Log is called for the final test output
	Log(args ...interface{})

	// FailNow is called when the retrying is abandoned.
	FailNow()
}

// Retryer provides an interface for repeating operations
// until they succeed or an exit condition is met.
type Retryer interface {
	// Next returns true if the operation should be repeated.
	// Otherwise, it calls fail and returns false.
	Next(fail func()) bool
}

// R provides context for the retryer.
type R struct {
	fail   bool
	output []string
}

func (r *R) FailNow() {
	r.fail = true
	runtime.Goexit()
}

func (r *R) Fatal(args ...interface{}) {
	r.log(fmt.Sprint(args...))
	r.FailNow()
}

func (r *R) Fatalf(format string, args ...interface{}) {
	r.log(fmt.Sprintf(format, args...))
	r.FailNow()
}

func (r *R) Error(args ...interface{}) {
	r.log(fmt.Sprint(args...))
	r.fail = true
}

func (r *R) Check(err error) {
	if err != nil {
		r.log(err.Error())
		r.FailNow()
	}
}

func (r *R) log(s string) {
	r.output = append(r.output, decorate(s))
}

// Timer repeats an operation for a given amount
// of time and waits between subsequent operations.
type Timer struct {
	Timeout time.Duration
	Wait    time.Duration

	// stop is the timeout deadline.
	// Set on the first invocation of Next().
	stop time.Time
}

func (r *Timer) Next(fail func()) bool {
	if r.stop.IsZero() {
		r.stop = time.Now().Add(r.Timeout)
		return true
	}
	if time.Now().After(r.stop) {
		fail()
		return false
	}
	time.Sleep(r.Wait)
	return true
}

func decorate(s string) string {
	_, file, line, ok := runtime.Caller(3)
	if ok {
		n := strings.LastIndex(file, "/")
		if n >= 0 {
			file = file[n+1:]
		}
	} else {
		file = "???"
		line = 1
	}
	return fmt.Sprintf("%s:%d: %s", file, line, s)
}

func dedup(a []string) string {
	if len(a) == 0 {
		return ""
	}
	m := map[string]int{}
	for _, s := range a {
		m[s] = m[s] + 1
	}
	var b bytes.Buffer
	for _, s := range a {
		if _, ok := m[s]; ok {
			b.WriteString(s)
			b.WriteRune('\n')
			delete(m, s)
		}
	}
	return b.String()
}

func run(t T, r Retryer, f func(r *R)) {
	rr := &R{}
	fail := func() {
		out := dedup(rr.output)
		if out != "" {
			t.Log(out)
		}
		t.FailNow()
	}
	for r.Next(fail) {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			f(rr)
		}()
		wg.Wait()
		if rr.fail {
			rr.fail = false
			continue
		}
		break
	}
}
