package fsm

import (
	"fmt"
	"reflect"
	"runtime"
	"time"

	fsmerror "github.com/HaesungSeo/goFSM/v2/internal/fsmerrors"
)

// FSM Handle Exit Enumeration
const (
	ExitOK    = 0   // Success
	ExitFail  = 1   // Failure
	ExitStart = 2   // Success, conditional code start
	ExitEnd   = 127 // Success, conditional code end
)

type HandleRetCode int

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
	time   time.Time // time event occurs
	state  string    // current State
	event  string    // Event
	handle string    // Func
	ret    int       // Func's return code
	next   string    // next event determined by Handler
	err    error     // Error, from handle
}

// FSM Entry
type Entry[OWNER any, USERDATA any] struct {
	Owner  OWNER                   // FSM owner
	table  *Table[OWNER, USERDATA] // FSM Rule for this Entry
	State  State                   // Current State
	Logs   []*TrnasitLog           // transition log, for debug
	LogMax int
	Datas  map[string]interface{} // storage for temp datas
}

// Set stores tempral variables.
// HandleFunc can call Set() to store temporal data needed between handleFuncs
func (e *Entry[OWNER, USERDATA]) Set(key string, value interface{}) {
	e.Datas[key] = value
}

// Get returns the stored tempral variables.
// HandleFunc can call Get() to get temporal data which saved by other handleFuncs
func (e *Entry[OWNER, USERDATA]) Get(key string) interface{} {
	if v, ok := e.Datas[key]; ok {
		return v
	}
	return nil
}

// status represent the end of transition
type EndOfTrans bool

// FSM State Event Handle Funcion
//
//	HandleRetCode - Handle return code
//	error - handler error, if any
type HandleFuncv2[OWNER any, USERDATA any] func(Owner OWNER, event Event, UserData USERDATA) (HandleRetCode, error)

type CandMap map[HandleRetCode]string

// FSM State Event Handler information
type Handle[OWNER any, USERDATA any] struct {
	Name    string                        // handle name
	Func    HandleFuncv2[OWNER, USERDATA] // handle function
	CandMap CandMap                       // valid next state candidates
}

// FSM Table
type Table[OWNER any, USERDATA any] struct {
	InitState   State
	FinalStates []string
	FSMap       map[string]interface{}
	LogMax      int

	// Valid States
	States map[State]interface{}

	// Valid Events
	Events map[Event]interface{}

	// Handles indexted by State,Event
	Handles map[State]map[Event]*Handle[OWNER, USERDATA]
}

// FSM Event Action Description Table
// CandList and CandMap describe corresponding next State for the handler's return code
// use CandList, for simple case,
//
//	{
//	    State: "Disabled",
//	    Events: []fsm.EventDesc[*MyOwner, *MyData]{
//	        {Event: "Add", Func: DoAdd, CandList: []string{"AddNext", "TryAgain"}},
//	    },
//	}
//
// Func DoAdd() returns one of {ExitOK, ExitFail},
// then the FSM library lookup the next state for the returned code from the CandList[]
// CandList[0] stores the "AddNext" next state for the return code ExitOK(0)
// CandList[1] stores the "TryAgain" next state for the return code ExitFail(0)
//
// use CandMap, if handler return codes are much more or complex
//
//	{
//		const (
//		     UserDefinedCode1 = 100
//		     UserDefinedCode2 = 200
//		     UserDefinedCode3 = 300
//		     UserDefinedCode4 = 400
//		)
//		...
//		{
//		    State: "Disabled",
//		    Events: []fsm.EventDesc[*MyOwner, *MyData]{
//		        {Event: "Add", Func: DoAdd, CandMap: map[HandleRetCode]string{
//		                  UserDefinedCode1: "DoNext",
//		                  UserDefinedCode1: "TryAgain",
//		                  UserDefinedCode1: "CheckRequest",
//		                  },
//		               },
//		    },
//		},
//	}
//
// Func DoAdd() returns one of {OK, NotFound, BadRequest},
// then the FSM library lookup the next state for the returned code from the CandMap[]
// CandMap[100] stores the "DoNext" next state for the return code UserDefinedCode1(100)
// CandMap[200] stores the "TryAgain" next state for the return code UserDefinedCode2(200)
// CandMap[300] stores the "CheckRequest" next state for the return code UserDefinedCode3(300)
type EventDesc[OWNER any, USERDATA any] struct {
	Event    string                        // Event
	Func     HandleFuncv2[OWNER, USERDATA] // Handler for this {State, Event}
	CandMap  CandMap                       // valid next state candidates,
	CandList []string                      // valid next state candidates,
	// if nil, handler MUST PROVIDE next state
}

