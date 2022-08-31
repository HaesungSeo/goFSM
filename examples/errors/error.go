package main

import (
	"errors"
	"fmt"

	fsm "github.com/HaesungSeo/goFSM/internal/fsm"
	fsmerror "github.com/HaesungSeo/goFSM/internal/fsmerrors"
)

type Door struct {
	entry *fsm.Entry
	name  string
}

//func OpenDoor(data interface{}, event fsm.Event) (fsm.State, error) {
func OpenDoor(owner interface{}, event fsm.Event, _ interface{}) (fsm.State, error) {
	door := owner.(*Door)
	entry := door.entry
	fmt.Printf("%s: State=%s, Event=%s, Action=OpenDoor\n", door.name, entry.State, event.Name)
	return fsm.State{"Opened"}, nil
}

func main() {
	d := fsm.TableDesc{
		InitState:   "Closed",
		FinalStates: []string{"Closed", "Opened"},
		LogMax:      20,
		States: []fsm.StateDesc{
			{
				State: "Closed",
				Events: []fsm.EventDesc{
					{Event: "Open", Func: OpenDoor, Candidates: []string{"Opened"}},
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

	// invalid event error
	state, _, err := e.Transit("lock")
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

	// Closed -> Opened
	e.Transit("Open")

	// Opened -> Opened
	state, _, err = e.Transit("Open")
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
