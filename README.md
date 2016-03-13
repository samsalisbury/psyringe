# syringe

Syringe is a fast, [lazy], easy to use [dependency injector] for [Go].

```go
syringe.Fill(ValuesAndConstructors...)
target := SomeThing{}
syringe.Inject(&target)
```

See a [simple usage example], below.

[lazy]: https://en.wikipedia.org/wiki/Lazy_initialization
[dependency injector]: https://en.wikipedia.org/wiki/Dependency_injection
[Go]: https://golang.org
[simple usage example]: #simple-usage-example

## Features

- **[Concurrent, lazy initialisation]:** with no extra work on your part.
- **[No tags]:** keep your code clean and readable.
- **[Simple API]:** usually only need two calls: `syringe.Fill()` and `syringe.Inject()`
- **[Supports advanced use cases]:** e.g. [scopes], [named instances], [debugging]

[Concurrent, lazy initialisation]: #concurrent-and-lazy
[No tags]: #no-tags
[Simple API]: #simple-api
[Supports advanced use cases]: #advanced-uses

[scopes]: #scopes
[named instances]: #named-instances
[debugging]: #debugging

### Concurrent and lazy

Value graphs are populated recursively, and concurrently where the structure allows. For example, in the following code, both `NewRay` and `NewComb` will be run simultaneously when running `syringe.Inject` since neither depends on the other. However `NewWhat` will not be run, since `NeedsWidget` does not have a `What` field.

```go
func NewRay() RayMachine              { return RayMachine{} }
func NewComb() Combinator             { return Combinator{} }
func NewWhat() What                   { panic("this won't be called") }
func NewWidget(x Ray, y *Comb) Widget { return Widget{x, y} }

type NeedsWidget struct { Widget Widget }

syringe.Fill(NewRay, NewComb, NewWidget)
nw := NeedsWidget{}
syringe.Inject(&nw)
```

### No Tags

Unlike most dependency injectors for Go, this one does not require you to litter your structs with tags. Instead, it relies on well-written Go code to perform injection based solely on the types of your struct fields.

### Simple API

Syringe follows through on its metaphor. You `Fill` the syringe with things, then you `Inject` them into other things. Syringe does not try to provide any other features, but instead makes it easy to implement more advanced features like dependency scoping yourself. For example, you can create multiple syringes and have all of them inject different dependencies into the same struct.

### Advanced Uses

Although the API is simple, and doesn't explicitly support scopes or named instances, these things are trivial to implement yourself. For example, scopes can be created by using multiple syringes, one at application level, and another within a http request, for example. See a complete example HTTP server using multiple scopes, below.

Likewise, named instances (i.e. multiple different instances of the same type) can be created by aliasing the type name.

## Usage

Call `syringe.Fill(...)` to add values and constructors to the syringe. Then, call `syringe.Inject(...)` to inject those values into structs which need them.

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
	if err := s.Fill(NewUsername, NewHostname, NewLoadAverage); err != nil {
		log.Fatal(err)
	}
	command := Command{}
	s.Inject(&command)
	command.Print()
	// output:
	// User: bob, Host: localhost, Load average: 0.83
}

```

### Advanced usage

- **[Named instances]**: You can only have one constructor or value of a given [injection type] per syringe. However, [you can use named types] to differentiate values of the same underlying type. This has the side benefit of making code more readable.
- **[Scopes]**: All values in a syringe are singletons, and are created exactly once, if at all. However, you can [use multiple syringes] to create your own scopes easily, and use `Clone()` to avoid paying the small initialisation cost of the syringe itself more than once.

[you can use named types]: #named-instances
[use multiple syringes]: #scopes

#### Named Instances

Sometimes, you may need to inject more than one value of the same type. For example, the following struct needs 2 strings, `Name` and `Desc`:

```go
type Something struct { Name, Desc string }
```

As it stands, syringe would be unable to inject `Name` and `Desc` with different values, since a syringe can only inject a single value of each type, and they are both `string`. However, by using an under-used feature of Go, [named types], it is possible to inject different values:

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

If you need values with different scopes (a.k.a. lifetimes), then you can use multiple syringes, one for each scope. This allows you to preciesly control value lifetimes using normal Go code. There is one method added to support this use case: `Clone()`. The main use of `Clone` is to generate a fresh syringe based on the constructors and values of one you've already defined. This is computationally cheaper than filling a blank syringe from scratch. See this example HTTP server below:

```go
var appScopedSyringe, requestScopedSyringe syringe.Syringe

func main() {
	appScopedSyringe = syringe.New().Fill(ApplicationScopedThings...)
	requestSyringe = syringe.New().Fill(RequestScopedThings...)
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
		fmt.Fprintf(w, "Error injecting app-scoped values: %s", err)
		return
	}
	// Then inject request-scoped things... Later injections beat earlier
	// ones, in case both syringes inject the same type.
	// Note the use of Clone() here. That is important, as once you call
	// Inject on a syringe, it uses up all the invoked constructors, and
	// replaces them with their constructed values. Clone() creates a
	// bytewise copy of the syringe value at this point, copying all
	// values that it has realised so far, as well as any constructors
	// that are still needed to construct as-yet unrealised values.
	if err := requestSyringe.Clone().Inject(&controller); err != nil {
		w.WriterHeader(500)
		fmt.Fprintf(w, "Error injecting request-scoped values: %s", err)
		return
	}
	controller.HandleRequest(w, r)
}
```

### How does it work?

Each item you pass into `.Fill()` is analysed to see whether or not it is a [constructor]. If it is a constructor, then the type of its first return value is registered as its [injection type]. Otherwise the item is considered to be a _value_ and its own type is used as its injection type. Your syringe knows how to inject values of each registered injection type.

When you call `.Inject(&someStruct)`, each field in someStruct is populated with an item of the corresponding injection type from the syringe. For constructors, it will call that constructor exactly once to generate its value, if needed. For non-constructor values that were passed in to `Fill`, it will simply inject that value when called to.

Each parameter in a constructor will need to also be available in the syringe, in order for that constructor to be successfully invoked. If not, `.Inject` will return an error.

Likewise, if the constructor is successfully invoked, but returns a non-nil error as its second return value, then `.Inject` will return the first such error encountered.

[injection type]: #injection-types
[constructor]: #constructors


#### Injection Types

Values and constructors passed into a syringe have an implicit **_injection type_** which is the type of value that thing represents. For non-constructor values, the injection type is the type of the value passed into the syringe. For constructors, it is the type of the first output (return) value. It is important to understand this concept, since a single syringe can have only one value or constructor per injection type. `Fill` will return an error if you try to register multiple values and/or constructors that resolve to the same injection type.

#### Constructors

Constructors can take 2 different forms:

1. `func (...Anything) Anything`
2. `func (...Anything) (Anything, error)`

Just to clarify: `Anything` means literally any type, and in the signatures above can have a different value each time it is seen. For example, all of the following types are considered to be constructors:

- func() int
- func() (int, error)
- func(int) int
- func(int) (int, error) 
- func (string, io.Reader, io.Writer) interface{}
- func (string, io.Reader, io.Writer) (interface{}, error)

If you need to inject a function which has a constructor's signature, you'll need to create a constructor that returns that function. For example, for an value with injection type `func(int) (int, error)`, you would need to create a func to return that:

```go
newFunc() func(int) (int, error) {
	return func(int) (int, error) { return 0, nil }
}
```

