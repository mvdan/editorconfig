// Copyright (c) 2019, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package editorconfig_test

import (
	"fmt"
	"strings"

	"mvdan.cc/editorconfig"
)

func Example() {
	props, err := editorconfig.Find("_sample/subdir/code.go")
	if err != nil {
		panic(err)
	}
	fmt.Println(props)

	fmt.Println(props.Get("indent_style"))
	fmt.Println(props.IndentSize())
	fmt.Println(props.TrimTrailingWhitespace())
	fmt.Println(props.InsertFinalNewline())

	// Output:
	// indent_style=tab
	// indent_size=8
	// end_of_line=lf
	// insert_final_newline=true
	//
	// tab
	// 8
	// false
	// true
}

func ExampleParse() {
	config := `
root = true

[*] # match all files
end_of_line = lf
insert_final_newline = true

[*.go] # only match Go
indent_style = tab
indent_size = 8
`
	file, err := editorconfig.Parse(strings.NewReader(config))
	if err != nil {
		panic(err)
	}
	fmt.Println(file)

	// Output:
	// root=true
	//
	// [*]
	// end_of_line=lf
	// insert_final_newline=true
	//
	// [*.go]
	// indent_style=tab
	// indent_size=8
}

func ExampleFile_Filter() {
	config := `
[*]
end_of_line = lf

[*.go]
indent_style = tab
`
	file, err := editorconfig.Parse(strings.NewReader(config))
	if err != nil {
		panic(err)
	}
	fmt.Println(file.Filter("main.go"))

	// Output:
	// indent_style=tab
	// end_of_line=lf
}
