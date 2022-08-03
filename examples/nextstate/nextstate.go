package main

import (
	"errors"
	"flag"
	"fmt"

	fsm "github.com/HaesungSeo/goFSM/internal/fsm"
	fsmerror "github.com/HaesungSeo/goFSM/internal/fsmerrors"
)

type Door struct {
	entry *fsm.Entry
	name  string
}

type Key struct {
	id string
}

func OpenDoor(owner interface{}, event fsm.Event, _ interface{}) (fsm.State, bool, error) {
	door := owner.(*Door)
	entry := door.entry
	fmt.Printf("%s: State=%s, Event=%s, Action=OpenDoor\n", door.name, entry.State, event.Name)
	return fsm.State{"Opened"}, true, nil
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

func LockDoor(data interface{}, event fsm.Event, userData interface{}) (fsm.State, bool, error) {
	door := data.(*Door)
	entry := door.entry
	key := userData.(*Key)

	if key != nil {
		fmt.Printf("%s: State=%s, Event=%s, Action=LockDoor, Key=%s\n", door.name, entry.State, event.Name, key.id)

		// fsm state can be changed inside handle, according to userData
		if key.id == "root" {
			return fsm.State{"Locked"}, true, nil
		}
		return entry.State, true, nil
	} else {
		fmt.Printf("%s: State=%s, Event=%s, Action=LockDoor, Oops\n", door.name, entry.State, event.Name)
		return fsm.State{}, true, &LockWithNoKeyError{State: entry.State.Name,
			Event: event.Name, Err: fsmerror.ErrInvalidEvent}
	}
}

func main() {
	user := flag.String("k", "", "key id")
	flag.Parse()

	d := fsm.TableDesc{
		InitState: "Closed",
		LogMax:    20,
		States: []fsm.StateDesc{
			{
				State: "Closed",
				Events: []fsm.EventDesc{
					{Event: "Open", Func: OpenDoor, Candidates: []string{"Opened"}},
					{Event: "Lock", Func: LockDoor, Candidates: []string{"Closed", "Locked"}},
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
	e := fsmCtl.NewEntry(door)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}
	door.entry = e

	var key *Key = nil
	if *user != "" {
		key = &Key{id: *user}
	}
	state, _, err := e.TransitWithData("Lock", key, true)
	if err != nil {
		if errors.Is(err, fsmerror.ErrInvalidEvent) {
			fmt.Printf("ERROR: %s\n", err.Error())
		} else if errors.Is(err, fsmerror.ErrHandleNotExists) {
			fmt.Printf("ERROR: %s\n", err.Error())
		} else {
			fmt.Printf("ERROR: %s\n", err.Error())
		}
	} else {
		fmt.Printf("%s Next State: %s\n", door.name, state.Name)
	}

	e.PrintLog(0)
}
