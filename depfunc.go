package depfunc

import (
	"context"

	"sync"

	"github.com/pkg/errors"
)

// Action is a function to execute after its dependencies have been executed
type Action func(ctx context.Context, arg interface{})

// Graph is a graph of Actions to execute concurrently in dependency order
type Graph struct {
	// treeOrder is the adjacency list where the dependent-most node is a root
	// applesauce < {apples, sugar}
	treeOrder stringmultimap

	// graphOrder is the adjacency list where the root-most node is a root
	// applesauce > {apples, sugar}
	graphOrder stringmultimap

	// actions is the map of actions by name
	actions map[string]Action
}

// NewGraph creates a new Graph
func NewGraph() *Graph {
	return &Graph{
		treeOrder:  make(stringmultimap),
		graphOrder: make(stringmultimap),
		actions:    make(map[string]Action),
	}
}

// AddAction adds an action to the graph
func (g *Graph) AddAction(name string, action Action) error {
	if name == "" {
		return errors.New("name must not be empty")
	}
	g.actions[name] = action
	return nil
}

// LinkDependency creates a dependency between two actions
func (g *Graph) LinkDependency(parent, name string) error {
	if name == "" {
		return errors.New("name must not be empty")
	}
	if _, exists := g.actions[name]; !exists {
		return errors.New("action not added")
	}
	if parent == "" {
		return errors.New("parent name must not be empty")
	}
	if _, exists := g.actions[parent]; !exists {
		return errors.New("parent action not added")
	}
	g.treeOrder.Add(name, parent)
	g.graphOrder.Add(parent, name)
	return nil
}

// Resolve executes this Graph on a given context.
// A child context is returned that is done when the
// Actions are all executed or an error occurs.
func (g *Graph) Resolve(ctx context.Context, arg interface{}, recorders ...VisitRecorder) (context.Context, error) {
	// Create a sub-context in which to execute the Actions in this Graph
	ctx, done := context.WithCancel(ctx)

	// Initialize our search data
	s := search{
		waits:   make(map[string]*sync.WaitGroup),
		visited: make(StringSet),
		path:    make(StringSet),
		ctx:     ctx,
		wg:      &sync.WaitGroup{},
		dfsWait: &sync.WaitGroup{},
		arg:     arg,
	}

	recorder := optionalVisitRecorder(recorders...)

	s.dfsWait.Add(1)
	defer s.dfsWait.Done()

	// Begin DFS on each root.
	rootFound := false
	for root := range g.collectRoots() {
		rootFound = true
		if err := g.dfsResolve(s, "", root, recorder); err != nil {
			done()
			return ctx, err
		}
	}

	if !rootFound {
		done()
		return ctx, errors.New("no roots in graph")
	}

	// Wait for all visits to finish, no errors occurred
	// after our DFS, so we are just waiting for execution to finish.
	go func() {
		s.wg.Wait()
		done()
	}()

	return ctx, nil
}

// dfsResolve will kick of a goroutine for each of our actions.
// Each goroutine will be waiting for its dependencies to complete, so a full
// traversal may be made before any Actions are run.
func (g *Graph) dfsResolve(s search, parent, name string, recorder VisitRecorder) error {
	//if s.searchContextDone() {
	//	return nil
	//}

	s.visited.Add(name)
	s.path.Add(name)

	g.visit(s, name, recorder)

	for child := range g.treeOrder[name] {
		if s.path.Contains(child) {
			return errors.New("cycle detected")
		}
		if s.visited.Contains(child) {
			continue
		}
		if !s.searchContextDone() {
			if err := g.dfsResolve(s, name, child, recorder); err != nil {
				return err
			}
		}
	}

	s.path.Remove(name)
	return nil
}

// visit visits a node in the graph, executing the action for the given name
func (g *Graph) visit(s search, name string, recorder VisitRecorder) {
	action := g.actions[name]

	children := g.treeOrder[name]
	wg := s.createWaitGroupForDependents(name, len(children))

	s.wg.Add(1)
	go func() {
		recorder.Enter(name)
		defer recorder.Exit(name)

		parents := g.graphOrder[name]
		defer s.visitComplete(name, parents)
		s.dfsWait.Wait()
		if s.searchContextDone() {
			return
		}
		wg.Wait()
		if !s.searchContextDone() {
			recorder.Start(name)
			action(s.ctx, s.arg)
			recorder.Finish(name)
		}
	}()
}

// collectRoots collects the name of all actions that have no dependencies
func (g *Graph) collectRoots() <-chan string {
	ch := make(chan string)
	go func() {
		for name := range g.actions {
			if len(g.graphOrder[name]) == 0 {
				ch <- name
			}
		}
		close(ch)
	}()
	return ch
}

// search contains data used during the DFS of resolving Graph actions in Resolve
type search struct {
	// ctx is the context in which actions are performed
	ctx context.Context

	// waits is the map of actions to a wait group waiting for dependencies to be resolved
	waits map[string]*sync.WaitGroup

	// visited is the set of visited actions
	visited StringSet

	// recursion is the stack of the currently visited path for cycle detection
	path StringSet

	// wg is the wait that signifies that Resolve is complete
	wg *sync.WaitGroup

	// wg is the wait that signifies that the dfs is complete
	dfsWait *sync.WaitGroup

	// arg is the Resolve argument
	arg interface{}
}

// visitComplete is an action to be performed after an action's goroutine has ended
func (s *search) visitComplete(name string, parents StringSet) {
	s.wg.Done()
	for parent := range parents {
		parentWg := s.waits[parent]
		if parentWg != nil {
			parentWg.Done()
		}
	}
}

// searchContextDone returns if the context for this search is done
func (s *search) searchContextDone() bool {
	select {
	case <-s.ctx.Done():
		return true
	default:
		return false
	}
}

// createWaitGroupForDependents creates a wait group for a name that
// will wait for each dependent
func (s *search) createWaitGroupForDependents(name string, numDependents int) *sync.WaitGroup {
	wg := &sync.WaitGroup{}
	wg.Add(numDependents)
	s.waits[name] = wg
	return wg
}
