package meh

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	endpoint "github.com/preludeorg/libraries/go/tests/endpoint"
	network "github.com/preludeorg/libraries/go/tests/network"
)

type fn func()

var cleanup fn = func() {}
var stopMutex *sync.Mutex = &sync.Mutex{}

// Callback detect - used to detect if a successfull callback
// was received by the TCP listener defined below
type CallbackDetect struct {
	Connected bool
	Err       error
}

// Delete files that have written to disk as part of the test
func CleanFiles(paths ...string) {
	for _, path := range paths {
		if endpoint.Exists(path) {
			removed := endpoint.Remove(path)
			if removed {
				endpoint.Say("Sucessfully removed %s", path)
			} else {
				endpoint.Say("Failed to remove %s", path)
			}
		}
	}
}

// Check if file was quarantined and, based on expectation,
// stop the test appropriately
func CheckQuarantine(
	fn string, b []byte, expectedQuarantine bool,
) {
	var ok bool = endpoint.Quarantined(fn, b)
	if !ok && expectedQuarantine {
		endpoint.Stop(endpoint.Unprotected)
	} else if ok && !expectedQuarantine {
		endpoint.Stop(endpoint.FileQuarantinedOnExtraction)
	}
}

// Copy a file from src to dst
func CopyFile(src, dst string) error {
	var err error
	var numbytes int64
	var origfile, dstfile *os.File

	origfile, err = os.Open(src)
	if err != nil {
		return err
	}
	defer origfile.Close()

	dstfile, err = os.Create(dst)
	if err != nil {
		return err
	}
	defer dstfile.Close()

	numbytes, err = io.Copy(dstfile, origfile)
	endpoint.Say("Wrote %d bytes", numbytes)

	return err
}

// Download a file from the specified URL, optionally check
// the expected MD5 hash, and optionally check if the download
// was blocked by the proxy by looking for a proxy-specific string
func Download(
	url string,
	expectedMD5 string,
	proxyblockstring string,
) []byte {
	var e error
	var req *network.Requester
	var res network.ResponseData
	var sum string
	var tmp [16]byte
	var ua string = "" +
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/115.0.0.0 Safari/537.36 " +
		"Edg/115.0.1901.188 " +
		"red team prelude test"

	endpoint.Say("Downloading file from %s", url)
	req = network.NewHTTPRequest(
		url,
		&network.RequestOptions{
			Timeout:   10 * time.Second,
			UserAgent: ua,
		},
	)
	if res, e = req.GET(network.RequestParameters{}); e != nil {
		endpoint.Say("Failed to download from %s: %s", url, e)
		return nil
	} else if res.StatusCode != 200 {
		endpoint.Say("Failed to download from %s: %s", url, res.Body)
		return nil
	}

	// Check for proxy block
	if proxyblockstring != "" {
		if bytes.Contains(res.Body, []byte(proxyblockstring)) {
			endpoint.Say("Download blocked by proxy")
			endpoint.Stop(endpoint.NetworkConnectionBlocked)
		}
	}

	// Check hash
	if expectedMD5 != "" {
		expectedMD5 = strings.ToLower(expectedMD5)
		tmp = md5.Sum(res.Body)
		sum = strings.ToLower(hex.EncodeToString(tmp[:]))
		if sum != expectedMD5 {
			endpoint.Say(
				"MD5 sum (%s) did not match expected value (%s)",
				sum,
				expectedMD5,
			)
			endpoint.Stop(endpoint.UnexpectedTestError)
		}
	}

	endpoint.Say("Successfully downloaded")
	return res.Body
}

// Look for a specific sequence of bytes, e.g., in a file
// and return the byte offset(s) where it was found
func EggHunt(b []byte, lookFor string) []int {
	var found []int
	var size int = len(lookFor) / 2

	for i := 0; i < len(b)-size; i++ {
		if fmt.Sprintf("%0x", b[i:i+size]) == lookFor {
			found = append(found, i)
		}
	}

	return found
}

// Use egghunt to find a sequence of bytes and then replace it
// wherever found
func Patch(b []byte, egg string, offset int, new string) []byte {
	var val []byte

	val, _ = hex.DecodeString(new)

	for _, idx := range EggHunt(b, egg) {
		endpoint.Say("Found %s at %d", egg, idx)

		for i := range val {
			b[idx+offset+i] = val[i]
		}

		endpoint.Say("Patched to %0x", b[idx:idx+len(egg)/2])
	}

	return b
}

