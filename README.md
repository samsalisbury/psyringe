# syringe

Syringe is a [lazy]&nbsp;[dependency injector] for [Go].

It's key feature is lazy, concurrent initialisation of dependency graphs. It can be used to implement the [inversion of control (IoC) pattern], but its primary focus is to speed up initialisation, epecially for command line applications.

```go
syringe.Fill(NewDependency, "hello, cruel world", NewThing, NewSomthingElse, io.Writer(os.Stdout))

dependentObject := SomethingWithDependencies{}

syringe.Inject(&dependentObject) // now your object is ready to use
```

[lazy]: https://en.wikipedia.org/wiki/Lazy_initialization
[dependency injector]: https://en.wikipedia.org/wiki/Dependency_injection
[Go]: https://golang.org
[inversion of control (IoC) pattern]: https://en.wikipedia.org/wiki/Inversion_of_control 

## Usage

You fill up your syringe by passing a mixture of constructors and fully realised objects. Constructors are functions taking any number of arguments of any kind, and returning either a single value (the constructor's [injection type]), or that value plus an `error`. Fully realised objects are objects of any type.

Once your syringe contains the necessary objects and constructors to build your object graph, you call `syringe.Inject` to inject these dependencies into whichever object needs populating with objects in the graph. These dependencies are resolved recursively, and in parallel where the structure allows.

[injection type]: #injection-types

### _Syringe is unflexible in some regards:_

- **No named objects:** You can only have one constructor or fully realised object of a given [injection type] per syringe.
- **No scopes:** All objects in a syringe are singletons, and are created exactly once, if at all.

#### No Named Objects

The first of these issues, "no named objects," is mitigated by using [named types], which also makes your code more precise and readable. For example, if you need to inject some strings, you can do the following, to differentiate them:

```go
type HostName                       string
type SomeExpensiveToCalculateString string
```

[named types]: https://golang.org/ref/spec#Types

#### No Scopes

The second of these issues, "no scoping," is also easily mitigated by using multiple syringes. For example, you could create a new syringe for each HTTP request, if you need request-scoped dependency injection. This is made computationally cheaper by using compiled syringes using `syringe.Compile()`.

```go
var appScopedSyringe, requestScopedSyringe syringe.Syringe

func main() {
	appScopedSyringe = syringe.New().Fill(NewApplicationScopedThing, Newfoundland)
	requestSyringe = syringe.New().Fill(NewRequestScopedThing, NewThingamabob).MustCompile()
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
	if err := appScopedSyringe.Inject(&controller); err != nil {
		w.WriterHeader(500)
		fmt.Fprintf(w, "Error injecting app-scoped objects: %s", err)
		return
	}
	// Then inject request-scoped things... Later injections beat earlier
	// ones, in case both syringes inject the same type.
	if err := requestSyringe.Clone().Inject(&controller); err != nil {
		w.WriterHeader(500)
		fmt.Fprintf(w, "Error: %s", err)
		return
	}
	controller.HandleRequest(w, r)
}
```

### Injection Types

Fully realised objects and constructors passed into a syringe have an implicit **_injection type_** which is the type of object that object or constructor represents, or can create, in the graph. For fully realised objects, the injection type is the type of the variable passed into the syringe. For constructors, it is the type of the first output (return) value. It is important to understand this concept, since a single syringe can have only one object or constructor per injection type.

Here is some go code to illustrate:

```go
package main

import (
	"bytes"
	"fmt"
	"io"

	"gopkg.in/syringe.v1"
)

type HostName string

func main() {
	fmt.Println(syringe.InjectionType("hello world"))
	fmt.Println(syringe.InjectionType(HostName("localhost"))
	fmt.Println(syringe.InjectionType(&bytes.Buffer{})
	fmt.Println(syringe.InjectionType(io.Writer(&bytes.Buffer{}))
	fmt.Println(syringe.InjectionType(io.Reader(&bytes.Buffer{}))
	fmt.Println(syringe.InjectionType(func() io.Reader { return &bytes.Buffer{} })
	fmt.Println(syringe.InjectionType(func() (io.Reader, error) { return &bytes.Buffer{}, nil })
	fmt.Println(syringe.InjectionType(func() {})
	fmt.Println(syringe.InjectionType(func() error { return nil })
	fmt.Println(syringe.InjectionType(func() (error, error) { return nil, nil })
	fmt.Println(syringe.InjectionType(func() (error, error, error) { return nil, nil, nil })
	fmt.Println(syringe.InjectionType(func() (interface{}, interface{}) { return nil, nil })
}
// output:
//
// string
// main.HostName
// *bytes.Buffer
// io.Writer
// io.Reader
// io.Reader
// io.Reader
// func() {}
// error
// error
// func() (error, error, error)
// func() (interface{}, interface{})
```

### simple example

```go
package main

import (
	"log"
	"math/rand"
	"time"

	"gopkg.in/syringe.v1"
)

type (
	SomeHeavyDependency        interface{}
	SlowDependency             interface{}
	HorriblySluggishDependency interface{}
	SomeSingleton              struct{}

	UserFacingCommand struct {
		Heavy     SomeHeavyDependency
		Slow      SlowDependency
		Horrible  HorriblySluggishDependency	
		Singleton SomeSingleton
	}
)

func main() {

	eagerSingleton := SomeSingleton{}

	syringe.Fill(NewHeavyDependency, NewSlowDependency, NewHorriblySluggishDependency, eagerSingleton)
	
	// user instigates UserFacingCommand...

	command := UserFacingCommand{}

	if err := syringe.Inject(&command); err !=  nil { // blocks until all known dependencies are resolved
		log.Fatal(err)
	}

	command.Execute()
}

func doWork() interface{} {
	time.Sleep(rand.Intn(500)*time.Millisecond)
	return nil
}

func NewHeavyDependency()            SomeHeavyDependency        { return doWork() }
func NewSlowDependency()             (SlowDependency, error)    { return doWork() }
func NewHorriblySluggishDependency() HorriblySluggishDependency { return doWork() }

func (ufc UserFacingCommand) Execute() { log.Println("Hello, dear user! I came as quickly as I could!") }

```
