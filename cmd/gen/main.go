package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"log"
	"os"
	"path"
	"strings"
)

func main() {

	if len(os.Args) < 2 {
		log.Fatal("usage: gen <import-path>")
	}

	importPath := os.Args[1]

	pkg, err := newPkg(importPath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Package name: %s", pkg.Name)
	for _, ctor := range pkg.Ctors() {
		fmt.Printf("%s -> %s; deps: %s\n", ctor.FuncName,
			ctor.InjectionTypeName,
			strings.Join(ctor.InTypeStrings(), ", "))
	}

	if len(os.Args) == 2 {
		return
	}

	typ := os.Args[2]
	ctor, ok := pkg.CtorByInjectionType(typ)
	if !ok {
		log.Fatalf("Injection type %q not found.", typ)
	}
	fmt.Println()
	fmt.Printf("Get%s() {\n%s\n}", typ, ctor.GetFunc())

}

// Pkg bundles an *ast.Package with a *token.FileSet.
type Pkg struct {
	Name    string
	FileSet *token.FileSet
	Package *ast.Package
}

func newPkg(importPath string) (*Pkg, error) {
	dirPath := path.Join(os.Getenv("GOPATH"), "src", importPath)

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dirPath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	if len(pkgs) != 1 {
		return nil, fmt.Errorf("Got %d packages in %q; want 1", len(pkgs), importPath)
	}

	var pkg *ast.Package
	for _, p := range pkgs {
		pkg = p
	}
	return &Pkg{
		Name:    pkg.Name,
		FileSet: fset,
		Package: pkg,
	}, nil
}

// FuncDecls returns all the func declarations in the package.
func (p *Pkg) FuncDecls() []*ast.FuncDecl {
	var out []*ast.FuncDecl
	for _, f := range p.Package.Files {
		for _, d := range f.Decls {
			if funcDecl, ok := d.(*ast.FuncDecl); ok {
				out = append(out, funcDecl)
			}
		}
	}
	return out
}

// Ctors returns all the declared psyringe constructors in the package.
func (p *Pkg) Ctors() []*CtorCode {
	var out []*CtorCode
	for _, f := range p.FuncDecls() {
		if ctor, ok := p.makeCtorCode(f); ok {
			out = append(out, ctor)
		}

	}
	return out
}

// CtorCode is the code representing a constructor in the graph.
type CtorCode struct {
	FuncName          string
	InjectionType     ast.Expr
	InjectionTypeName string
	HasErr            bool
	Inputs            []*ast.Field
	Pkg               *Pkg
}

// InTypeStrings returns a string representation of each input type in order.
func (c *CtorCode) InTypeStrings() []string {
	out := make([]string, len(c.Inputs))
	for i, f := range c.Inputs {
		out[i] = c.Pkg.exprToString(f.Type)
	}
	return out
}

// GetFunc returns code for a new function that returns the same injection type
// as this ctor with minimal inputs, in terms of other constructors in p.
func (c *CtorCode) GetFunc() string {
	unresolvedInputs := map[string]struct{}{}
	resolvedInputs := map[string]*CtorCall{}
	for _, in := range c.InTypeStrings() {
		if _, ok := resolvedInputs[in]; ok {
			continue
		}
		unresolvedInputs[in] = struct{}{}
		if ctor, ok := c.Pkg.CtorByInjectionType(in); ok {
			resolvedInputs[in] = &CtorCall{CtorCode: ctor}
			delete(unresolvedInputs, in)
		}
	}

	out := &bytes.Buffer{}
	for typ, ctorCall := range resolvedInputs {
		fmt.Fprintf(out, "%s, err :=", typ)
		ctorCall.Fprint(out)
	}

	return out.String()
}

// CtorCall is a call of a constructor.
type CtorCall struct {
	CtorCode   *CtorCode
	ArgsByType map[string]string
}

// Fprint writes this ctor call to the writer.
func (c *CtorCall) Fprint(w io.Writer) {
	fmt.Fprint(w, c.CtorCode.FuncName)
	fmt.Fprint(w, "(")
	for _, a := range c.CtorCode.InTypeStrings() {
		fmt.Fprint(w, c.ArgsByType[a])
		fmt.Fprint(w, ",")
	}
	fmt.Fprint(w, ")")
}

// CtorByInjectionType returns the ctor for the injectiontype t and true, or nil
// and false if there isn't one.
func (p *Pkg) CtorByInjectionType(t string) (*CtorCode, bool) {
	for _, ctor := range p.Ctors() {
		if ctor.InjectionTypeName == t {
			return ctor, true
		}
	}
	return nil, false
}

func (p *Pkg) newCtorCode(name string, inputs *ast.FieldList, outType ast.Expr, hasErr bool) *CtorCode {
	return &CtorCode{
		FuncName:          name,
		InjectionType:     outType,
		InjectionTypeName: p.exprToString(outType),
		HasErr:            hasErr,
		Inputs:            inputs.List,
		Pkg:               p,
	}
}

func (p *Pkg) exprToString(expr ast.Expr) string {
	var buf bytes.Buffer
	p.Fprint(&buf, expr)
	return buf.String()
}

// Fprint writes the provided node n to the writer.
func (p *Pkg) Fprint(w io.Writer, n interface{}) {
	printer.Fprint(w, p.FileSet, n)
}

func (p *Pkg) makeCtorCode(f *ast.FuncDecl) (*CtorCode, bool) {
	if !strings.HasPrefix(f.Name.Name, "new") || f.Type.Results == nil {
		return nil, false
	}
	out := f.Type.Results.List
	numOut := len(out)
	switch {
	default:
		return nil, false
	case numOut == 1:
		return p.newCtorCode(f.Name.Name, f.Type.Params, f.Type.Results.List[0].Type, false), true
	case numOut == 2 && isBuiltinErrorType(out[1].Type):
		return p.newCtorCode(f.Name.Name, f.Type.Params, f.Type.Results.List[0].Type, true), true
	}
}

func isBuiltinErrorType(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	return id.Name == "error"
}
