// Copyright (c) 2019, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package editorconfig

import (
	"flag"
	"fmt"
	"log"
)

func cmd() {
	var (
		configName     = flag.String("f", DefaultName, "")
		emulateVersion = flag.String("b", "", "")
		version        = flag.Bool("v", false, "")
		versionLong    = flag.Bool("version", false, "")
	)
	flag.Parse()
	if *version || *versionLong {
		fmt.Printf("EditorConfig Go mvdan.cc/editorconfig, Version 0.0.0-devel\n")
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
	}

	query := Query{
		ConfigName: *configName,
		Version:    *emulateVersion,
	}
	for _, arg := range args {
		result, err := query.Find(arg, nil)
		if err != nil {
			log.Fatal(err)
		}
		if len(args) > 1 {
			result.Name = arg
		}
		fmt.Printf("%s", result)
	}
}
