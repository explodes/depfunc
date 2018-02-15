package depfunc

import (
	"sync"
	"time"
)

// VisitRecorder is used to monitor events
// when resolving a Graph
type VisitRecorder interface {
	// Enter is when an Action is prepared
	// to be resolved
	Enter(name string)

	// Start is when an Action begins execution
	Start(name string)

	// Finish is when an Action has finished execution
	Finish(name string)

	// Exit is when an Action has finished or
	// was aborted because of error or finished Context
	Exit(name string)
}

// Statistics is a way to get statistics about resolved (or cancelled) actions.
// To record statistics, use the .VisitRecorder() method to get a VisitRecorder
// that can be used with Resolve. Statistics should not be re-used between Resolves.
type Statistics struct {
	*sync.RWMutex
	enter  map[string]time.Time
	start  map[string]time.Time
	finish map[string]time.Time
	exit   map[string]time.Time

	recorder *recorder
}

// NewStatistics creates a new Statistics. Statistics can be used to analyze a Resolve.
// It should not be re-used between multiple Graph Resolves.
func NewStatistics() *Statistics {
	p := &Statistics{
		RWMutex: &sync.RWMutex{},
		enter:   make(map[string]time.Time),
		start:   make(map[string]time.Time),
		finish:  make(map[string]time.Time),
		exit:    make(map[string]time.Time),
	}

	return p
}

// VisitRecorder returns a VisitRecord that will record details into this Statistics.
// It should not be re-used between multiple Graph Resolves.
func (s *Statistics) VisitRecorder() VisitRecorder {
	if s.recorder == nil {
		s.recorder = &recorder{
			RWMutex: s.RWMutex,
			enter:   s.enter,
			start:   s.start,
			finish:  s.finish,
			exit:    s.exit,
		}
	}
	return s.recorder
}

// Names returns the set of Names this Statistics has information about.
func (s *Statistics) Names() StringSet {
	ss := make(StringSet)

	s.RLock()
	addKeys(ss, s.enter)
	addKeys(ss, s.start)
	addKeys(ss, s.finish)
	addKeys(ss, s.exit)
	s.RUnlock()

	return ss
}

func addKeys(ss StringSet, m map[string]time.Time) {
	for key := range m {
		ss.Add(key)
	}
}

func (s *Statistics) duration(from, to map[string]time.Time, name string) time.Duration {
	s.RLock()
	defer s.RUnlock()

	b, ok := to[name]
	if !ok {
		return 0
	}
	a, ok := from[name]
	if !ok {
		return 0
	}
	return b.Sub(a)
}

// Action returns the duration of the actual execution of
// an Action, or 0 if the action was not executed.
func (s *Statistics) Action(name string) time.Duration {
	return s.duration(s.start, s.finish, name)
}

// Wait returns how long the Action waited to be executed
// or 0 if the action was not executed.
func (s *Statistics) Wait(name string) time.Duration {
	return s.duration(s.enter, s.start, name)
}

// Total returns the amount of time between preparing the Action
// and either aborting or finishing execution of the Action.
func (s *Statistics) Total(name string) time.Duration {
	return s.duration(s.enter, s.exit, name)
}

// recorder is a helper for Statistics that implements the VisitRecorder interface
type recorder struct {
	*sync.RWMutex
	enter  map[string]time.Time
	start  map[string]time.Time
	finish map[string]time.Time
	exit   map[string]time.Time
}

func (p *recorder) recordTime(m map[string]time.Time, name string) {
	p.Lock()
	m[name] = time.Now()
	p.Unlock()
}

func (p *recorder) Enter(name string) {
	p.recordTime(p.enter, name)
}

func (p *recorder) Start(name string) {
	p.recordTime(p.start, name)
}

func (p *recorder) Finish(name string) {
	p.recordTime(p.finish, name)
}

func (p *recorder) Exit(name string) {
	p.recordTime(p.exit, name)
}

// visitRecorderList is a VisitRecorder that
// operates on a slice of VisitRecorders
type visitRecorderList struct {
	recorders []VisitRecorder
}

func (v visitRecorderList) Enter(name string) {
	for _, vr := range v.recorders {
		vr.Enter(name)
	}
}

func (v visitRecorderList) Start(name string) {
	for _, vr := range v.recorders {
		vr.Start(name)
	}
}

func (v visitRecorderList) Finish(name string) {
	for _, vr := range v.recorders {
		vr.Finish(name)
	}
}

func (v visitRecorderList) Exit(name string) {
	for _, vr := range v.recorders {
		vr.Exit(name)
	}
}

// noopVisitRecorder is a VisitRecorder that does nothing
type noopVisitRecorder struct{}

func (*noopVisitRecorder) Enter(name string) {}

func (*noopVisitRecorder) Start(name string) {}

func (*noopVisitRecorder) Finish(name string) {}

func (*noopVisitRecorder) Exit(name string) {}

func optionalVisitRecorder(recorders ...VisitRecorder) VisitRecorder {
	switch len(recorders) {
	case 0:
		var noop *noopVisitRecorder
		return noop
	case 1:
		return recorders[0]
	default:
		return visitRecorderList{recorders: recorders}
	}
}