type StateDesc[OWNER any, USERDATA any] struct {
	State  string
	Events []EventDesc[OWNER, USERDATA]
}

// FSM State-Event Table Descriptor
type TableDesc[OWNER any, USERDATA any] struct {
	InitState   string   // Initial State for Entry
	FinalStates []string // Final States for Entry
	LogMax      int      // maximum lengh of log
	States      []StateDesc[OWNER, USERDATA]
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
	return e.Err.Error() + ": State=" + e.State + ", Event=" + e.Event +
		", Old Func=" + e.OldHandle +
		", New Func=" + e.NewHandle
}

func (e *StateEventConflictError) Unwrap() error { return e.Err }

type HandleRetCodeRangeError struct {
	State  string // current state
	Event  string // input event
	Handle string // current handle
	Code   HandleRetCode
	Err    error
}

func (e *HandleRetCodeRangeError) Error() string {
	return e.Err.Error() + ": Code=" + string(e.Code) + ", State=" + e.State +
		", Event=" + e.Event + ", Func=" + e.Handle
}

func (e *HandleRetCodeRangeError) Unwrap() error { return nil }

type HandleRetCodeDupError struct {
	State  string // current state
	Event  string // input event
	Handle string // current handle
	Code   HandleRetCode
	Err    error
}

func (e *HandleRetCodeDupError) Error() string {
	return e.Err.Error() + ": Code=" + string(e.Code) + ", State=" + e.State +
		", Event=" + e.Event + ", Func=" + e.Handle
}

func (e *HandleRetCodeDupError) Unwrap() error { return nil }

type HandleEmptyRetCodeError struct {
	State  string // current state
	Event  string // input event
	Handle string // current handle
	Err    error
}

func (e *HandleEmptyRetCodeError) Error() string {
	return e.Err.Error() + ": State=" + e.State +
		", Event=" + e.Event + ", Func=" + e.Handle
}

func (e *HandleEmptyRetCodeError) Unwrap() error { return nil }

type Opts map[string]map[int]interface{}

