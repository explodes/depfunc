depfunc
=======

`depfunc` is a package that enables concurrent resolution of dependent actions.

Image you have a network of actions that should be evaluated, by this depends on that, and that depends on another.

This package allows the construction of such a graph and a way to evaluate the result.

# Motivation

It seemed like a good idea at the time.

# Applesauce

Imagine applesauce.

To produce applesauce, you need apples, sugar, and a can. To make a can, you need metal. You also might want to perform 
some kind of QA on the canning process.

Your manufacturing process may look something like this:

```plain

 | [apples]   [sugar]   [metal]
 |      \       /         /
 |       \     /         /
 |        \   /   [qa] [can]
 |         \  \    /  /
 ↓          [applesauce]
```

You could create your graph like this:

```go
graph := depfunc.NewGraph()

graph.AddAction("apples", growApples)
graph.AddAction("sugars", growSugar)
graph.AddAction("metals", recycleMetal)
graph.AddAction("cans", formCans)
graph.AddAction("applesauce", canApplesauce)
graph.AddAction("qa", qa)

graph.LinkDependency("metals", "cans")
graph.LinkDependency("cans", "applesauce")

graph.LinkDependency("apples", "applesauce")
graph.LinkDependency("sugars", "applesauce")
graph.LinkDependency("qa", "applesauce")
```

Great! Now every season you can run it like so:

```go
graph.Resolve(context.Background(), &factory{})
```

And your factory will be full of various products.
