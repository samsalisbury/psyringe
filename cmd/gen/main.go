package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
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

	for _, f := range pkg.Package.Files {
		for _, d := range f.Decls {
			funcDecl, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			ctor, ok := pkg.makeCtorCode(funcDecl)
			if !ok {
				continue
			}

			fmt.Printf("%s -> %s\n", ctor.FuncName, ctor.InjectionTypeName)
		}
	}
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

// CtorCode is the code representing a constructor in the graph.
type CtorCode struct {
	FuncName          string
	InjectionType     ast.Expr
	InjectionTypeName string
	HasErr            bool
	Inputs            []*ast.Field
}

func (p *Pkg) newCtorCode(name string, inputs *ast.FieldList, outType ast.Expr, hasErr bool) *CtorCode {
	return &CtorCode{
		FuncName:          name,
		InjectionType:     outType,
		InjectionTypeName: p.exprToString(outType),
		HasErr:            hasErr,
		Inputs:            inputs.List,
	}
}

func (p *Pkg) exprToString(expr ast.Expr) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, p.FileSet, expr)
	return buf.String()
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
