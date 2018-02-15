package depfunc

import (
	"testing"

	"context"

	"strings"

	"time"

	"sync"

	"sync/atomic"

	"github.com/stretchr/testify/assert"
)

const (
	testTimeout = 250 * time.Millisecond
)

type Fataler interface {
	Fatal(args ...interface{})
}

type visitordata struct {
	mx      *sync.Mutex
	visited []string
}

func newVisitordata() *visitordata {
	return &visitordata{
		mx: &sync.Mutex{},
	}
}

func (v *visitordata) Visit(name string) {
	v.mx.Lock()
	v.visited = append(v.visited, name)
	v.mx.Unlock()
}

func visitorAction(name string) Action {
	return func(ctx context.Context, arg interface{}) {
		arg.(*visitordata).Visit(name)
	}
}

func sampleaction(ctx context.Context, arg interface{}) {}

func testContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), testTimeout)
	return ctx
}

func TestNewGraph(t *testing.T) {
	g := NewGraph()

	assert.NotNil(t, g)
}

func TestGraph_AddAction(t *testing.T) {
	g := NewGraph()

	err := g.AddAction("action", sampleaction)

	assert.NoError(t, err)
	assert.Len(t, g.actions, 1)
}

func TestGraph_AddAction_noName(t *testing.T) {
	g := NewGraph()

	err := g.AddAction("", sampleaction)

	assert.Error(t, err)
}

func TestGraph_LinkDependency(t *testing.T) {
	g := NewGraph()
	g.AddAction("a", sampleaction)
	g.AddAction("b", sampleaction)

	err := g.LinkDependency("a", "b")

	assert.NoError(t, err)
	assert.Len(t, g.treeOrder, 1)
	assert.Len(t, g.graphOrder, 1)
}

func TestGraph_LinkDependency_noName(t *testing.T) {
	g := NewGraph()
	g.AddAction("a", sampleaction)
	g.AddAction("b", sampleaction)

	err := g.LinkDependency("a", "")

	assert.Error(t, err)
}

func TestGraph_LinkDependency_noActionForName(t *testing.T) {
	g := NewGraph()
	g.AddAction("a", sampleaction)

	err := g.LinkDependency("a", "b")

	assert.Error(t, err)
}

func TestGraph_LinkDependency_noParentName(t *testing.T) {
	g := NewGraph()
	g.AddAction("a", sampleaction)
	g.AddAction("b", sampleaction)

	err := g.LinkDependency("", "b")

	assert.Error(t, err)
}

func TestGraph_LinkDependency_noActionForParentName(t *testing.T) {
	g := NewGraph()
	g.AddAction("b", sampleaction)

	err := g.LinkDependency("a", "b")

	assert.Error(t, err)
}

func TestGraph_Resolve(t *testing.T) {
	g := NewGraph()
	g.AddAction("a", visitorAction("a"))
	g.AddAction("b", visitorAction("b"))
	g.LinkDependency("a", "b")

	visitorData := newVisitordata()

	ctx, err := g.Resolve(testContext(), visitorData)
	<-ctx.Done()

	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, visitorData.visited)
}

func TestGraph_Resolve_contextDone(t *testing.T) {
	g := NewGraph()
	g.AddAction("a", visitorAction("a"))
	g.AddAction("b", visitorAction("b"))
	g.LinkDependency("a", "b")

	visitorData := newVisitordata()

	resolveCtx, done := context.WithCancel(testContext())
	done()

	ctx, err := g.Resolve(resolveCtx, visitorData)
	if err != nil {
		t.Fatal(err)
	}
	<-ctx.Done()

	assert.Len(t, visitorData.visited, 0)
}

func TestGraph_Resolve_noRoots(t *testing.T) {
	g := NewGraph()
	g.AddAction("a", visitorAction("a"))
	g.AddAction("b", visitorAction("b"))
	g.LinkDependency("a", "b")
	g.LinkDependency("b", "a")

	visitorData := newVisitordata()

	ctx, err := g.Resolve(testContext(), visitorData)
	<-ctx.Done()

	assert.EqualError(t, err, "no roots in graph")
}

func TestGraph_Resolve_deepCycle(t *testing.T) {
	g := NewGraph()
	g.AddAction("a", visitorAction("a"))
	g.AddAction("b", visitorAction("b"))
	g.AddAction("c", visitorAction("c"))
	g.LinkDependency("a", "b")
	g.LinkDependency("b", "c")
	g.LinkDependency("b", "a")

	visitorData := newVisitordata()

	ctx, err := g.Resolve(testContext(), visitorData)
	<-ctx.Done()

	assert.EqualError(t, err, "cycle detected")
}

