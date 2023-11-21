package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"text/template"
)

const decodeTemplate = `
func ({{.Receiver}} *{{.StructName}}) {{.MethodName}}(data []byte) {
	l, offset, err := mapLength(0, k, data)
	if err != nil {
		return 0, err
	}
}
`

// DecodeMethodSpec defines the specifications for a new method to be added to a struct.
type DecodeMethodSpec struct {
	StructName string
	Receiver   string // Receiver is usually one or two letters from the struct name
	MethodName string
	Parameter  string
}

// generateMethod generates a method for a struct using a template.
func generateMethod(spec DecodeMethodSpec) string {
	tmpl, err := template.New("method").Parse(decodeTemplate)
	if err != nil {
		fmt.Println("Error parsing template:", err)
		return ""
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, spec); err != nil {
		fmt.Println("Error executing template:", err)
		return ""
	}

	return buf.String()
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run code_gen.go <path_to_go_file>")
		os.Exit(1)
	}

	filePath := os.Args[1]
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		fmt.Printf("Error parsing file %s: %v\n", filePath, err)
		os.Exit(1)
	}

	for _, f := range node.Decls {
		genDecl, ok := f.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			if structType.Fields.NumFields() == 0 {
				// Skip empty structs.
				continue
			}

			methodSpec := DecodeMethodSpec{
				StructName: typeSpec.Name.Name,
				Receiver:   strings.ToLower(typeSpec.Name.Name[:1]),
				MethodName: "Decode",
				Parameter:  "data []byte",
			}

			method := generateMethod(methodSpec)
			fmt.Println(method)
		}
	}
}