// Create New FSM Control Instance
// d FSM Descritor
func NewTable[OWNER any, USERDATA any](d *TableDesc[OWNER, USERDATA], opts ...Opts) (*Table[OWNER, USERDATA], error) {
	tbl := Table[OWNER, USERDATA]{}

	tbl.States = make(map[State]interface{})
	tbl.Events = make(map[Event]interface{})
	tbl.Handles = make(map[State]map[Event]*Handle[OWNER, USERDATA])
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
			for _, nstate := range event.CandList {
				tbl.States[State{nstate}] = nil
			}
			// Index NextState
			for _, v := range event.CandMap {
				tbl.States[State{v}] = nil
			}
		}
	}

	// Allocate Handles
	for _, state := range d.States {
		tbl.Handles[State{state.State}] = make(map[Event]*Handle[OWNER, USERDATA])
	}

	// Add User defined State-Event-Handles
	for _, state := range d.States {
		for _, event := range state.Events {
			hName := getFunctionName(event.Func)
			handle := &Handle[OWNER, USERDATA]{
				hName,
				event.Func,
				make(CandMap, 0),
			}
			// build vaild next states for corresponding return codes
			for idx, nstate := range event.CandList {
				switch {
				case idx == ExitFail:
					fallthrough
				case idx == ExitOK:
					fallthrough
				case idx >= ExitStart && idx <= ExitEnd:
					if _, ok := handle.CandMap[HandleRetCode(idx)]; ok {
						return nil, &HandleRetCodeDupError{
							State:  state.State,
							Event:  event.Event,
							Handle: hName,
							Code:   HandleRetCode(idx),
						}
					}
					handle.CandMap[HandleRetCode(idx)] = nstate
				default:
					return nil, &HandleRetCodeRangeError{
						State:  state.State,
						Event:  event.Event,
						Handle: hName,
						Code:   HandleRetCode(idx),
					}
				}
			}
			for idx, nstate := range event.CandMap {
				if _, ok := handle.CandMap[HandleRetCode(idx)]; ok {
					return nil, &HandleRetCodeDupError{
						State:  state.State,
						Event:  event.Event,
						Handle: hName,
						Code:   HandleRetCode(idx),
					}
				}
				handle.CandMap[HandleRetCode(idx)] = nstate
			}

			if len(handle.CandMap) == 0 {
				return nil, &HandleEmptyRetCodeError{
					State:  state.State,
					Event:  event.Event,
					Handle: hName,
					Err:    fsmerror.ErrHandleNoRetCode,
				}
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

	// sanity check, return &tbl with error
	for state, _ := range tbl.States {
		// check the state has any handle
		if hmap, ok := tbl.Handles[state]; !ok {
			if _, ok := tbl.FSMap[state.Name]; !ok {
				return &tbl, &UndefinedHandle{
					State: state.Name,
					Event: "any",
					Err:   fsmerror.ErrHandleNotExists,
				}
			}
		} else {
			// check the state is final state and has a (useless) event handler
			_, finalState := tbl.FSMap[state.Name]

			// check the handle has any Funcion
			if len(hmap) == 0 && !finalState {
				return &tbl, &UndefinedHandle{
					State: state.Name,
					Event: "any",
					Err:   fsmerror.ErrHandleNotExists,
				}
			}
			// check the handle has next state, any
			for e, h := range hmap {
				if len(h.CandMap) == 0 && !finalState {
					return &tbl, &UndefinedHandle{
						State: state.Name,
						Event: e.Name,
						Err:   fsmerror.ErrHandleNotExists,
					}
				}
			}
		}
	}

	// check the next state-event has handler
	for _, state := range d.States {
		for _, event := range state.Events {
			for _, nstate := range event.CandList {
				// check the next state-event has handler
				//tbl.States[State{nstate}] = nil
				if _, ok := tbl.Handles[State{nstate}][Event{event.Event}]; !ok {
					if _, ok := tbl.FSMap[nstate]; !ok {
						return &tbl, &UndefinedHandle{
							State: nstate,
							Event: event.Event,
							Err:   fsmerror.ErrHandleNotExists,
						}
					}
				}
			}
			for _, nstate := range event.CandMap {
				//tbl.States[State{v}] = nil
				if _, ok := tbl.Handles[State{nstate}][Event{event.Event}]; !ok {
					if _, ok := tbl.FSMap[nstate]; !ok {
						return &tbl, &UndefinedHandle{
							State: nstate,
							Event: event.Event,
							Err:   fsmerror.ErrHandleNotExists,
						}
					}
				}
			}

			// check all possible return code
			hName := getFunctionName(event.Func)
			for _, fmap := range opts {
				if retmap, ok := fmap[hName]; ok {
					// we have validator
					for ret_code, _ := range retmap {
						hExists := false
						// check the handle's CandList or CandMap has the ret_code defined
						if len(event.CandList) >= ret_code {
							hExists = true
						} else {
							if _, ok := event.CandMap[HandleRetCode(ret_code)]; ok {
								hExists = true
							}
						}
						if !hExists {
							// handler for return code not found
							return nil, &UndefinedRetCode{
								State:   state.State,
								Event:   event.Event,
								Handle:  hName,
								RetCode: HandleRetCode(ret_code),
								Err:     fsmerror.ErrInvalidRetCode,
							}
						}
					}
				}
			}
		}
	}

	return &tbl, nil
}

// Dump Handlers
func (tbl *Table[ONWER, USERDATA]) Dump() {
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
			keys := make([]HandleRetCode, 0, len(handle.CandMap))
			for hrc := range handle.CandMap {
				keys = append(keys, hrc)
			}
			for i, k := range keys {
				switch i {
				case 0:
					// first return code
					fmt.Printf("  Event[%s] Func[%s] Return code[%d] Next State[%s]\n",
						event, handle.Name, k, handle.CandMap[HandleRetCode(k)])
				default:
					fmt.Printf("    Return code[%d] Next State[%s]\n", k, handle.CandMap[HandleRetCode(k)])
				}
			}
		}
	}
}

