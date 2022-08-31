// Finite State Machine(FSM) for go
package fsm

import (
	"fmt"
	"reflect"
	"runtime"
	"time"

	fsmerror "github.com/HaesungSeo/goFSM/internal/fsmerrors"
)

// FSM State
type State struct {
	Name string
}

// FSM Event
type Event struct {
	Name string
}

// FSM State Event Transition log information
type TrnasitLog struct {
	time    time.Time // time event occurs
	state   string    // current State
	event   string    // Event
	handle  string    // Func
	success bool      // Func result
	next    string    // next event determined by Handler
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
// returns
//   State - next state
//   bool - End of Transition
//   error - handler error
type HandleFunc func(Owner interface{}, event Event, UserData interface{}) (State, bool, error)

type Handle struct {
	Default bool       // is default handler
	Name    string     // handle name
	Func    HandleFunc // handle function
	Cands   []State    // valid next state candidates
}

type Table struct {
	InitState   State
	FinalStates []string
	FSMap       map[string]interface{}
	LogMax      int

	// Valid States
	States map[State]interface{}

	// Valid Events
	Events map[Event]interface{}

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
	InitState   string   // Initial State for Entry
	FinalStates []string // Final States for Entry
	LogMax      int      // maximum lengh of log
	States      []StateDesc
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
	return e.Err.Error() + ": State " + e.State + " Event " + e.Event + " Old Func " +
		e.OldHandle + " New Func " + e.NewHandle
}

func (e *StateEventConflictError) Unwrap() error { return e.Err }

// Create New FSM Control
// d FSM Descritor
func New(d TableDesc) (*Table, error) {
	tbl := Table{}

	tbl.States = make(map[State]interface{})
	tbl.Events = make(map[Event]interface{})
	tbl.Handles = make(map[State]map[Event]Handle)
	tbl.FSMap = make(map[string]interface{})

	tbl.InitState = State{d.InitState}
	tbl.FinalStates = d.FinalStates
	for _, s := range d.FinalStates {
		tbl.FSMap[s] = nil
	}
	tbl.LogMax = d.LogMax

	// Initialize given states, events
	for _, state := range d.States {
		// Index State
		tbl.States[State{state.State}] = nil

		for _, event := range state.Events {
			// Index Events
			tbl.Events[Event{event.Event}] = nil

			// Index NextState
			for _, nstate := range event.Candidates {
				tbl.States[State{nstate}] = nil
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

	fmt.Printf("FinalStates\n")
	for _, state := range tbl.FinalStates {
		fmt.Printf("  [%s]\n", state)
	}

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
func (tbl *Table) NewEntry(owner interface{}) *Entry {
	entry := &Entry{}
	entry.Owner = owner
	entry.table = tbl
	entry.State = tbl.InitState
	entry.Logs = make([]*TrnasitLog, 0)
	entry.LogMax = tbl.LogMax

	return entry
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
	State  string
	Event  string
	Handle string
	nState string
	Err    error
}

func (e *UndefinedHandle) Error() string {
	return e.Err.Error() + ": State " + e.State + " Event " + e.Event +
		": Handle not defined"
}

func (e *UndefinedHandle) Unwrap() error { return e.Err }

// Undefined next State Error
type UndefinedNextState struct {
	State  string
	Event  string
	Handle string
	nState string
	Err    error
}

func (e *UndefinedNextState) Error() string {
	return e.Err.Error() + ": State " + e.State +
		" Event " + e.Event + " Handle " + e.Handle +
		": returns undefined state " + e.nState
}

func (e *UndefinedNextState) Unwrap() error { return e.Err }

// Do FSM
// ev Event
// userData event specific data
// returns
//    State - next state
//    bool - represents end of transition
//    error - handler returned error
func (e *Entry) TransitWithData(ev string, userData interface{}) (State, bool, error) {
	event := Event{ev}
	_, found := e.table.Events[event]
	if !found {
		return State{}, true, &InvalidEvent{Event: ev, Err: fsmerror.ErrInvalidEvent}
	}

	handle, found := e.table.Handles[e.State][event]
	if !found {
		// no handle for this state-event pair
		// may stop the transition for this {state, event} pair
		return State{}, true, &UndefinedHandle{State: e.State.Name, Event: ev, Err: fsmerror.ErrHandleNotExists}
	}

	state := e.State.Name
	stateReturned, eot, err := handle.Func(e.Owner, event, userData)

	if stateReturned.Name == "" {
		// fsm handle follow the fsm description tables' next state
		if len(handle.Cands) > 1 {
			// too many next state
			eot = true
			err = &UndefinedNextState{
				State:  e.State.Name,
				Event:  ev,
				Handle: handle.Name,
				nState: "(nil)",
				Err:    fsmerror.ErrInvNextState,
			}
		} else {
			e.State = handle.Cands[0]
		}
	} else {
		// state which fsm handle returned, must be one of the pre-defined candidates
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
			} else {
				eot = true
				err = &UndefinedNextState{
					State:  e.State.Name,
					Event:  ev,
					Handle: handle.Name,
					nState: stateReturned.Name,
					Err:    fsmerror.ErrInvNextState,
				}
			}
		} else if handle.Cands[0] == stateReturned {
			// static nextState determined by FSMCtrl
			e.State = handle.Cands[0]
		} else {
			eot = true
			err = &UndefinedNextState{
				State:  e.State.Name,
				Event:  ev,
				Handle: handle.Name,
				nState: stateReturned.Name,
				Err:    fsmerror.ErrInvNextState,
			}
		}
	}

	// check the next state is defined as final state
	if _, ok := e.table.FSMap[e.State.Name]; ok {
		eot = true
	}

	if e.LogMax > 0 {
		log := &TrnasitLog{}
		log.time = time.Now()
		log.state = state
		log.event = event.Name
		log.handle = handle.Name
		log.next = e.State.Name
		log.err = err

		if len(e.Logs) >= e.LogMax {
			// truncate old
			e.Logs = e.Logs[1:len(e.Logs)]
		}
		e.Logs = append(e.Logs, log)
	}

	return e.State, eot, err
}

// Do FSM
// ev Event
func (e *Entry) Transit(ev string) (State, bool, error) {
	return e.TransitWithData(ev, nil)
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
		if log.err != nil {
			fmt.Printf("%s State=[%s] Event=[%s] Func=[%s] NextState=[%s] Err=[%s]\n",
				t2s(log.time), log.state, log.event, log.handle, log.err.Error())
		} else {
			fmt.Printf("%s State=[%s] Event=[%s] Func=[%s] NextState=[%s]\n",
				t2s(log.time), log.state, log.event, log.handle)
		}
	}
}
