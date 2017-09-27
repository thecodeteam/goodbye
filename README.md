# Goodbye
Goodbye is a Golang library that provides a standard way to execute code
when a process exits normally or due to a signal.

## Overview
The Goodbye library uses a `sync.Once` to ensure that the registered exit
handlers are executed only once -- whether as a result of the process exiting
normally or as a result of a received signal.

## Install
Say hello to goodbye with `go get github.com/thecodeteam/goodbye`.

## Example
There is an [`example`](./example/example.go) program that illustrates
how to use the Goodbye library:

```golang
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/thecodeteam/goodbye"
)

func main() {
	// Create a context to use with the Goodbye library's functions.
	ctx := context.Background()

	// Always defer `goodbye.Exit` as early as possible since it is
	// safe to execute no matter what.
	defer goodbye.Exit(ctx, -1)

	// Invoke `goodbye.Notify` to begin trapping signals this process
	// might receive. The Notify function can specify which signals are
	// trapped, but if none are specified then a default list is used.
	// The default set is platform dependent. See the files
	// "goodbye_GOOS.go" for more information.
	goodbye.Notify(ctx)

	// Register two functions that will be executed when this process
	// exits.
	goodbye.Register(func(ctx context.Context, sig os.Signal) {
		fmt.Printf("1: %[1]d: %[1]s\n", sig)
	})

	goodbye.Register(func(ctx context.Context, sig os.Signal) {
		fmt.Printf("2: %[1]d: %[1]s\n", sig)
	})

	// Register a function with a priority that is higher than the two
	// handlers above. Since the default priority is 0, a priority of -1
	// will ensure this function, registered last, is executed first.
	goodbye.RegisterWithPriority(func(ctx context.Context, sig os.Signal) {

		// Use the `goodbye.IsNormalExit` function in conjunction with
		// the signal to determine if this is a signal-based exit. If it
		// is then emit a leading newline character to place the desired
		// text on a line after the `CTRL-C` if that was used to send
		// SIGINT to the process.
		//
		// Note that the extra text is being printed inside the second,
		// registered handler. This is because handlers are executed in
		// reverse order -- the earlier a handler is registered, the later
		// it is executed.
		if !goodbye.IsNormalExit(sig) {
			fmt.Println()
		}

		fmt.Printf("0: %[1]d: %[1]s\n", sig)
	}, -1)

	if len(os.Args) < 2 {
		return
	}

	// If the program's first argument is "wait" then block until the
	// program is killed with a signal -- either from the "kill" command
	// or a CTRL-C.
	if strings.EqualFold("wait", os.Args[1]) {
		c := make(chan int)
		<-c
	}
}
```

### Example 1
This example shows how the exit handlers are invoked when the program
exits normally.

```bash
$ go run example/example.go
0: 0: nosig
1: 0: nosig
2: 0: nosig
```

Please note that the first registered exit handler was executed last.

### Example 2
This example shows how the exit handlers are executed when a signal is
received:

```bash
$ go run example/example.go wait
^C
0: 2: interrupt
1: 2: interrupt
2: 2: interrupt
```
