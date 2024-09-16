package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
	"os"
)

func exprType(expr ast.Expr, pkg *packages.Package) types.Type {
	switch x := expr.(type) {
	case *ast.Ident:
		if obj := pkg.TypesInfo.Defs[x]; obj != nil {
			return obj.Type()
		}
		if obj := pkg.TypesInfo.Uses[x]; obj != nil {
			return obj.Type()
		}
		if x.Obj != nil && x.Obj.Kind == ast.Typ {
			if typeSpec, ok := x.Obj.Decl.(*ast.TypeSpec); ok {
				return exprType(typeSpec.Type, pkg)
			}
		}
		if basicType := types.Universe.Lookup(x.Name); basicType != nil {
			return basicType.Type()
		}

	case *ast.ArrayType:
		if elemType := exprType(x.Elt, pkg); elemType != nil {
			return types.NewSlice(elemType)
		}
	case *ast.StarExpr:
		if baseType := exprType(x.X, pkg); baseType != nil {
			return types.NewPointer(baseType)
		}
	case *ast.StructType:
		fields := make([]*types.Var, 0)
		for _, field := range x.Fields.List {
			fieldType := exprType(field.Type, pkg)
			if fieldType != nil {
				for _, fieldName := range field.Names {
					fields = append(fields, types.NewVar(0, nil, fieldName.Name, fieldType))
				}
			}
		}
		return types.NewStruct(fields, nil)
	}
	return nil
}

func typeStateSpace(typ types.Type) int {
	switch t := typ.(type) {
	case *types.Basic: // базовые типы
		switch t.Kind() {
		case types.Bool:
			return 2
		case types.Int:
			return 10 // по идее int(math.Pow(2, 32))
		case types.Uint64:
			return 10 // по идее int(math.Pow(2, 64))
		case types.String:
			return 10 // по идее много
		default:
			fmt.Printf("не указан тип: %v", t.Kind())
		}
	case *types.Slice:
		elemSpace := typeStateSpace(t.Elem())
		return elemSpace * 100 // хз сколько ставить
	case *types.Pointer:
		return typeStateSpace(t.Elem())
	case *types.Struct: // структура = произведение входящих простых типов
		stateSpace := 1
		for i := 0; i < t.NumFields(); i++ {
			fieldType := t.Field(i).Type()
			stateSpace *= typeStateSpace(fieldType)
		}
		return stateSpace
	}
	return 1
}

// analyzeInputStateSpace считает пространство состояний входных данных функции
func analyzeInputStateSpace(f *ast.FuncDecl, pkg *packages.Package) int {
	stateSpace := 1

	if f.Type.Params != nil {
		for _, param := range f.Type.Params.List {
			typ := exprType(param.Type, pkg)
			if typ != nil {
				stateSpace *= typeStateSpace(typ)
			}
		}
	}

	return stateSpace
}

// analyzeOutputStateSpace считает пространство состояний выходных данных функции
func analyzeOutputStateSpace(f *ast.FuncDecl, pkg *packages.Package) int {
	stateSpace := 1

	if f.Type.Results != nil {
		for _, result := range f.Type.Results.List {
			typ := exprType(result.Type, pkg)
			if typ != nil {
				stateSpace *= typeStateSpace(typ)
			}
		}
	}

	return stateSpace
}

// analyzeBranching считает кол-во ветвлений
func analyzeBranching(f *ast.FuncDecl) int {
	branchingFactor := 0
	ast.Inspect(f.Body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt, *ast.SwitchStmt, *ast.ForStmt, *ast.RangeStmt:
			branchingFactor++
		}
		return true
	})
	return branchingFactor
}

// analyzeWTFComplexity считает кол-во вызовов, операций и присваиваний
func analyzeWTFComplexity(f *ast.FuncDecl) int {
	complexity := 0
	ast.Inspect(f.Body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.BinaryExpr, *ast.CallExpr, *ast.AssignStmt:
			complexity++
		}
		return true
	})
	return complexity
}

// countLocalAssignment считает кол-во локальных переменных внутри функции
func countLocalAssignment(f *ast.FuncDecl) int {
	localVars := 0
	ast.Inspect(f.Body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.AssignStmt:
			localVars++
		}
		return true
	})
	return localVars
}

func main() {
	var path string
	flag.StringVar(&path, "path", "/Users/a/Documents/github/complex_lint/example.go", "путь до go файла для анализа")
	flag.Parse()

	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("open path ", err)
		return
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, src, parser.AllErrors)
	if err != nil {
		fmt.Println(err)
		return
	}

	cfg := &packages.Config{Mode: packages.LoadSyntax | packages.NeedDeps}
	pkgs, err := packages.Load(cfg, "main")
	if err != nil {
		fmt.Println(err)
		return
	}
	if len(pkgs) == 0 {
		fmt.Println("No packages loaded")
		return
	}
	pkg := pkgs[0]

	for _, f := range node.Decls {
		if fn, ok := f.(*ast.FuncDecl); ok {
			inputStateSpace := analyzeInputStateSpace(fn, pkg)
			outputStateSpace := analyzeOutputStateSpace(fn, pkg)
			branchingFactor := analyzeBranching(fn)
			wtfComplexity := analyzeWTFComplexity(fn)
			localAssignment := countLocalAssignment(fn)

			fmt.Printf("\nFunction \"%s\" analysis:\n", fn.Name.Name)
			fmt.Printf(" - input state space: %d\n", inputStateSpace)
			fmt.Printf(" - output state space: %d\n", outputStateSpace)
			fmt.Printf(" - branching factor: %d\n", branchingFactor)
			fmt.Printf(" - operational complexity: %d\n", wtfComplexity)
			fmt.Printf(" - local assignment: %d\n", localAssignment)
		}
	}
}
