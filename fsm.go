package goFSM

import (
	"fmt"
	"reflect"
	"runtime"
	"time"

	fsmerror "github.com/HaesungSeo/goFSM/internal/fsmerrors"
)

// FSM State
type State struct {
	State string
}

// FSM Event
type Event struct {
	Event string
}

// FSM State Event Transition log information
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
type FSMEntry[OWNER any, USERDATA any] struct {
	Owner  OWNER                    // FSM owner
	Ctrl   *FSMCTL[OWNER, USERDATA] // FSM Rule for this Entry
	State  State                    // Current State
	Logs   []*TrnasitLog            // transition log, for debug
	LogMax int
}

// FSM State Event Handle func
type FsmCallback[OWNER any, USERDATA any] func(Owner OWNER, event Event, UserData USERDATA) (State, error)

// FSM State Event Handler information
type FsmHandle[OWNER any, USERDATA any] struct {
	Name   string                       // handle name
	Handle FsmCallback[OWNER, USERDATA] // handle function
	Cands  []State                      // valid next state candidates
}

// FSM Table
type FSMCTL[OWNER any, USERDATA any] struct {
	InitState State
	LogMax    int

	// Valid States
	States map[State]struct{}

	// Valid Events
	Events map[Event]struct{}

	// Handles indexted by State,Event
	Handles map[State]map[Event]FsmHandle[OWNER, USERDATA]
}

// FSM Action Description Table
type FSMDescEvent[OWNER any, USERDATA any] struct {
	Event      string                       // Event
	Handle     FsmCallback[OWNER, USERDATA] // Handler for this {State, Event}
	Candidates []string                     // valid next state candidates,
	// if nil, handler MUST PROVIDE next state
}

type EventDesc[OWNER any, USERDATA any] []FSMDescEvent[OWNER, USERDATA]

type FSMDescState[OWNER any, USERDATA any] struct {
	State  string
	Events EventDesc[OWNER, USERDATA]
}
type StateDesc[OWNER any, USERDATA any] []FSMDescState[OWNER, USERDATA]

