package GoHclGen

import (
	"fmt"
	"go/build"
)

type Status int

//go:generate go run "github.com/dinimicky/myenumstr" -type Status,Color
const (
	Offline Status = iota
	Online
	Disable
	Added
	Deleted
)

type Color int

const (
	Write Color = iota
	Red
	Blue
)

var (
	pkgInfo *build.Package
)

func ReadPackage() {
	pkgInfo, err := build.ImportDir(".", 0)
	if err != nil {
		panic(err)
	}
	fmt.Println("%v", pkgInfo)
}
