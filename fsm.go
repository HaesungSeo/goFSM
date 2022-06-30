package main

import (
	"fmt"
	"reflect"
	"runtime"
)

type State struct {
	State string
}

type Event struct {
	Event string
}

type TrnasitLog struct {
	state   State  // current State
	event   Event  // Event
	success bool   // Handler result
	next    Event  // next event determined by Handler
	msg     string // Messages related for this fsm event
}

// FSM Entry
type FSMEntry struct {
	Head  interface{}  // Owner Entry
	State State        // Current State
	Logs  []TrnasitLog // transition log, for debug
}

type FsmCallback func(n *FSMEntry, e Event) (State, error)

type FsmHandle struct {
	Default   bool        // is default handler
	Name      string      // handle name
	Handle    FsmCallback // handle function
	NextState State       // default next states,
}

type FSMCTL struct {
	InitalState State

	// Valid States
	States map[State]struct{}

	// Valid Events
	Events map[Event]struct{}

	// Handles indexted by State,Event
	Handles map[State]map[Event]FsmHandle
}

// FSM Action Description Table
type FSMDescEvent struct {
	Event     Event       // Event
	Handle    FsmCallback // Handler for this {State, Event}
	NextState State       // default next states,
	// if nil, handler MUST PROVIDE next state
}

type FSMDescState struct {
	State  State
	Events []FSMDescEvent
}

// FSM State-Event Descriptor
type FSMDesc struct {
	InitState State // Initial State for FSMEntry
	States    []FSMDescState
}

func getFunctionName(i interface{}) string {
	funcName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	// iIdx := strings.Index(funcName, "Holder.") + 7
	// jIdx := strings.Index(funcName, "-fm")
	// funcName = funcName[iIdx:jIdx]
	return funcName
}

// Methods
func FSMCTLNew(d FSMDesc) (*FSMCTL, error) {
	newFsm := FSMCTL{}

	newFsm.States = make(map[State]struct{})
	newFsm.Events = make(map[Event]struct{})
	newFsm.Handles = make(map[State]map[Event]FsmHandle)

	newFsm.InitalState = d.InitState

	for _, state := range d.States {
		// Index State
		newFsm.States[state.State] = struct{}{}

		// Init Handles
		newFsm.Handles[state.State] = make(map[Event]FsmHandle)

		for _, event := range state.Events {
			// Index Events
			newFsm.Events[event.Event] = struct{}{}

			// Index handles
			newFsm.Handles[state.State][event.Event] = FsmHandle{
				false,
				getFunctionName(event.Handle),
				event.Handle,
				event.NextState}
		}
	}

	return &newFsm, nil
}

// Dump Handlers
func (f *FSMCTL) DumpTable() {
	for state, events := range f.Handles {
		fmt.Printf("State[%s]\n", state)
		isHead := false
		for event, handle := range events {
			if isHead {
				fmt.Printf("  Event[%s] Func[%s]\n", event, handle.Name)
				isHead = true
			} else {
				fmt.Printf("    Func[%s]\n", handle.Name)
			}
		}
	}
}

func OpenDoor(n *FSMEntry, e Event) (State, error) {
	fmt.Printf("%p: DO State=%s, Event=%s\n", n, n.State, e.Event)
	return State{"Opened"}, nil
}

func main() {
	e := FSMDescEvent{Event{"Open"}, OpenDoor, State{"Opened"}}
	s := FSMDescState{State{"Closed"}, []FSMDescEvent{e}}
	d := FSMDesc{State{"Closed"}, []FSMDescState{s}}
	fsmCtl, err := FSMCTLNew(d)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}
	fsmCtl.DumpTable()
}
