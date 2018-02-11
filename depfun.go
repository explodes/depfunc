package depfunc

import (
	"context"

	"bytes"
	"fmt"
	"strings"
	"sync"

	"strconv"

	"github.com/pkg/errors"
)

type Answer struct {
	Value int
}

func (a Answer) String() string {
	return strconv.Itoa(a.Value)
}

type Answers struct {
	Sugars *Answer
	Apples *Answer

	Metals *Answer
	Cans   *Answer

	AppleSauce *Answer
}

func (a Answers) String() string {
	return fmt.Sprintf("sugar=%s apple=%s metal=%s can=%s sauce=%s", a.Sugars, a.Apples, a.Metals, a.Cans, a.AppleSauce)
}

type Question func(context.Context, *Answers) context.Context

type Graph struct {
	// treeOrder is the adjacency list where the dependent-most node is a root
	// applesauce < {apples, sugar}
	treeOrder map[string]map[string]struct{}

	// graphOrder is the adjacency list where the root-most node is a root
	// applesauce > {apples, sugar}
	graphOrder map[string]map[string]struct{}

	// questions is the set of questions, accessible by name
	questions map[string]Question
}

func NewGraph() *Graph {
	return &Graph{
		treeOrder:  make(map[string]map[string]struct{}),
		graphOrder: make(map[string]map[string]struct{}),
		questions:  make(map[string]Question),
	}
}

func (g *Graph) AddRootQuestion(name string, question Question) error {
	g.questions[name] = question
	return nil
}

func (g *Graph) AddDependentQuestion(parent, name string, question Question) error {
	if name == "" {
		return errors.New("question name must not be empty")
	}
	return g.addInverted(parent, name, question)
}

func (g *Graph) addInverted(parent, name string, question Question) error {
	g.questions[name] = question

	// add "parent" as a child of "name"
	g.addDependent(g.treeOrder, name, parent)
	g.addDependent(g.graphOrder, parent, name)

	return nil
}

func (g *Graph) addDependent(m map[string]map[string]struct{}, parent, name string) {
	adjacent := m[name]
	if adjacent == nil {
		adjacent = make(map[string]struct{})
		m[name] = adjacent
	}
	adjacent[parent] = struct{}{}
}

type search struct {
	waits   map[string]*sync.WaitGroup
	visited map[string]struct{}
	answers *Answers
	ctx     context.Context
}

func (g *Graph) Resolve(ctx context.Context) (*Answers, error) {
	s := search{
		waits:   make(map[string]*sync.WaitGroup),
		visited: make(map[string]struct{}),
		answers: &Answers{},
		ctx:     ctx,
	}

	for root := range g.questions {
		g.dfsResolve(s, "", root)
	}

	return s.answers, nil
}

func (g *Graph) dfsResolve(s search, parent, name string) {
	if _, visited := s.visited[name]; visited {
		return
	}
	s.visited[name] = struct{}{}

	g.visit(s, parent, name)
	for child := range g.treeOrder[name] {
		select {
		case <-s.ctx.Done():
			return
		default:
			g.dfsResolve(s, name, child)
		}
	}
}

func (g *Graph) collectRoots() []string {

	//var roots []string
	//
	//for name := range g.questions {
	//	if len(g.treeOrder[name]) == 0 {
	//		roots = append(roots, name)
	//	}
	//}
	//
	//fmt.Println("roots", roots)
	//return roots

	return []string{"applesauce"}
}

func (g *Graph) visit(s search, parent, name string) {
	numChildren := len(g.treeOrder[name])
	question := g.questions[name]

	wg := &sync.WaitGroup{}
	wg.Add(numChildren)
	s.waits[name] = wg

	go func() {
		wg.Wait()

		ctx := question(s.ctx, s.answers)
		select {
		case <-ctx.Done():
		}

		parentWg := s.waits[parent]
		if parentWg != nil {
			parentWg.Done()
		}
	}()

}

func (g *Graph) String() string {
	buf := &bytes.Buffer{}

	for _, root := range g.collectRoots() {
		g.stringHelper(buf, g.treeOrder, root, 0)
	}

	return buf.String()
}

func (g *Graph) stringHelper(buf *bytes.Buffer, m map[string]map[string]struct{}, name string, depth int) {
	buf.WriteString(fmt.Sprintf("%s%s\n", strings.Repeat("-", depth), name))
	for child := range m[name] {
		g.stringHelper(buf, m, child, depth+1)
	}
}
