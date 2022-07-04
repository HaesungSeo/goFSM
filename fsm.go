package goFSM

import (
	"fmt"
	"reflect"
	"runtime"
	"time"

	fsmerror "github.com/HaesungSeo/goFSM/internal/fsmerrors"
)

type State struct {
	State string
}

type Event struct {
	Event string
}

type TrnasitLog struct {
	time    time.Time // time event occurs
	state   string    // current State
	event   string    // Event
	handle  string    // Handle
	success bool      // Handle result
	next    string    // next event determined by Handler
	msg     string    // Messages related for this fsm event
	err     error     // Error, from handle
}

// FSM Entry
type FSMEntry struct {
	Owner  interface{}   // Owner Entry
	Ctrl   *FSMCTL       // FSM Rule for this Entry
	State  State         // Current State
	Logs   []*TrnasitLog // transition log, for debug
	LogMax int
}

type FsmCallback func(Owner interface{}, event Event) (State, error)

type FsmHandle struct {
	Default bool        // is default handler
	Name    string      // handle name
	Handle  FsmCallback // handle function
	Cands   []State     // valid next state candidates
}

type FSMCTL struct {
	InitState State
	LogMax    int

	// Valid States
	States map[State]struct{}

	// Valid Events
	Events map[Event]struct{}

	// Handles indexted by State,Event
	Handles map[State]map[Event]FsmHandle
}

// FSM Action Description Table
type FSMDescEvent struct {
	Event      string      // Event
	Handle     FsmCallback // Handler for this {State, Event}
	Candidates []string    // valid next state candidates,
	// if nil, handler MUST PROVIDE next state
}

type EventDesc []FSMDescEvent

type FSMDescState struct {
	State  string
	Events EventDesc
}
type StateDesc []FSMDescState

// FSM State-Event Descriptor
type FSMDesc struct {
	InitState string // Initial State for FSMEntry
	LogMax    int    // maximum lengh of log
	States    StateDesc
}

func getFunctionName(i interface{}) string {
	funcName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	// iIdx := strings.Index(funcName, "Holder.") + 7
	// jIdx := strings.Index(funcName, "-fm")
	// funcName = funcName[iIdx:jIdx]
	return funcName
}

type StateEventConflictError struct {
	State     string // current state
	Event     string // input event
	OldHandle string // current handle
	NewHandle string // overwritting handle
	Err       error
}

func (e *StateEventConflictError) Error() string {
	return "State:" + e.State + ", Event:" + e.Event + " Old Handle " +
		e.OldHandle + " New Handle " + e.NewHandle + ": " +
		e.Err.Error()
}

func (e *StateEventConflictError) Unwrap() error { return e.Err }

// Create New FSM Control
// d FSM Descritor
func New(d FSMDesc) (*FSMCTL, error) {
	newFsm := FSMCTL{}

	newFsm.States = make(map[State]struct{})
	newFsm.Events = make(map[Event]struct{})
	newFsm.Handles = make(map[State]map[Event]FsmHandle)

	newFsm.InitState = State{d.InitState}
	newFsm.LogMax = d.LogMax

	// Initialize given states, events
	for _, state := range d.States {
		// Index State
		newFsm.States[State{state.State}] = struct{}{}

		for _, event := range state.Events {
			// Index Events
			newFsm.Events[Event{event.Event}] = struct{}{}

			// Index NextState
			for _, nstate := range event.Candidates {
				newFsm.States[State{nstate}] = struct{}{}
			}
		}
	}

	// Allocate Handles
	for _, state := range d.States {
		newFsm.Handles[State{state.State}] = make(map[Event]FsmHandle)
	}

	// Add User defined State-Event-Handles
	for _, state := range d.States {
		for _, event := range state.Events {
			hName := getFunctionName(event.Handle)
			handle := FsmHandle{
				false,
				hName,
				event.Handle,
				make([]State, 0),
			}
			for _, nstate := range event.Candidates {
				handle.Cands = append(handle.Cands, State{nstate})
			}

			s, statefound := newFsm.Handles[State{state.State}]
			if statefound {
				old, handlefound := s[Event{event.Event}]
				if handlefound {
					if &old.Handle != &handle.Handle {
						// state-event table MUST HAVE only one handle per entry
						return nil, &StateEventConflictError{
							State:     state.State,
							Event:     event.Event,
							OldHandle: old.Name,
							NewHandle: hName,
							Err:       fsmerror.ErrHandle,
						}
					}
				}
			}

			// Add handle
			newFsm.Handles[State{state.State}][Event{event.Event}] = handle
		}
	}

	return &newFsm, nil
}

