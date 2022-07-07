// Finite State Machine(FSM) for go
package fsm

import (
	"fmt"
	"reflect"
	"runtime"
	"time"

	fsmerror "github.com/HaesungSeo/goFSM/internal/fsmerrors"
)

type State struct {
	Name string
}

type Event struct {
	Name string
}

type TrnasitLog struct {
	time    time.Time // time event occurs
	state   string    // current State
	event   string    // Event
	handle  string    // Func
	success bool      // Func result
	next    string    // next event determined by Handler
	msg     string    // Messages related for this fsm event
	err     error     // Error, from handle
}

// FSM Entry
type Entry struct {
	Owner  interface{}   // Owner Entry
	table  *Table        // FSM Rule for this Entry
	State  State         // Current State
	Logs   []*TrnasitLog // transition log, for debug
	LogMax int
}

// State Event Handle Function
// returns (nextState, error)
type HandleFunc func(Owner interface{}, event Event, UserData interface{}) (State, error)

type Handle struct {
	Default bool       // is default handler
	Name    string     // handle name
	Func    HandleFunc // handle function
	Cands   []State    // valid next state candidates
}

type Table struct {
	InitState State
	LogMax    int

	// Valid States
	States map[State]struct{}

	// Valid Events
	Events map[Event]struct{}

	// Handles indexted by State,Event
	Handles map[State]map[Event]Handle
}

// FSM Action Description Table
type EventDesc struct {
	Event      string     // Event
	Func       HandleFunc // Handler for this {State, Event}
	Candidates []string   // valid next state candidates,
	// if nil, handler MUST PROVIDE next state
}

type StateDesc struct {
	State  string
	Events []EventDesc
}

// FSM State-Event Descriptor
type TableDesc struct {
	InitState string // Initial State for Entry
	LogMax    int    // maximum lengh of log
	States    []StateDesc
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
	return "State:" + e.State + ", Event:" + e.Event + " Old Func " +
		e.OldHandle + " New Func " + e.NewHandle + ": " +
		e.Err.Error()
}

func (e *StateEventConflictError) Unwrap() error { return e.Err }

// Create New FSM Control
// d FSM Descritor
func New(d TableDesc) (*Table, error) {
	tbl := Table{}

	tbl.States = make(map[State]struct{})
	tbl.Events = make(map[Event]struct{})
	tbl.Handles = make(map[State]map[Event]Handle)

	tbl.InitState = State{d.InitState}
	tbl.LogMax = d.LogMax

	// Initialize given states, events
	for _, state := range d.States {
		// Index State
		tbl.States[State{state.State}] = struct{}{}

		for _, event := range state.Events {
			// Index Events
			tbl.Events[Event{event.Event}] = struct{}{}

			// Index NextState
			for _, nstate := range event.Candidates {
				tbl.States[State{nstate}] = struct{}{}
			}
		}
	}

	// Allocate Handles
	for _, state := range d.States {
		tbl.Handles[State{state.State}] = make(map[Event]Handle)
	}

	// Add User defined State-Event-Handles
	for _, state := range d.States {
		for _, event := range state.Events {
			hName := getFunctionName(event.Func)
			handle := Handle{
				false,
				hName,
				event.Func,
				make([]State, 0),
			}
			for _, nstate := range event.Candidates {
				handle.Cands = append(handle.Cands, State{nstate})
			}

			s, statefound := tbl.Handles[State{state.State}]
			if statefound {
				old, handlefound := s[Event{event.Event}]
				if handlefound {
					if &old.Func != &handle.Func {
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
			tbl.Handles[State{state.State}][Event{event.Event}] = handle
		}
	}

	return &tbl, nil
}

// Dump Handlers
func (tbl *Table) Dump() {
	fmt.Printf("InitState[%s]\n", tbl.InitState)

	fmt.Printf("All States\n")
	for state, _ := range tbl.States {
		fmt.Printf("  [%s]\n", state)
	}

	fmt.Printf("All Events\n")
	for event, _ := range tbl.Events {
		fmt.Printf("  [%s]\n", event)
	}

	for state, events := range tbl.Handles {
		fmt.Printf("State[%s]\n", state)
		for event, handle := range events {
			fmt.Printf("  Event[%s] Func[%s] NextState[%s]\n", event, handle.Name, handle.Cands)
		}
	}
}

// Do FSM
func (tbl *Table) NewEntry(owner interface{}) (*Entry, error) {
	entry := &Entry{}
	entry.Owner = owner
	entry.table = tbl
	entry.State = tbl.InitState
	entry.Logs = make([]*TrnasitLog, 0)
	entry.LogMax = tbl.LogMax

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

// Undefined Func Error
type UndefinedHandle struct {
	State string
	Event string
	Err   error
}

func (e *UndefinedHandle) Error() string {
	return e.Err.Error() + ": State " + e.State + " Event " + e.Event
}

func (e *UndefinedHandle) Unwrap() error { return e.Err }

// Do FSM
// ev Event
// logging save transit log
func (e *Entry) TransitWithData(ev string, userData interface{}, logging bool) (State, error) {
	event := Event{ev}
	_, found := e.table.Events[event]
	if !found {
		return State{}, &InvalidEvent{Event: ev, Err: fsmerror.ErrInvalidEvent}
	}

	handle, found := e.table.Handles[e.State][event]
	if !found {
		// no handle for this state-event pair
		// may stop the transition for this {state, event} pair
		return State{}, &UndefinedHandle{State: e.State.Name, Event: ev, Err: fsmerror.ErrHandleNotExists}
	}

	state := e.State.Name
	stateReturned, err := handle.Func(e.Owner, event, userData)
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
				// nextState determined by Func
				e.State = stateReturned
				success = true
			}
		} else {
			// static nextState determined by FSMCtrl
			e.State = handle.Cands[0]
			success = true
		}
	}

	if e.LogMax > 0 && logging {
		// logging enabled
		log := &TrnasitLog{}
		log.time = time.Now()
		log.state = state
		log.event = event.Name
		log.handle = handle.Name
		log.success = success
		log.next = e.State.Name
		log.err = err

		if len(e.Logs) >= e.LogMax {
			// truncate old
			e.Logs = e.Logs[1:len(e.Logs)]
		}
		e.Logs = append(e.Logs, log)
	}

	return e.State, err
}

func (e *Entry) Transit(ev string, logging bool) (State, error) {
	return e.TransitWithData(ev, nil, logging)
}

func t2s(t time.Time) string {
	return t.Format("2006-01-02 15:04:05 MST")
}

// PrintLog
// last print number of latest n logs, if n > 0
//   otherwise print all logs
func (e *Entry) PrintLog(last int) {
	nLogs := len(e.Logs)
	start := 0
	if last > 0 && nLogs > last {
		start += nLogs - last
	}

	for i := start; i < nLogs; i++ {
		log := e.Logs[i]
		if log.success {
			fmt.Printf("%s State=[%s] Event=[%s] Func=[%s] Return=%t NextState=[%s] Msg=[%s]\n",
				t2s(log.time), log.state, log.event, log.handle, log.success, log.msg)
		} else {
			fmt.Printf("%s State=[%s] Event=[%s] Func=[%s] Return=%t NextState=[%s] Msg=[%s] Err=[%s]\n",
				t2s(log.time), log.state, log.event, log.handle, log.success, log.next, log.msg, log.err.Error())
		}
	}
}
