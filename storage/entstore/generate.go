//go:build ignore

package main

import (
	"log"
	"path/filepath"
	"runtime"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
)

func main() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("failed to determine generate.go location")
	}
	baseDir := filepath.Dir(filename)

	if err := entc.Generate(
		filepath.Join(baseDir, "schema"),
		&gen.Config{
			Target:  filepath.Join(baseDir, "ent"),
			Package: "github.com/mwangaben/permission/storage/entstore/ent",
		},
	); err != nil {
		log.Fatalf("running ent codegen: %v", err)
	}
	log.Println("ent codegen complete")
}