// Dump Handlers
func (f *FSMCTL) DumpTable() {
	fmt.Printf("InitState[%s]\n", f.InitState)

	fmt.Printf("All States\n")
	for state, _ := range f.States {
		fmt.Printf("  [%s]\n", state)
	}

	fmt.Printf("All Events\n")
	for event, _ := range f.Events {
		fmt.Printf("  [%s]\n", event)
	}

	for state, events := range f.Handles {
		fmt.Printf("State[%s]\n", state)
		for event, handle := range events {
			fmt.Printf("  Event[%s] Func[%s] NextState[%s]\n", event, handle.Name, handle.Cands)
		}
	}
}

// Do FSM
func (f *FSMCTL) NewEntry(owner interface{}) (*FSMEntry, error) {
	entry := &FSMEntry{}
	entry.Owner = owner
	entry.Ctrl = f
	entry.State = f.InitState
	entry.Logs = make([]*TrnasitLog, 0)
	entry.LogMax = f.LogMax

	return entry, nil
}

type InvalidEvent struct {
	Event string
	Err   error
}

func (e *InvalidEvent) Error() string {
	return e.Err.Error() + ": " + e.Event
}

func (e *InvalidEvent) Unwrap() error { return e.Err }

type UndefinedHandle struct {
	State string
	Event string
	Err   error
}

func (e *UndefinedHandle) Error() string {
	return e.Err.Error() + ": State " + e.State + " can't accept " + e.Event
}

func (e *UndefinedHandle) Unwrap() error { return e.Err }

// Do FSM
// ev Event
// logging save transit log
func (e *FSMEntry) DoFSM(ev string, logging bool) (*State, error) {
	event := Event{ev}
	_, found := e.Ctrl.Events[event]
	if !found {
		return nil, &InvalidEvent{Event: ev, Err: fsmerror.ErrEvent}
	}

	handle, found := e.Ctrl.Handles[e.State][event]
	if !found {
		return nil, &UndefinedHandle{State: e.State.State, Event: ev, Err: fsmerror.ErrHandle}
	}

	state := e.State.State
	stateReturned, err := handle.Handle(e.Owner, event)

	// log transit
	log := &TrnasitLog{}
	log.time = time.Now()
	log.state = state
	log.event = event.Event
	log.handle = handle.Name
	log.success = false

	if logging {
		if len(e.Logs) >= e.LogMax {
			// truncate old
			e.Logs = e.Logs[1:len(e.Logs)]
		}
		e.Logs = append(e.Logs, log)
	}

	if err != nil {
		log.err = err
		// DO NOT CHANGE entry.State, if handle failed
		return nil, err
	}

	if len(handle.Cands) > 1 {
		// validate the handle result with candidates
		valid := false
		for _, c := range handle.Cands {
			if c == stateReturned {
				valid = true
				break
			}
		}
		if valid {
			// nextState determined by Handle
			e.State = stateReturned

			log.success = true
			log.next = stateReturned.State
		}
	} else {
		// static nextState determined by FSMCtrl
		e.State = handle.Cands[0]

		log.success = true
		log.next = handle.Cands[0].State
	}

	return &stateReturned, nil
}

func t2s(t time.Time) string {
	return t.Format("2006-01-02 15:04:05 MST")
}

// PrintLog
// last print number of latest n logs, if n > 0
//   otherwise print all logs
func (e *FSMEntry) PrintLog(last int) {
	nLogs := len(e.Logs)
	start := 0
	if last > 0 && nLogs > last {
		start += nLogs - last
	}

	for i := start; i < nLogs; i++ {
		log := e.Logs[i]
		if log.success {
			fmt.Printf("%s State=[%s] Event=[%s] Handle=[%s] Return=%t NextState=[%s] Err=[]\n",
				t2s(log.time), log.state, log.event, log.handle, log.success, log.next)
		} else {
			fmt.Printf("%s State=[%s] Event=[%s] Handle=[%s] Return=%t NextState=[%s] Err=[%s]\n",
				t2s(log.time), log.state, log.event, log.handle, log.success, log.next, log.msg)
		}
	}
}
