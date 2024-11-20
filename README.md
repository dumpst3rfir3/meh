# My Endpoint Helper (MEH)

<img src="img/meh_logo.jpg" alt="meh" width="200">

Meh...this is just a module that provides some helper functions that I have used often when writing custom [Prelude Detect](https://docs.preludesecurity.com/docs/the-basics) tests. Ho hum. Just import it and use the functions.

See [sample.go](examples/sample.go) for an example that imports the module and uses some of its functions.

Some of the functions include:
* `CleanFiles` - delete files written to disk as part of the test
* `CheckQuarantine` - check if a file was quarantine and, based on expectation, stop the test appropriately
* `CopyFile` - copy a file from source to destination
* `Download` - download a file from a specified URL, optionally check the MD5 hash and whether it was blocked by a proxy
* `EggHunt` - look for a specific sequence of bytes, e.g., within a file, and return the offset(s) where it was found
* `Patch` - Use egghunt to find a sequence of bytes, then replace it wherever found
* `StartHTTPFileServer` - start an HTTP file server on localhost
* ` StartTCPListener` - start a TCP listener on localhost to receive callbacks
* `MinimizeWindow` - minimize any window in Windows (if the test opens any)
* `RegDelete` - delete a registry key (and all subkeys) in Windows
* `TaskKillName` - kill a process in Windows by name
* `TaskKillPID` - kill a process in Windows by PID