func definedGraph(t Fataler) *Graph {
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	//       a
	//    //  \\
	//   b c   d h
	//  /  |      \
	// e   g      i
	// |        / \
	// f       j   k
	g := NewGraph()
	must(g.AddAction("a", visitorAction("a")))
	must(g.AddAction("b", visitorAction("b")))
	must(g.AddAction("c", visitorAction("c")))
	must(g.AddAction("d", visitorAction("d")))
	must(g.AddAction("e", visitorAction("e")))
	must(g.AddAction("f", visitorAction("f")))
	must(g.AddAction("g", visitorAction("g")))
	must(g.AddAction("h", visitorAction("h")))
	must(g.AddAction("i", visitorAction("i")))
	must(g.AddAction("j", visitorAction("j")))
	must(g.AddAction("k", visitorAction("k")))

	links := [][2]string{
		{"a", "b"},
		{"b", "e"},
		{"e", "f"},

		{"a", "c"},
		{"c", "g"},

		{"a", "d"},

		{"a", "h"},
		{"h", "i"},
		{"i", "j"},
		{"i", "k"},
	}
	for _, link := range links {
		must(g.LinkDependency(link[0], link[1]))
	}

	return g
}

func deepGraph(t Fataler, depth int) *Graph {
	root := "0"
	g := NewGraph()
	g.AddAction(root, visitorAction(root))
	index := int64(0)
	deepGraphHelper(t, root, depth, &index, g)
	return g
}

func deepGraphHelper(t Fataler, root string, depth int, index *int64, g *Graph) {
	if depth == 0 {
		return
	}
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}

	left := atomic.AddInt64(index, 1)
	right := atomic.AddInt64(index, 1)

	leftName := string(left)
	must(g.AddAction(leftName, visitorAction(leftName)))
	must(g.LinkDependency(root, leftName))
	deepGraphHelper(t, leftName, depth-1, index, g)

	rightName := string(right)
	must(g.AddAction(rightName, visitorAction(rightName)))
	must(g.LinkDependency(root, rightName))
	deepGraphHelper(t, rightName, depth-1, index, g)
}

func TestGraph_collectRoots(t *testing.T) {
	g := definedGraph(t)

	roots := g.collectRoots()

	expected := make(StringSet)
	for _, c := range "fgdjk" {
		expected.Add(string(c))
	}

	actual := make(StringSet)
	for c := range roots {
		actual.Add(string(c))
	}

	assert.Equal(t, expected, actual)
}

func TestGraph_Resolve_dfs(t *testing.T) {
	g := definedGraph(t)

	visitorData := newVisitordata()
	ctx, err := g.Resolve(testContext(), visitorData)
	if err != nil {
		t.Fatal(err)
	}
	<-ctx.Done()

	assert.Len(t, visitorData.visited, 11)
	assertOccursBefore(t, 'a', "bcdefghijk", strings.Join(visitorData.visited, ""))
	assertOccursBefore(t, 'b', "ef", strings.Join(visitorData.visited, ""))
	assertOccursBefore(t, 'c', "g", strings.Join(visitorData.visited, ""))
	assertOccursBefore(t, 'e', "f", strings.Join(visitorData.visited, ""))
	assertOccursBefore(t, 'h', "ijk", strings.Join(visitorData.visited, ""))
	assertOccursBefore(t, 'i', "jk", strings.Join(visitorData.visited, ""))
}

func BenchmarkGraph_Resolve(b *testing.B) {
	g := deepGraph(b, 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		visitorData := newVisitordata()
		ctx, _ := g.Resolve(testContext(), visitorData)
		<-ctx.Done()
	}
}

func BenchmarkGraph_Resolve_recorded(b *testing.B) {
	g := deepGraph(b, 10)
	recorder := NewStatistics().VisitRecorder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		visitorData := newVisitordata()
		ctx, _ := g.Resolve(testContext(), visitorData, recorder)
		<-ctx.Done()
	}
}

func BenchmarkGraph_Resolve_recorded_multiple(b *testing.B) {
	g := deepGraph(b, 10)
	recorderA := NewStatistics().VisitRecorder()
	recorderB := NewStatistics().VisitRecorder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		visitorData := newVisitordata()
		ctx, _ := g.Resolve(testContext(), visitorData, recorderA, recorderB)
		<-ctx.Done()
	}
}

func BenchmarkGraph_Resolve_done(b *testing.B) {
	g := deepGraph(b, 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		visitorData := newVisitordata()
		resolveCtx, done := context.WithCancel(testContext())
		done()
		ctx, _ := g.Resolve(resolveCtx, visitorData)
		<-ctx.Done()
	}
}

func BenchmarkGraph_collectRoots(b *testing.B) {
	g := deepGraph(b, 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for range g.collectRoots() {
		}
	}
}
