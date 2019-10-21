// Copyright (c) 2019, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package editorconfig

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("EDITORCONFIG_CMD") != "" {
		cmd()
		os.Exit(0)
	}
	// call flag.Parse() here if TestMain uses flags
	os.Exit(m.Run())
}

func run(dir, command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func mustRun(t *testing.T, dir, command string, args ...string) {
	out, err := run(dir, command, args...)
	if err != nil {
		fmt.Println(out)
		t.Fatal(err)
	}
}

func TestViaCmake(t *testing.T) {
	os.Setenv("EDITORCONFIG_CMD", os.Args[0])
	mustRun(t, "core-test", "cmake", "..")

	// Run with a high number of parallel jobs, as we have a lot of test
	// cases to run, and the test binary lags with -race.
	os.Setenv("GORACE", "atexit_sleep_ms=10")
	out, err := run("core-test", "ctest")
	if err != nil {
		rxFailed := regexp.MustCompile(` - ([a-zA-Z0-9_]+) \((.*)\)`)
		matches := rxFailed.FindAllStringSubmatch(out, -1)
		if len(matches) == 0 {
			// something went very wrong
			fmt.Println(out)
			t.Fatal(err)
		}

		for _, match := range matches {
			name, reason := match[1], match[2]
			t.Errorf("%s failed: %s", name, reason)
		}
	}
}
