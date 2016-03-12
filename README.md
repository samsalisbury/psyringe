# syringe

Syringe is a fast, easy to use, and [lazy]&nbsp;[dependency injector] for [Go].

It's key features are:

- Automatic concurrent initialisation of dependency graphs.
- A very simple API, just 2 calls are needed for most apps: `syringe.Fill()` and `syringe.Inject()`
- No tags, keep your code clean and readable.
- No named instances. Instead, uses named types.
- Lazy initialisation, don't create anything which isn't needed.
- It can be used to implement the [inversion of control (IoC) pattern]
- Suitable for both short and long-running apps (e.g. CLIs and web servers)

See a [simple usage example], below.

[lazy]: https://en.wikipedia.org/wiki/Lazy_initialization
[dependency injector]: https://en.wikipedia.org/wiki/Dependency_injection
[Go]: https://golang.org
[inversion of control (IoC) pattern]: https://en.wikipedia.org/wiki/Inversion_of_control 
[simple usage example]: #simple-usage-example

## Usage

Call `syringe.Fill(...)` to add objects and constructors to the syringe. Then, call syringe.Inject(...)` to inject those objects into other objects which need them.

### Simple Usage Example

```go
package syringe_test

import (
	"fmt"
	"log"

	"github.com/samsalisbury/syringe"
)

type (
	Command struct {
		User Username
		Host Hostname
		Load LoadAverage
	}
	Username    string
	Hostname    string
	LoadAverage float64
)

func NewUsername() Username       { return "bob" }
func NewHostname() Hostname       { return Hostname("localhost") }
func NewLoadAverage() LoadAverage { return 0.83 }

func (c Command) Print() {
	fmt.Printf("User: %s, Host: %s, Load average: %.2f", c.User, c.Host, c.Load)
}

func ExampleSyringe_Simple() {
	s := syringe.Syringe{}
	_, err := s.Fill(NewUsername, NewHostname, NewLoadAverage)
	if err != nil {
		log.Fatal(err)
	}
	command := Command{}
	s.Inject(&command)
	command.Print()
	// output:
	// User: bob, Host: localhost, Load average: 0.83
}
```

### How does it work?

Each item you pass into `.Fill()` is analysed to see whether or not it is a [constructor]. If it is a constructor, then the type of its first return value is registered as its [injection type]. Otherwise the item is consitered to be a _fully realised object,_ and its own type is used as its injection type. Your syringe knows how to inject objects of each registered injection type.

When you call `.Inject(&someStruct)`, each field in someStruct is populated with an item of the corresponding injection type from the syringe. For constructors, it will call that constructor exactly once to generate its object, if needed. For fully realised objects, it will simply inject that object when called to.

Each parameter in a constructor will need to also be available in the syringe, in order for that constructor to be successfully invoked. If not, `.Inject` will return an error.

Likewise, if the constructor is successfully invoked, but return a non-nil error as its second return value, then `.Inject` will return the first such error encountered.

[injection type]: #injection-types
[constructor]: #constructors


### Syringe is inflexible in some regards:

- **No named objects:** You can only have one constructor or fully realised object of a given [injection type] per syringe. Instead, [you use named types] to differentiate objects.
- **No scopes:** All objects in a syringe are singletons, and are created exactly once, if at all. However, you can [use multiple syringes] and `Clone()` to create your own scopes easily.

[you can use named types]: #no-named-objects
[use multiple syringes]: #no-scopes

#### No Named Objects

The first of these issues, "no named objects," is mitigated by using [named types], which also makes your code more precise and readable. For example, if you need to inject some strings, you can do the following, to differentiate them:

```go
type HostName                       string
type SomeExpensiveToCalculateString string
```

[named types]: https://golang.org/ref/spec#Types

#### No Scopes

The second of these issues, "no scoping," is also easily mitigated by using multiple syringes. For example, you could create a new syringe for each HTTP request, if you need request-scoped dependency injection. This can be made computationally cheaper by simply cloning a pre-made syringe for each request.

```go
var appScopedSyringe, requestScopedSyringe syringe.Syringe

func main() {
	appScopedSyringe = syringe.New().Fill(NewApplicationScopedThing, Newfoundland)
	requestSyringe = syringe.New().Fill(NewRequestScopedThing, NewThingamabob)
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

	"github.com/samsalisbury/syringe"
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

## Constructors

Constructors can take 2 different forms:

1. `func (...Anything) Anything`
2. `func (...Anything) (Anything, error)`

Just to clarify: `Anything` means literally any type, and in the signatures above can heave a different value each time it is seen. For example, all of the following types are considered to be constructors:

- func() int
- func() (int, error)
- func(int) int
- func(int) (int, error) 
- func (string, io.Reader, io.Writer) interface{}
- func (string, io.Reader, io.Writer) (interface{}, error)

If you need to inject a fully-realised object which matches a constructor's signature, you'll need to create a function that returns that object. For example, for an object with injection type `func(int) (int, error)`, you would need to create a func to return that:

```go
newFunc() func(int) (int, error) {
	return func(int) (int, error) { return 0, nil }
}
```

