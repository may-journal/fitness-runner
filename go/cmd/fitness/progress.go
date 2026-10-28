package main

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/render"
)

// heartbeatEvery is how often the heartbeat names each running check.
const heartbeatEvery = 5 * time.Second

// progressEnv turns the heartbeat on off a terminal, as a requirement test
// that has none needs.
const progressEnv = "FITNESS_PROGRESS"

// liveCheck is a running check as its progress line names it.
type liveCheck struct {
	name   string
	start  time.Time
	budget time.Duration
}

// line is the check's progress line: elapsed time, name and budget.
func (c liveCheck) line() string {
	return fmt.Sprintf("  … %s  running: %s (budget %s)\n", span(time.Since(c.start)), c.name, span(c.budget))
}

// span renders a duration the way a person reads it: tenths of a second
// below ten seconds, whole seconds below a minute, then minutes.
func span(d time.Duration) string {
	if d >= time.Minute {
		d = d.Round(time.Second)
		if d%time.Minute == 0 {
			return fmt.Sprintf("%dm", d/time.Minute)
		}
		return d.String()
	}
	if d >= 10*time.Second {
		return trimFloat(d.Round(time.Second).Seconds()) + "s"
	}
	return trimFloat(d.Round(100*time.Millisecond).Seconds()) + "s"
}

// heartbeatOn reports whether a run prints the heartbeat: on a terminal, or
// when FITNESS_PROGRESS=1 asks for it.
func heartbeatOn() bool {
	return os.Getenv(progressEnv) == "1" || render.StderrIsTerminal()
}

// startHeartbeat prints each running check's progress line every interval
// until the returned func runs, which waits for the last line to finish.
func startHeartbeat(every time.Duration) (stop func()) {
	ticker := time.NewTicker(every)
	done, ended := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(ended)
		for {
			select {
			case <-ticker.C:
				printRunning()
			case <-done:
				return
			}
		}
	}()
	return func() {
		ticker.Stop()
		close(done)
		<-ended
	}
}

// printRunning prints the progress line of every running check, the
// earliest started first.
func printRunning() {
	for _, c := range runningChecks() {
		os.Stderr.WriteString(c.line())
	}
}

// runningChecks lists the running checks by start time.
func runningChecks() []liveCheck {
	running.Lock()
	defer running.Unlock()
	checks := make([]liveCheck, 0, len(running.pids))
	for _, c := range running.pids {
		checks = append(checks, c)
	}
	slices.SortFunc(checks, func(a, b liveCheck) int { return a.start.Compare(b.start) })
	return checks
}

// warnHalfway prints c's progress line once when it passes half its budget,
// so a log with no heartbeat still names the slow check. The returned func
// cancels the line for a check that ends first.
func warnHalfway(c liveCheck, heartbeat bool) (cancel func()) {
	if heartbeat {
		return func() {}
	}
	t := time.AfterFunc(c.budget/2, func() { os.Stderr.WriteString(c.line()) })
	return func() { t.Stop() }
}

// stream holds one check's stderr. Once the check is the last one running,
// it also passes the stderr to the runner's own as it arrives, so a long
// final check is never silent.
type stream struct {
	mu   sync.Mutex
	buf  bytes.Buffer
	live bool
}

// Write keeps p, and passes it on while the stream is live.
func (s *stream) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.live {
		os.Stderr.Write(p)
	}
	return s.buf.Write(p)
}

// goLive prints what the stream holds so far and passes the rest on as it
// arrives.
func (s *stream) goLive() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.live {
		os.Stderr.Write(s.buf.Bytes())
		s.live = true
	}
}

// replay prints the held stderr after the run, unless it already streamed.
func (s *stream) replay() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.live {
		os.Stderr.Write(s.buf.Bytes())
	}
}

// lastLine is the last line with text the check wrote to stderr, or "".
func (s *stream) lastLine() string {
	text := s.text()
	return strings.TrimSpace(text[strings.LastIndexByte(text, '\n')+1:])
}

// text is all the check wrote, trimmed; empty for a check with no stream.
func (s *stream) text() string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.TrimSpace(s.buf.String())
}

// streams holds every check's stderr in a run, and makes the last check
// still running stream live.
type streams struct {
	mu   sync.Mutex
	all  []*stream
	left map[int]bool
}

// newStreams makes one stream per check. A run of one check streams from
// the start.
func newStreams(n int) *streams {
	s := &streams{all: make([]*stream, n), left: map[int]bool{}}
	for i := range s.all {
		s.all[i] = &stream{}
		s.left[i] = true
	}
	s.streamLast()
	return s
}

// watch is how a run follows one check: where its stderr goes, and whether
// the heartbeat already names it while it runs.
type watch struct {
	stderr    *stream
	heartbeat bool
}

// watch is how the run follows check i.
func (s *streams) watch(i int, heartbeat bool) watch {
	return watch{stderr: s.all[i], heartbeat: heartbeat}
}

// finish marks check i done, so a lone check left starts streaming.
func (s *streams) finish(i int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.left, i)
	s.streamLast()
}

// streamLast makes the one check left stream live; the caller holds mu.
func (s *streams) streamLast() {
	if len(s.left) != 1 {
		return
	}
	for i := range s.left {
		s.all[i].goLive()
	}
}