// Create New FSM Entry Instance, controlled by Table(FSM Control) Instance
// owner Entry Owner
func (tbl *Table[OWNER, USERDATA]) NewEntry(owner OWNER) *Entry[OWNER, USERDATA] {
	entry := &Entry[OWNER, USERDATA]{}
	entry.Owner = owner
	entry.table = tbl
	entry.State = tbl.InitState
	entry.Logs = make([]*TrnasitLog, 0)
	entry.LogMax = tbl.LogMax
	entry.Datas = make(map[string]interface{})

	return entry
}

// Invalid Event Error
type InvalidEvent struct {
	Event string
	Err   error
}

func (e *InvalidEvent) Error() string {
	return e.Err.Error() + ": Event=" + e.Event
}

func (e *InvalidEvent) Unwrap() error { return e.Err }

// Undefined Func Error
type UndefinedHandle struct {
	State string
	Event string
	Err   error
}

func (e *UndefinedHandle) Error() string {
	return e.Err.Error() + ": State=" + e.State + ", Event=" + e.Event
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
	return e.Err.Error() + ": State=" + e.State +
		", Event=" + e.Event + ", Handle=" + e.Handle +
		", Next State=" + e.nState
}

func (e *UndefinedNextState) Unwrap() error { return e.Err }

// Undefined return code Error
type UndefinedRetCode struct {
	State   string
	Event   string
	Handle  string
	RetCode HandleRetCode
	Err     error
}

func (e *UndefinedRetCode) Error() string {
	return e.Err.Error() + ": Code=" + string(e.RetCode) +
		", State=" + e.State + ", Event=" + e.Event +
		", Handle=" + e.Handle
}

func (e *UndefinedRetCode) Unwrap() error { return e.Err }

// Do FSM
// ev Event
// userData event specific data
// returns
//
//	State - next state
//	bool - represents end of transition
//	error - handler returned error
func (e *Entry[OWNER, USERDATA]) TransitWithData(ev string, userData USERDATA) (State, bool, error) {
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

	eot := false // remark the end of transit
	state := e.State.Name
	retCode, err := handle.Func(e.Owner, event, userData)

	if state, ok := handle.CandMap[retCode]; !ok {
		eot = true
		err = &UndefinedRetCode{
			State:   e.State.Name,
			Event:   ev,
			Handle:  handle.Name,
			RetCode: retCode,
			Err:     fsmerror.ErrInvalidRetCode,
		}
	} else {
		e.State = State{state}

		// check the next state is defined as final state
		if _, ok := e.table.FSMap[e.State.Name]; ok {
			eot = true
		}
	}

	if e.LogMax > 0 {
		log := &TrnasitLog{}
		log.time = time.Now()
		log.state = state
		log.event = event.Name
		log.handle = handle.Name
		log.ret = int(retCode)
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
func (e *Entry[OWNER, USERDATA]) Transit(ev string) (State, bool, error) {
	var d USERDATA
	return e.TransitWithData(ev, d)
}

func t2s(t time.Time) string {
	return t.Format("2006-01-02 15:04:05 MST")
}

// PrintLog
// last print number of latest n logs, if n > 0
//
//	otherwise print all logs
func (e *Entry[OWNER, USERDATA]) PrintLog(last int) {
	nLogs := len(e.Logs)
	start := 0
	if last > 0 && nLogs > last {
		start += nLogs - last
	}

	for i := start; i < nLogs; i++ {
		log := e.Logs[i]
		if log.err != nil {
			fmt.Printf("%s State=[%s] Event=[%s] Func=[%s] RetCode=[%d] NextState=[%s] Err=[%s]\n",
				t2s(log.time), log.state, log.event, log.handle, log.ret, log.next, log.err.Error())
		} else {
			fmt.Printf("%s State=[%s] Event=[%s] Func=[%s] RetCode=[%d] NextState=[%s]\n",
				t2s(log.time), log.state, log.event, log.handle, log.ret, log.next)
		}
	}
}