// FSM State-Event Descriptor
type FSMDesc[OWNER any, USERDATA any] struct {
	InitState string // Initial State for FSMEntry
	LogMax    int    // maximum lengh of log
	States    StateDesc[OWNER, USERDATA]
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

// Create New FSM Control Instance
// d FSM Descritor
func FsmNew[OWNER any, USERDATA any](d *FSMDesc[OWNER, USERDATA]) (*FSMCTL[OWNER, USERDATA], error) {
	newFsm := FSMCTL[OWNER, USERDATA]{}

	newFsm.States = make(map[State]struct{})
	newFsm.Events = make(map[Event]struct{})
	newFsm.Handles = make(map[State]map[Event]FsmHandle[OWNER, USERDATA])

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
		newFsm.Handles[State{state.State}] = make(map[Event]FsmHandle[OWNER, USERDATA])
	}

	// Add User defined State-Event-Handles
	for _, state := range d.States {
		for _, event := range state.Events {
			hName := getFunctionName(event.Handle)
			handle := FsmHandle[OWNER, USERDATA]{
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
							Err:       fsmerror.ErrDupHandle,
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
func (f *FSMCTL[ONWER, USERDATA]) DumpTable() {
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

type linker[OWNER any, USERDATA any] func(owner OWNER, entry *FSMEntry[OWNER, USERDATA])

// Create New FSM Entry Instance, controlled by FSMCTL(FSM Control) Instance
// owner Entry Owner
func (f *FSMCTL[OWNER, USERDATA]) NewEntry(owner OWNER, l linker[OWNER, USERDATA]) (*FSMEntry[OWNER, USERDATA], error) {
	entry := &FSMEntry[OWNER, USERDATA]{}
	entry.Owner = owner
	entry.Ctrl = f
	entry.State = f.InitState
	entry.Logs = make([]*TrnasitLog, 0)
	entry.LogMax = f.LogMax
	l(owner, entry)

	return entry, nil
}

// Invalid Event Error
type InvalidEvent struct {
	Event string
	Err   error
}

func (e *InvalidEvent) Error() string {
	return e.Err.Error() + ": " + e.Event
}

func (e *InvalidEvent) Unwrap() error { return e.Err }

// Undefined Handle Error
type UndefinedHandle struct {
	State string
	Event string
	Err   error
}

func (e *UndefinedHandle) Error() string {
	return e.Err.Error() + ": State " + e.State + " Event " + e.Event
}

func (e *UndefinedHandle) Unwrap() error { return e.Err }

// Undefined next State Error
type UndefinedNextState struct {
	State  string
	Event  string
	nState string
	Err    error
}

func (e *UndefinedNextState) Error() string {
	return e.Err.Error() + ": State " + e.State + " Event " + e.Event + " nextState " + e.nState
}

func (e *UndefinedNextState) Unwrap() error { return e.Err }

// Do FSM
// ev Event
// userData event specific data
func (e *FSMEntry[OWNER, USERDATA]) DoFSMwithData(ev string, userData USERDATA) (State, error) {
	event := Event{ev}
	_, found := e.Ctrl.Events[event]
	if !found {
		return State{}, &InvalidEvent{Event: ev, Err: fsmerror.ErrInvalidEvent}
	}

	handle, found := e.Ctrl.Handles[e.State][event]
	if !found {
		// no handle for this state-event pair
		// may stop the transition for this {state, event} pair
		return State{}, &UndefinedHandle{State: e.State.State, Event: ev, Err: fsmerror.ErrHandleNotExists}
	}

	state := e.State.State
	stateReturned, err := handle.Handle(e.Owner, event, userData)
	success := false

	if err != nil {
		// no state change at all
	} else {
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
				success = true
			} else {
				err = &UndefinedNextState{State: e.State.State, Event: ev, nState: stateReturned.State, Err: fsmerror.ErrInvNextState}
			}
		} else if handle.Cands[0] == stateReturned {
			// static nextState determined by FSMCtrl
			e.State = handle.Cands[0]
			success = true
		} else {
			err = &UndefinedNextState{State: e.State.State, Event: ev, nState: stateReturned.State, Err: fsmerror.ErrInvNextState}
		}
	}

	if e.LogMax > 0 {
		log := &TrnasitLog{}
		log.time = time.Now()
		log.state = state
		log.event = event.Event
		log.handle = handle.Name
		log.success = success
		log.next = e.State.State
		log.err = err

		if len(e.Logs) >= e.LogMax {
			// truncate old
			e.Logs = e.Logs[1:len(e.Logs)]
		}
		e.Logs = append(e.Logs, log)
	}

	return e.State, err
}

// Do FSM
// ev Event
func (e *FSMEntry[OWNER, USERDATA]) DoFSM(ev string) (State, error) {
	var d USERDATA
	return e.DoFSMwithData(ev, d)
}

func t2s(t time.Time) string {
	return t.Format("2006-01-02 15:04:05 MST")
}

// PrintLog
// last print number of latest n logs, if n > 0
//   otherwise print all logs
func (e *FSMEntry[OWNER, USERDATA]) PrintLog(last int) {
	nLogs := len(e.Logs)
	start := 0
	if last > 0 && nLogs > last {
		start += nLogs - last
	}

	for i := start; i < nLogs; i++ {
		log := e.Logs[i]
		if log.success {
			fmt.Printf("%s State=[%s] Event=[%s] Handle=[%s] Return=%t NextState=[%s] Msg=[%s]\n",
				t2s(log.time), log.state, log.event, log.handle, log.success, log.next, log.msg)
		} else {
			fmt.Printf("%s State=[%s] Event=[%s] Handle=[%s] Return=%t NextState=[%s] Msg=[%s] Err=[%s]\n",
				t2s(log.time), log.state, log.event, log.handle, log.success, log.next, log.msg, log.err.Error())
		}
	}
}
