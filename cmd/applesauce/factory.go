package main

import (
	"context"
	"time"

	"math/rand"

	"fmt"

	"strconv"

	"sync"

	"flag"

	"github.com/explodes/depfunc"
)

var (
	startDone = flag.Bool("done", false, "cancel the resolve context immediately to see what happens")
	withCycle = flag.Bool("cycle", false, "add a cycle to the graph to see what happens")
	logDebug  = flag.Bool("debug", false, "print debug messages")

	timeMin = flag.Int("timemin", 100, "minimum time to run each factory")
	timeMax = flag.Int("timemax", 500, "maximum time to run each factory")

	valueMin = flag.Int("valuemin", 1e3, "minimum value of each factory")
	valueMax = flag.Int("valuemax", 1e6, "maximum value of each factory")

	numResolves         = flag.Int("resolves", 1, "number of times to generate applesauce")
	resolveConcurrently = flag.Bool("concurrent", false, "resolve concurrently")

	showStats = flag.Bool("stats", false, "print stats of a resolve")
)

var (
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func main() {
	flag.Parse()

	graph := depfunc.NewGraph()

	must(graph.AddAction("apples", growApples()))
	must(graph.AddAction("sugars", growSugar()))
	must(graph.AddAction("metals", recycleMetal()))
	must(graph.AddAction("cans", formCans()))
	must(graph.AddAction("applesauce", canApplesauce()))
	must(graph.AddAction("qa", qa()))

	must(graph.LinkDependency("metals", "cans"))
	must(graph.LinkDependency("apples", "applesauce"))
	must(graph.LinkDependency("sugars", "applesauce"))
	must(graph.LinkDependency("cans", "applesauce"))

	must(graph.LinkDependency("apples", "qa"))
	must(graph.LinkDependency("sugars", "qa"))
	must(graph.LinkDependency("cans", "qa"))

	if *withCycle {
		must(graph.LinkDependency("qa", "sugars"))
	}

	ctx, done := context.WithCancel(context.Background())
	if *startDone {
		done()
	}

	// Resolve the graph multiple times concurrently
	wg := &sync.WaitGroup{}
	for i := 0; i < *numResolves; i++ {
		wg.Add(1)
		exec := func() {

			stats := depfunc.NewStatistics()

			var recorders []depfunc.Recorder
			if *showStats {
				recorders = []depfunc.Recorder{stats.Recorder()}
			}

			defer wg.Done()
			answers := &Answers{}
			ctx, err := graph.Resolve(ctx, answers, recorders...)
			must(err)
			select {
			case <-ctx.Done():
				fmt.Printf("%s\n", answers)
			}

			if *showStats {
				printStats(stats)
			}
		}
		if *resolveConcurrently {
			go exec()
		} else {
			exec()
		}
	}
	wg.Wait()
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type Answer struct {
	Value int
}

func (a *Answer) String() string {
	return strconv.Itoa(a.Value)
}

type Answers struct {
	Sugars *Answer
	Apples *Answer

	Metals *Answer
	Cans   *Answer

	AppleSauce *Answer

	QA *Answer
}

func (a *Answers) String() string {
	return fmt.Sprintf("sugar=%s apple=%s metal=%s can=%s sauce=%s qa=%s", a.Sugars, a.Apples, a.Metals, a.Cans, a.AppleSauce, a.QA)
}

func growApples() depfunc.Action {
	return makeAction("apple", func(answers *Answers, answer *Answer) {
		answers.Apples = answer
	})
}

func growSugar() depfunc.Action {
	return makeAction("sugar", func(answers *Answers, answer *Answer) {
		answers.Sugars = answer
	})
}

func qa() depfunc.Action {
	return makeAction("qa", func(answers *Answers, answer *Answer) {
		answers.QA = answer
	})
}

func recycleMetal() depfunc.Action {
	return makeAction("metal", func(answers *Answers, answer *Answer) {
		answers.Metals = answer
	})
}

func formCans() depfunc.Action {
	return makeAction("can", func(answers *Answers, answer *Answer) {
		// Can recipe:
		// - 2 metals
		answer.Value = answers.Metals.Value / 2
		answers.Cans = answer
	})
}

func canApplesauce() depfunc.Action {
	return makeAction("applesauce", func(answers *Answers, answer *Answer) {
		// Applesauce recipe:
		// - 2 apples
		// - 4 sugars
		// - 1 can
		apples := answers.Apples.Value
		sugars := answers.Sugars.Value
		cans := answers.Cans.Value
		answer.Value = min(apples/2, min(sugars/4, cans))

		answers.AppleSauce = answer
	})
}

func makeAction(name string, assign func(*Answers, *Answer)) depfunc.Action {
	return func(ctx context.Context, arg interface{}) {
		answers := arg.(*Answers)
		debug("→%s", name)
		select {
		case <-time.After(time.Duration(*timeMin) + time.Duration(rng.Intn(*timeMax-*timeMin))*time.Millisecond):
			answer := &Answer{Value: *valueMin + rng.Intn(*valueMax-*valueMin)}
			assign(answers, answer)
			debug("←%s: made %d", name, answer.Value)
		case <-ctx.Done():
			return
		}
	}
}

func debug(msg string, args ...interface{}) {
	if *logDebug {
		fmt.Printf("%s\n", fmt.Sprintf(msg, args...))
	}
}

func printStats(stats *depfunc.Statistics) {
	for name := range stats.Names() {
		fmt.Printf("%s: wait=%v action=%v total=%v\n", name, stats.Wait(name), stats.Action(name), stats.Total(name))
	}
}
