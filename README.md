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
* `Patch` - use egghunt to find a sequence of bytes, then replace it wherever found
* `Run` - executes a command, but unlike Prelude's `Endpoint.Shell`, this will run the command in the background, and returns a process handle (rather than the command output) - NOTE: you should kill the process when it's done, see [sample.go](examples/sample.go) for an example
* `StartHTTPFileServer` - start an HTTP file server on localhost
* `StartTCPListener` - start a TCP listener on localhost to receive callbacks
* `StartWithCustomTimeout` - is exactly the same as Prelude's Endpoint.Start, but allows you to pass a custom timeout so that you're not stuck with the 30 second timeout
* `MinimizeWindow` - minimize any window in Windows (if the test opens any)
* `RegDelete` - delete a registry key (and all subkeys) in Windows
* `TaskKillName` - kill a process in Windows by name
* `TaskKillPID` - kill a process in Windows by PID

## Credit

Shout out to [mjwhitta](https://github.com/mjwhitta) for all of the help and contributions (mainly allowing me to steal some of his code).