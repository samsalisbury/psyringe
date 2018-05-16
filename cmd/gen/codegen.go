package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"go/ast"

	"github.com/nyarly/rocketsurgery"
)

// LoadConstructors examines the named Go package for constructors and returns
// them.
func LoadConstructors(packageImportPath string) ([]CtorCode, error) {

	dirPath := path.Join(os.Getenv("GOPATH"), "src", packageImportPath)

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var ctorCodes []CtorCode
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".go") {
			continue
		}
		filePath := path.Join(dirPath, f.Name())
		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Panicf("failed to close file %q", filePath)
			}
		}()
		sc, err := rocketsurgery.ParseReader(filePath, file)
		if err != nil {
			return nil, err
		}

		for _, f := range sc.Funcs() {
			if f == nil || f.Type == nil || f.Type.Results == nil {
				continue
			}
			outputs := f.Type.Results.List
			if len(outputs) == 0 || len(outputs) > 2 {
				continue
			}
			if len(outputs) == 2 && !isErrorType(outputs[1].Type) {
				continue
			}
			ctorCodes = append(ctorCodes, CtorCode{
				FuncName: f.Name.String(),
			})
		}
	}

	return ctorCodes, nil
}

func isErrorType(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	return id.Name == "error"
}
