package main

import (
	"errors"
	"flag"
	"fmt"

	fsm "github.com/HaesungSeo/goFSM/internal/fsm"
	fsmerror "github.com/HaesungSeo/goFSM/internal/fsmerrors"
)

type Door struct {
	fmsEntry *fsm.FSMEntry
	name     string
}

type Key struct {
	id string
}

func OpenDoor(owner interface{}, event fsm.Event, _ interface{}) (fsm.State, error) {
	door := owner.(*Door)
	entry := door.fmsEntry
	fmt.Printf("%s: State=%s, Event=%s, Action=OpenDoor\n", door.name, entry.State, event.Event)
	return fsm.State{"Opened"}, nil
}

type LockWithNoKeyError struct {
	State string // current state
	Event string
	Err   error
}

func (e *LockWithNoKeyError) Error() string {
	return e.Err.Error() + ": State " + e.State + " Event " + e.Event + " with no key"
}

func (e *LockWithNoKeyError) Unwrap() error { return e.Err }

func LockDoor(data interface{}, event fsm.Event, userData interface{}) (fsm.State, error) {
	door := data.(*Door)
	entry := door.fmsEntry
	key := userData.(*Key)

	if key != nil {
		fmt.Printf("%s: State=%s, Event=%s, Action=LockDoor, Key=%s\n", door.name, entry.State, event.Event, key.id)

		// fsm state can be changed inside handle, according to userData
		if key.id == "root" {
			return fsm.State{"Locked"}, nil
		}
		return entry.State, nil
	} else {
		fmt.Printf("%s: State=%s, Event=%s, Action=LockDoor, Oops\n", door.name, entry.State, event.Event)
		return fsm.State{}, &LockWithNoKeyError{State: entry.State.State,
			Event: event.Event, Err: fsmerror.ErrInvalidEvent}
	}
}

func main() {
	user := flag.String("k", "", "key id")
	flag.Parse()

	d := fsm.FSMDesc{
		InitState: "Closed",
		LogMax:    20,
		States: fsm.StateDesc{
			{
				State: "Closed",
				Events: fsm.EventDesc{
					{Event: "Open", Handle: OpenDoor, Candidates: []string{"Opened"}},
					{Event: "Lock", Handle: LockDoor, Candidates: []string{"Closed", "Locked"}},
				},
			},
		},
	}

	fsmCtl, err := fsm.New(d)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}

	door := &Door{name: "myDoor"}
	e, err := fsmCtl.NewEntry(door)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}
	door.fmsEntry = e

	var key *Key = nil
	if *user != "" {
		key = &Key{id: *user}
	}
	state, err := e.DoFSMwithData("Lock", key, true)
	if err != nil {
		if errors.Is(err, fsmerror.ErrInvalidEvent) {
			fmt.Printf("ERROR: %s\n", err.Error())
		} else if errors.Is(err, fsmerror.ErrHandleNotExists) {
			fmt.Printf("ERROR: %s\n", err.Error())
		} else {
			fmt.Printf("ERROR: %s\n", err.Error())
		}
	} else {
		fmt.Printf("%s Next State: %s\n", door.name, state.State)
	}

	e.PrintLog(0)
}
