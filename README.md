# editorconfig

[![GoDoc](https://godoc.org/mvdan.cc/editorconfig?status.svg)](https://godoc.org/mvdan.cc/editorconfig)

A small package to parse and use [EditorConfig][1] files. Currently passes all
of the official [test cases][2], which are run via `go test`.

Note that an [official library][3] exists for Go already. This alternative
implementation exists as its design is fairly different.

[1]: https://editorconfig.org/
[2]: https://github.com/editorconfig/editorconfig-core-test
[3]: https://github.com/editorconfig/editorconfig-core-go
