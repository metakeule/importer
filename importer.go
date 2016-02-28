package importer

import (
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
)

type imp struct {
	checkfn func(name string, fset *token.FileSet, astFiles []*ast.File) (*types.Package, error)
	cache   map[string]*types.Package
}

// Import implements the go/types.ImporterFrom interface
// copied from https://github.com/golang/go/blob/master/src/go/importer/importer.go (m gcimports) Import
func (i imp) Import(path string) (*types.Package, error) {
	return i.ImportFrom(path, "" /* no vendoring */, 0)
}

// ImportFrom implements the go/types.ImporterFrom interface
func (i imp) ImportFrom(pkgpath, srcDir string, mode types.ImportMode) (*types.Package, error) {
	pkg, has := i.cache[pkgpath]
	if has {
		return pkg, nil
	}

	if pkgpath == "runtime" {
		pkg, err := importer.Default().(types.ImporterFrom).ImportFrom(pkgpath, srcDir, mode)
		i.cache[pkgpath] = pkg // cache even if pkg == nil and err != nil to prevent never ending loops
		return pkg, err
	}

	buildPkg, err := build.Import(pkgpath, srcDir, build.AllowBinary)
	if err != nil {
		return nil, err
	}

	var (
		fset     = token.NewFileSet()
		astFiles []*ast.File
	)

	astFiles, err = ParseAstFiles(fset, buildPkg.Dir, append(buildPkg.GoFiles, buildPkg.CgoFiles...))
	if err != nil {
		return nil, err
	}

	pkg, err = i.checkfn(buildPkg.ImportPath, fset, astFiles)
	i.cache[pkgpath] = pkg // cache even if pkg == nil and err != nil to prevent never ending loops
	return pkg, err
}

// CheckImporter creates an go/types.ImporterFrom that can be used with go/types.Config.Check
// to resolv imports that might not be installed (fixes https://github.com/golang/go/issues/14496)
// It uses the given infogen and configgen callbacks to generate the desired types.Info and types.Config
// that are used with config.Check which is called to return the types.Package.
func CheckImporter(infogen func() types.Info, configgen func() types.Config) types.ImporterFrom {
	i := &imp{
		cache: make(map[string]*types.Package),
	}

	i.checkfn = func(name string, fset *token.FileSet, astFiles []*ast.File) (*types.Package, error) {
		info := infogen()
		config := configgen()
		config.Importer = i
		return (&config).Check(name, fset, astFiles, &info)
	}
	return i
}

func parseFile(fset *token.FileSet, name string) (*ast.File, error) {
	return parser.ParseFile(fset, name, nil, parser.AllErrors)
}

// ParseAstFiles is a shortcut to parse files from a directory into a set of ast.Files.
func ParseAstFiles(fset *token.FileSet, dir string, files []string) (astFiles []*ast.File, err error) {
	for _, filename := range files {
		var afile *ast.File
		afile, err = parseFile(fset, filepath.Join(dir, filename))
		if err != nil {
			return
		}
		astFiles = append(astFiles, afile)
	}
	return
}
