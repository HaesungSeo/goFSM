package main

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"time"
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
}

// FSM Entry
type FSMEntry struct {
	Head  interface{}   // Owner Entry
	ctrl  *FSMCTL       // FSM Rule for this Entry
	State State         // Current State
	Logs  []*TrnasitLog // transition log, for debug
}

type FsmCallback func(n *FSMEntry, e Event) (State, error)

func fsmCallbackDefault(n *FSMEntry, e Event) (State, error) {
	errStr := fmt.Sprintf("State[%s] Event[%s] Undefined",
		n.State.State, e.Event)
	return State{}, errors.New(errStr)
}

type FsmHandle struct {
	Default bool        // is default handler
	Name    string      // handle name
	Handle  FsmCallback // handle function
	Cands   []State     // valid next state candidates
}

type FSMCTL struct {
	InitState State

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

type FSMDescEvents []FSMDescEvent

type FSMDescState struct {
	State  string
	Events FSMDescEvents
}
type FSMDescStates []FSMDescState

// FSM State-Event Descriptor
type FSMDesc struct {
	InitState string // Initial State for FSMEntry
	States    FSMDescStates
}

func getFunctionName(i interface{}) string {
	funcName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	// iIdx := strings.Index(funcName, "Holder.") + 7
	// jIdx := strings.Index(funcName, "-fm")
	// funcName = funcName[iIdx:jIdx]
	return funcName
}

// Create New FSM Control
// d FSM Descritor
// verbose, level of verbosity, allow print warnings (0 for disabled)
func FSMCTLNew(d FSMDesc, verbose int) (*FSMCTL, error) {
	newFsm := FSMCTL{}

	newFsm.States = make(map[State]struct{})
	newFsm.Events = make(map[Event]struct{})
	newFsm.Handles = make(map[State]map[Event]FsmHandle)

	newFsm.InitState = State{d.InitState}

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

		// Init Handles
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

			s, found := newFsm.Handles[State{state.State}]
			if found {
				old, found := s[Event{event.Event}]
				if found {
					if &old.Handle != &handle.Handle {
						errStr := fmt.Sprintf("Duplicated: State[%s] Event[%s] old Handle[%s] != new Handle[%s]",
							state.State, event.Event, old.Name, hName)
						if verbose > 0 {
							fmt.Fprintf(os.Stderr, "Warning: %s\n", errStr)
						}
					}
				}
			}

			// Add handle
			newFsm.Handles[State{state.State}][Event{event.Event}] = handle
		}
	}

	// Fill Undefined State-Event-Handles with NO-OP
	for state, _ := range newFsm.States {
		_, found := newFsm.Handles[state]
		if !found {
			// Maybe NextState not used as Current State for some Events
			errStr := fmt.Sprintf("No State[%s], but NextState exists",
				state)
			// Dont return incomplete fsm, it may cause panic
			return nil, errors.New(errStr)
		}

		for event, _ := range newFsm.Events {
			hName := getFunctionName(fsmCallbackDefault)
			nopHandle := FsmHandle{
				true,
				hName,
				fsmCallbackDefault,
				make([]State, 0),
			}

			_, found := newFsm.Handles[state][event]
			if found {
				continue
			}
			newFsm.Handles[state][event] = nopHandle
			errStr := fmt.Sprintf("State[%s] Event[%s] get NoOp", state, event)
			if verbose > 0 {
				fmt.Fprintf(os.Stderr, "Warning: %s\n", errStr)
			}
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
func (f *FSMCTL) NewEntry() (*FSMEntry, error) {
	entry := &FSMEntry{}
	entry.ctrl = f
	entry.State = f.InitState
	entry.Logs = make([]*TrnasitLog, 0)

	return entry, nil
}

// Do FSM
// ev Event
// logging save transit log
func (e *FSMEntry) DoFSM(ev string, logging bool) (*State, error) {
	event := Event{ev}
	_, found := e.ctrl.Events[event]
	if !found {
		errStr := fmt.Sprintf("undefined Event: %s", ev)
		return nil, errors.New(errStr)
	}

	handle, found := e.ctrl.Handles[e.State][event]
	if !found {
		errStr := fmt.Sprintf("No handle for State: %s, Event: %s", e.State.State, ev)
		return nil, errors.New(errStr)
	}

	if handle.Default {
		return &State{e.State.State}, nil
	}

	state := e.State.State
	stateReturned, err := handle.Handle(e, event)

	// log transit
	log := &TrnasitLog{}
	log.time = time.Now()
	log.state = state
	log.event = event.Event
	log.handle = handle.Name
	log.success = false

	if logging {
		e.Logs = append(e.Logs, log)
	}

	if err != nil {
		log.msg = err.Error()
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
			log.success = true
			log.next = stateReturned.State
			e.State = stateReturned
		}
	} else {
		log.success = true
		log.next = handle.Cands[0].State
		// static nextState determined by FSMCtrl
		e.State = handle.Cands[0]
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

func OpenDoor(n *FSMEntry, e Event) (State, error) {
	fmt.Printf("%p: State=%s, Event=%s, OpenDoor\n", n, n.State, e.Event)
	return State{"Opened"}, nil
}

func CloseDoor(n *FSMEntry, e Event) (State, error) {
	fmt.Printf("%p: State=%s, Event=%s, CloseDoor\n", n, n.State, e.Event)
	return State{"Closed"}, nil
}

func LockDoor(n *FSMEntry, e Event) (State, error) {
	fmt.Printf("%p: State=%s, Event=%s, LockDoor\n", n, n.State, e.Event)
	return State{"Locked"}, nil
}

func UnlockDoor(n *FSMEntry, e Event) (State, error) {
	fmt.Printf("%p: State=%s, Event=%s, UnlockDoor\n", n, n.State, e.Event)
	return State{"Closed"}, nil
}

func hello() {
	d := FSMDesc{
		InitState: "Closed",
		States: FSMDescStates{
			{
				State: "Closed",
				Events: FSMDescEvents{
					{Event: "Open", Handle: OpenDoor, Candidates: []string{"Opened"}},
					{Event: "Lock", Handle: LockDoor, Candidates: []string{"Closed", "Locked"}},
				},
			},
			{
				State: "Opened",
				Events: FSMDescEvents{
					{Event: "Close", Handle: CloseDoor, Candidates: []string{"Closed"}},
				},
			},
			{
				State: "Locked",
				Events: FSMDescEvents{
					{Event: "Unlock", Handle: UnlockDoor, Candidates: []string{"Closed", "Locked"}},
				},
			},
		},
	}

	fsmCtl, err := FSMCTLNew(d, 0)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}
	fsmCtl.DumpTable()

	e, err := fsmCtl.NewEntry()
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}

	e.DoFSM("Open", true)
	e.DoFSM("Close", true)
	e.DoFSM("Close", true)

	e.PrintLog(0)
}

func main() {
	hello()
}