// Run will attempt to run the provided command and args as a new
// process. It returns the new process handle and any error that
// occurs. The caller should decide whether to call Kill() or Wait()
// on the returned process handle. This is in case you don't want
// to use Prelude's Endpoint.Shell, which will wait until the
// command completes and returns command output
func Run(args []string) (*os.Process, error) {
	endpoint.Say(
		"Running \"%s\" in the background",
		strings.Join(args, " "),
	)
	cmd := exec.Command(args[0], args[1:]...)
	if attrs := procAttrs(args); attrs != nil {
		cmd.SysProcAttr = attrs
	}
	if err := cmd.Start(); err != nil {
		err = fmt.Errorf("failed to run cmd: %w", err)
		return nil, err
	}
	return cmd.Process, nil
}

// Start an HTTP file server on localhost that can be used to
// download files via http
// Example usage below:
//
//	var stop chan struct{} = StartHTTPFileServer(8080)
//
//	// do stuff
//
//	stop <- struct{} // Tell server to shutdown
//	<- stop // Wait for shutdown to complete
func StartHTTPFileServer(lport int, dir ...string) chan struct{} {
	var e error
	var srv *http.Server
	var stop chan struct{} = make(chan struct{}, 1)

	if len(dir) == 0 {
		cwd, e := os.Getwd()
		if e != nil {
			dir = append(dir, cwd)
		} else {
			endpoint.Stop(endpoint.UnexpectedTestError)
		}
	}

	srv = &http.Server{
		Addr:    fmt.Sprintf(":%d", lport),
		Handler: http.FileServer(http.Dir(dir[0])),
	}

	// Start file server in go routine
	go func() {
		endpoint.Say("Starting HTTP file server on port %d", lport)
		if e = srv.ListenAndServe(); e != nil {
			if e != http.ErrServerClosed {
				endpoint.Say("HTTP server error: %s", e)
			}
		}
	}()

	// Start go routine that waits for caller to send stop signal
	go func() {
		<-stop

		endpoint.Say("Shutting down HTTP file server...")
		srv.Close()
		endpoint.Say("Success")

		stop <- struct{}{}
		close(stop)
	}()

	// Return chan so caller can signal stop
	return stop
}

// Start a TCP listener on localhost that can receive reverse
// shells. If it receives any TCP, it assumes it was a successful
// execution of the reverse shell. This can be used if, for example,
// using a simple reverse shell meterpreter payload that reaches
// out to localhost
// Example usage below:
//
//	var cbd CallbackDetect
//	var r chan CallbackDetect = StartTCPListener(4444, 5 * time.Second)
//
//	cbd = <- r
//
//	if cbd.Connected {
//	    fmt.Println("yes")
//	} else {
//	    fmt.Println("no")
//	    if cbd.Err != nil {
//	        fmt.Printf("because %s\n", cbd.Err)
//	    }
//	}
func StartTCPListener(lport int, to time.Duration) chan CallbackDetect {
	var rcvd chan CallbackDetect = make(chan CallbackDetect, 1)

	go func() {
		var a *net.TCPAddr
		var c *net.TCPConn
		var cbd CallbackDetect
		var e error
		var l *net.TCPListener

		defer close(rcvd)

		a, e = net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", lport))
		if e != nil {
			endpoint.Say("TCP resolution error: %s", e)
			cbd.Err = e
			rcvd <- cbd
			return
		}

		endpoint.Say("Starting TCP listener on port %d", lport)
		if l, e = net.ListenTCP("tcp", a); e != nil {
			endpoint.Say("TCP listener error: %s", e)
			cbd.Err = e
			rcvd <- cbd
			return
		}
		defer l.Close()

		endpoint.Say("Waiting %s for TCP connection", to)
		l.SetDeadline(time.Now().Add(to))

		if c, e = l.AcceptTCP(); e != nil {
			endpoint.Say("Error receiving connection %s", e)
			cbd.Err = e
			rcvd <- cbd
			return
		}
		defer c.Close()

		endpoint.Say("Received a TCP connection: %s", c.RemoteAddr())
		cbd.Connected = true
		rcvd <- cbd
	}()

	return rcvd
}

// This is exactly the same as Prelude Endpoint's Start function,
// only you can pass it a custom timeout in case your test needs
// more than 30 seconds (which is the endpoint timeout)
func StartWithCustomTimeout(
	test fn,
	timeout time.Duration,
	clean ...fn,
) {
	if len(clean) > 0 {
		cleanup = clean[0]
	}

	endpoint.Say(
		"Starting test at: %s",
		time.Now().Format("2006-01-02T15:04:05"),
	)

	go func() {
		test()
	}()

	time.Sleep(timeout)
	Stop(endpoint.TimeoutExceeded)

}

// Prelude's Endpoint.Stop function allows for a race condition
// where, for example, a background thread might try to call
// Endpoint.Stop at the same time as the main thread. This
// version of Stop just calls Endpoint.Stop, but locks a mutex
// first so that only one thread can call it
func Stop(code int) {
	// Only allow one stop
	stopMutex.Lock()
	defer stopMutex.Unlock()

	cleanup()

	endpoint.Stop(code)
}
