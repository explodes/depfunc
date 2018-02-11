package main

import (
	"context"
	"time"

	"math/rand"

	"fmt"

	"github.com/explodes/scratch/depfunc"
)

const (
	timeMin = 100
	timeMax = 900

	valueMin = 10
	valueMax = 100
)

var (
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func main() {
	graph := depfunc.NewGraph()

	must(graph.AddRootQuestion("apples", growApples))
	must(graph.AddRootQuestion("sugars", growSugar))
	must(graph.AddRootQuestion("metals", recycleMetal))

	must(graph.AddDependentQuestion("metals", "cans", formCans))

	must(graph.AddDependentQuestion("apples", "applesauce", canApplesauce))
	must(graph.AddDependentQuestion("sugars", "applesauce", canApplesauce))
	must(graph.AddDependentQuestion("cans", "applesauce", canApplesauce))

	fmt.Printf("%s\n", graph)
	//fmt.Printf("%+#v\n", graph)

	ctx, _ := context.WithCancel(context.Background())
	//done()

	answers, err := graph.Resolve(ctx)
	must(err)

	select {
	case <-time.After(5 * time.Second):
	}

	fmt.Printf("%s\n", answers)

}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

var (
	growApples = makeQuestion("apple", func(answers *depfunc.Answers, answer *depfunc.Answer) {
		answers.Apples = answer
	})
	growSugar = makeQuestion("sugar", func(answers *depfunc.Answers, answer *depfunc.Answer) {
		answers.Sugars = answer
	})
	recycleMetal = makeQuestion("metal", func(answers *depfunc.Answers, answer *depfunc.Answer) {
		answers.Metals = answer
	})
	formCans = makeQuestion("can", func(answers *depfunc.Answers, answer *depfunc.Answer) {
		// Can recipe:
		// - 2 metals
		answer.Value = answers.Metals.Value / 2
		answers.Cans = answer
	})
	canApplesauce = makeQuestion("applesauce", func(answers *depfunc.Answers, answer *depfunc.Answer) {
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
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func makeQuestion(name string, assign func(*depfunc.Answers, *depfunc.Answer)) depfunc.Question {
	return func(ctx context.Context, answers *depfunc.Answers) context.Context {
		debug("*start %s", name)
		out, done := context.WithCancel(context.Background())
		go func() {
			defer done()
			select {
			case <-time.After(time.Duration(timeMin) + time.Duration(rng.Intn(timeMax-timeMin))*time.Millisecond):
				answer := &depfunc.Answer{Value: valueMin + rng.Intn(valueMax-valueMin)}
				assign(answers, answer)
				fmt.Printf("%s: made %d\n", name, answer.Value)
				debug("*finish %s", name)
			case <-ctx.Done():
				return
			}
		}()
		return out
	}
}

func debug(msg string, args ...interface{}) {
	fmt.Printf("%s\n", fmt.Sprintf(msg, args...))
}
