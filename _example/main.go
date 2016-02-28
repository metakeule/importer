package main

import (
	"fmt"
	"github.com/metakeule/importer"
	"go/ast"
	"go/build"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
)

func mkInfo() types.Info {
	return types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
}

func mkConfig() types.Config {
	return types.Config{
		IgnoreFuncBodies: true,
	}
}

func panicOnErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func main() {
	var (
		pkgpath = "github.com/metakeule/importer"

		// assumes a single GOPATH directory
		srcDir = filepath.Join(os.Getenv("GOPATH"), "src", pkgpath)

		fset     = token.NewFileSet()
		astFiles []*ast.File
	)

	buildPkg, err := build.Import(pkgpath, srcDir, 0)
	panicOnErr(err)

	astFiles, err = importer.ParseAstFiles(fset, buildPkg.Dir, append(buildPkg.GoFiles, buildPkg.CgoFiles...))

	panicOnErr(err)

	info, config := mkInfo(), mkConfig()
	// here the importer is used with the same config and info settings
	config.Importer = importer.CheckImporter(mkInfo, mkConfig)
	_, err = (&config).Check(buildPkg.Name, fset, astFiles, &info)

	panicOnErr(err)

	for ident, obj := range info.Defs {
		if ident.IsExported() {
			fmt.Printf("%s:\n    %s\n", ident.Name, obj.Type())
		}
	}
}
