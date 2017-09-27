// +build go1.8

package goodbye

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"sync"
)

// ExitHandler is a function that is registerd with the "Register" function
// and is invoked when this process exits, either normally or due to a process
// signal.
type ExitHandler func(ctx context.Context, s os.Signal)

type noSig struct {
}

func (s noSig) String() string {
	return "nosig"
}
func (s noSig) Signal() {
}
func (s noSig) Format(f fmt.State, c rune) {
	val := s.String()
	if c == 'd' {
		val = "0"
	}
	f.Write([]byte(val))
}

var (
	// ExitCode is the exit code used by the Exit function if it is called
	// with an exit code value of -1.
	ExitCode int

	// handlers is a list of exit handlers to invoke when the process exits
	// or receives a signal that causes an exit behavior
	handlers    = map[int][]ExitHandler{}
	handlersRWL sync.RWMutex

	// noSigVal is provided to the handleOnce function when Exit is invoked
	// so that exit handlers can use the IsNormalExit function to determine
	// if the process is exiting normally or due to a process signal.
	noSigVal noSig

	// defaultSignals is the default list of signals and exit codes to use
	// if the Notify function is invoked with an empty signals argument value
	defaultSignals map[os.Signal]int

	// notified is the list of signals that are trapped as a result of the
	// Notify function. This list is what the Reset function uses when undoing
	// the effects of the Notify function.
	notified []os.Signal

	// once is used by the handleOnce function to execute the exit handlers
	// and os.Exit exactly once, regardless of how many times Exit is invoked
	// or if a signal is received at the same time that Exit is invoked
	once sync.Once

	// lock is used to prevent the Exit, Notify, and Reset functions
	// from being called concurrently.
	lock sync.Mutex
)

// Register registers a function to be invoked when this process exits
// normally or due to a process signal.
//
// Handlers registered with this function are given a priority of 0.
func Register(f ExitHandler) {
	RegisterWithPriority(f, 0)
}

// RegisterWithPriority registers a function to be invoked when
// this process exits normally or due to a process signal.
//
// The priority determines when an exit handler is executed. Handlers
// with a lower integer value execute first and higher integer values
// execute later. If multiple handlers share the same priority level
// then the handlers are invoked in the order in which they were
// registered.
func RegisterWithPriority(f ExitHandler, priority int) {
	handlersRWL.Lock()
	defer handlersRWL.Unlock()
	if a, ok := handlers[priority]; !ok {
		handlers[priority] = []ExitHandler{f}
	} else {
		handlers[priority] = append(a, f)
	}
}

// IsNormalExit returns true if the program is exiting as a result of
// the Exit function being invoked versus a process signal.
func IsNormalExit(sig os.Signal) bool {
	return sig == noSigVal
}

// Exit executes all of the registered exit handlers.
//
// The handlers may use the IsNormalExit function and the signal provided
// to the handler to check if the program is exiting normally or due to
// a process signal.
func Exit(ctx context.Context, exitCode int) {
	lock.Lock()
	defer lock.Unlock()
	if exitCode < 0 {
		exitCode = ExitCode
	}
	handleOnce(ctx, noSigVal, exitCode)
}

// Notify begins trapping the specified signals. This function should be
// invoked as early as possible by the executing program.
//
// The signals argument accepts a series of os.Signal values. Any os.Signal
// value in the list may be succeeded with an integer to be used as the
// process's exit code when the associated signal is received. By default
// the process will exit with an exit code of zero, indicating a graceful
// shutdown.
//
// The default value for the signals variadic depends on the operating
// system (OS):
//
//   UNIX
//     SIGKILL, 1, SIGHUP, 0, SIGINT, 0, SIGQUIT, 0, SIGTERM, 0
//
//   Windows
//     SIGKILL, 1, SIGHUP, 0, os.Interrupt, 0, SIGQUIT, 0, SIGTERM, 0
func Notify(ctx context.Context, signals ...interface{}) {
	lock.Lock()
	defer lock.Unlock()

	var sigs map[os.Signal]int
	if len(signals) == 0 {
		sigs = defaultSignals
	} else {
		sigs = map[os.Signal]int{}
		var s os.Signal
		for _, v := range signals {
			switch tv := v.(type) {
			case os.Signal:
				s = tv
				sigs[s] = 0
			case int:
				sigs[s] = tv
			}
		}
	}

	var (
		i        = 0
		sigc     = make(chan os.Signal, 1)
		notified = make([]os.Signal, len(sigs))
	)
	for s := range sigs {
		notified[i] = s
		i++
	}

	signal.Notify(sigc, notified...)

	go func() {
		for s := range sigc {

			// Get the exit code associated with the signal. If no
			// exit code exists then the signal was not trapped and
			// should not be handled.
			x, ok := sigs[s]
			if !ok {
				continue
			}

			// Execute the signal handlers and exit the program.
			handleOnce(ctx, s, x)
		}
	}()
}

// Reset clears the list of registered exit handlers and stops trapping
// the signals that were trapped as a result of the Notify function.
func Reset() {
	lock.Lock()
	defer lock.Unlock()
	signal.Reset(notified...)

	handlersRWL.Lock()
	defer handlersRWL.Unlock()
	handlers = nil
}

func handleOnce(ctx context.Context, s os.Signal, x int) {
	once.Do(func() {
		handle(ctx, s)
		os.Exit(x)
	})
}

func handle(ctx context.Context, s os.Signal) {
	handlersRWL.RLock()
	defer handlersRWL.RUnlock()

	keys := []int{}
	for k := range handlers {
		keys = append(keys, k)
	}

	sort.Ints(keys)

	for _, k := range keys {
		for _, h := range handlers[k] {
			h(ctx, s)
		}
	}
}
