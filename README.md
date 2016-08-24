# psyringe

[![CircleCI](https://circleci.com/gh/samsalisbury/psyringe.svg?style=svg)](https://circleci.com/gh/samsalisbury/psyringe)
[![codecov](https://codecov.io/gh/samsalisbury/psyringe/branch/master/graph/badge.svg)](https://codecov.io/gh/samsalisbury/psyringe)
[![Go Report Card](https://goreportcard.com/badge/github.com/samsalisbury/psyringe)](https://goreportcard.com/report/github.com/samsalisbury/psyringe)
[![GoDoc](https://godoc.org/github.com/samsalisbury/psyringe?status.svg)](https://godoc.org/github.com/samsalisbury/psyringe)
[![License MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Psyringe is a [fast], **p**arallel, [lazy], easy to use [dependency injector] for [Go].

```go
//file: example_test.go

type Speaker struct {
	Message    string
	MessageLen int
}

// Contrived example showing how to create a Psyringe with interdependent
// constructors and then inject their values into a struct that depends on them.
func Example() {
	p := psyringe.New(
		func() string { return "Hi!" },       // string constructor
		func(s string) int { return len(s) }, // int constructor (needs string)
	)
	v := Speaker{}
	if err := p.Inject(&v); err != nil {
		panic(err) // a little drastic I'm sure
	}
	fmt.Printf("Speaker says %q in %d characters.", v.Message, v.MessageLen)
	// output:
	// Speaker says "Hi!" in 3 characters.
}
```

[Fully documented at GoDoc.org].

[lazy]: https://en.wikipedia.org/wiki/Lazy_initialization
[dependency injector]: https://en.wikipedia.org/wiki/Dependency_injection
[Go]: https://golang.org
[simple usage example]: #simple-usage-example
[Fully documented at GoDoc.org]: https://godoc.org/github.com/samsalisbury/psyringe
[fast]: ./bench_test.go

## Features

- **[Concurrent initialisation]:** with no extra work on your part.
- **[No tags]:** keep your code clean and readable.
- **[Simple API]:** usually only needs two calls: `p := psyringe.New()` and `p.Inject()`
- **[Supports advanced use cases]:** e.g. [scopes] and [named instances].

[Concurrent initialisation]: #concurrent-initialisation
[No tags]: #no-tags
[Simple API]: #simple-api
[Supports advanced use cases]: #advanced-uses

[scopes]: #scopes
[named instances]: #named-instances

### Concurrent Initialisation

A dependency graph already contains enough information to know which parts can be run concurrently: Any two dependencies in the graph that do not have an edge between them can be generated at the same time. Psyringe uses channels internally to represent the edges in the graph, piping the results of each successful constructor to all the other constructors that need its generated value. The beauty of Go's channel primitives mean this graph is determined implicitly without heavy up-front analysis.

### No Tags

Unlike most dependency injectors for Go, this one does not require you to litter your structs with tags. Instead, it relies on well-written Go code to perform injection based solely on the types of your struct fields and constructor parameters.

### Simple API

Psyringe follows through on its metaphor. You `Add` things to the psyringe, then you `Inject` them into other things. Psyringe does not try to provide any other features, but instead makes it easy to implement more advanced features like scoping yourself. For example, you can create multiple Psyringes and have each of them inject different dependencies into the same struct.

### Advanced Uses

Although the API is simple, and doesn't explicitly support scopes or named instances, these things are trivial to implement yourself. For example, scopes can be created by using multiple Psyringes, one at application level, and another within a http request, for example. See a complete example HTTP server using multiple scopes, below.

Likewise, named instances (i.e. multiple different instances of the same type) can be created by aliasing the type name, or wrapping it in a struct.

### Why "psyringe"?

Psyringe is a _parallel syringe_ which automatically injects values concurrently, based on the needs of your interdependent constructors.

## Usage

Create a new psyringe with `p := psyringe.New` passing in constructors and other values.
Then, call `p.Inject(...)` to inject those values into structs with correspondingly typed fields.

Please see [the documentation] for more usage examples.

[the documentation]: https://godoc.org/github.com/samsalisbury/psyringe

### Advanced usage

- **[Named instances]**: You can only have one constructor or value of a given [injection type] per psyringe. However, [you can use named types] to differentiate values of the same underlying type. This has the side benefit of making code more readable.
- **[Scopes]**: All values in a psyringe are singletons, and are created exactly once, if at all. However, you can [use multiple psyringes] to create your own scopes easily, and use `Clone()` to avoid paying the small initialisation cost of the psyringe itself more than once.

[you can use named types]: #named-instances
[use multiple psyringes]: #scopes

#### Named Instances

Sometimes, you may need to inject more than one value of the same type. For example, the following struct needs 2 strings, `Name` and `Desc`:

```go
type Something struct { Name, Desc string }
```

As it stands, psyringe would be unable to inject `Name` and `Desc` with different values, since a psyringe can only inject a single value of each type, and they are both `string`. However, by using different [named types], it is possible to inject different values:

```go
type Something struct {
	Name Name
	Desc Description
}

type Name string
type Desc string
```

Using these named types can also improve the readability of your code in many cases.

[named types]: https://golang.org/ref/spec#Types

#### Scopes

If you need values with different scopes, then you can use multiple Psyringes, one for each scope. This allows you to precisely control value lifetimes using normal Go code. There is one method which supports this use case: `Clone`. The main use of `Clone` is to generate a fresh psyringe based on the constructors and values of one you've already defined. This is cheaper than filling a blank psyringe from scratch. See this example HTTP server below:

```go
var appScopedPsyringe, requestScopedPsyringe psyringe.Psyringe

func main() {
	appScopedPsyringe = psyringe.New(ApplicationScopedThings...)
	requestPsyringe = psyringe.New(RequestScopedThings...)
	http.HandleFunc("/", HandleHTTPRequest)
	http.ListenAndServe(":8080")
}

func HandleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	var controller Controller
	switch r.URL.Path {
	default:
		controller = &NotFoundController{}
	case "/":
		controller = &HomeController{}
	case "/about":
		controller = &AboutController{}	
	}
	// First inject app-scoped things into the controller...
	if err := appScopedPsyringe.Inject(&controller); err != nil {
		w.WriterHeader(500)
		fmt.Fprintf(w, "Error injecting app-scoped values: %s", err)
		return
	}
	// Then inject request-scoped things... Later injections beat earlier
	// ones, in case both psyringes inject the same type.
	// Note the use of Clone() here. That is important, as once you call
	// Inject on a psyringe, it uses up all the invoked constructors, and
	// replaces them with their constructed values. Clone() creates a
	// bytewise copy of the psyringe value at this point, copying all
	// values that it has realised so far, as well as any constructors
	// that are still needed to construct as-yet unrealised values.
	if err := requestPsyringe.Clone().Inject(&controller); err != nil {
		w.WriterHeader(500)
		fmt.Fprintf(w, "Error injecting request-scoped values: %s", err)
		return
	}
	controller.HandleRequest(w, r)
}
```

### How does it work?

Each item you pass into `Add` or `New` is analysed to see whether or not it is a [constructor]. If it is a constructor, then the type of its first return value is registered as its [injection type]. Otherwise the item is considered to be a _value_ and its own type is used as its injection type. Your psyringe knows how to inject values of each registered injection type.

All the constructors together form a dependency graph, where each parameter in each constructor needs a corresponding constructor or value in the same psyringe in order for it to be successfully invoked. You can call `Psyringe.Test` to check that all constructor parameters are satisfied within a Psyringe, and that the graph is acyclic.

When you call `p.Inject(&someStruct)`, each field in `someStruct` is populated with an item of the corresponding injection type from the `Psyringe` `p`. For constructors, it will call that constructor exactly once to generate its value, if needed. For non-constructor values that were passed in to `p`, it will simply inject that value when called to.

For each constructor parameter in each constructor, you will need to `Add`, in order for that constructor to be successfully invoked. If not, `Inject` will return an error.

Likewise, if the constructor is successfully _invoked_, but returns an error as its second return value, then `Inject` will return the first such error encountered. Thus you can return meaningful errors from your constructors, and handle them in one place in your app.

[injection type]: #injection-types
[constructor]: #constructors


#### Injection Types

Values and constructors passed into a psyringe have an implicit **_injection type_** which is the type of value that thing represents. For non-constructor values, the injection type is the type of the value passed into the psyringe. For constructors, it is the type of the first output (return) value. It is important to understand this concept, since a single psyringe can have only one value or constructor per injection type. `Add` will return an error if you try to register multiple values and/or constructors that resolve to the same injection type.

#### Constructors

Constructors can take 2 different forms:

1. `func(...Anything) Anything`
2. `func(...Anything) (Anything, error)`

Just to clarify: `Anything` means literally any type, and in the signatures above can have a different value each time it is seen. For example, all of the following types are considered to be constructors:

    func() int
    func() (int, error)
    func(int) int
    func(int) (int, error) 
    func(string, io.Reader, io.Writer) interface{}
    func(string, io.Reader, io.Writer) (interface{}, error)

If you need to inject a function which has a constructor's signature, you'll need to create a constructor that returns that function. For example, for a value with injection type `func(int) (int, error)`, you would need to create a func to return that func, otherwise psyringe will think it's a constructor for int.

```go
func newFunc() func(int) (int, error) {
	return func(int) (int, error) { return 0, nil }
}
```

# TODO

- Add examples for: New, Add, Clone, Inject
- Add HTTP server example
- Make injection more efficient.
  (The benchmarks imply this is relatively expensive still, may be
  worth caching injection plan per target type, for use in cloned
  Psyringes.)
- Add support for injection cancellation, maybe with golang.org/x/net/context 
- Find other benchmarks to compare with.
- Add Even Lazier TM injection using struct func fields.
- Add Windows build with AppVeyor 
- Never add tags! Never!

===
Copyright (c) 2016 Sam Salisbury; [License MIT](./LICENSE)
