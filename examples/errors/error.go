package main

import (
	"errors"
	"fmt"

	fsm "github.com/HaesungSeo/goFSM"
	fsmerror "github.com/HaesungSeo/goFSM/internal/fsmerrors"
)

type Door struct {
	fmsEntry *fsm.FSMEntry
	name     string
}

//func OpenDoor(data interface{}, event fsm.Event) (fsm.State, error) {
func OpenDoor(data interface{}, event fsm.Event) (fsm.State, error) {
	door := data.(*Door)
	entry := door.fmsEntry
	fmt.Printf("%s: State=%s, Event=%s, Action=OpenDoor\n", door.name, entry.State, event.Event)
	return fsm.State{"Opened"}, nil
}

func main() {
	d := fsm.FSMDesc{
		InitState: "Closed",
		LogMax:    20,
		States: fsm.StateDesc{
			{
				State: "Closed",
				Events: fsm.EventDesc{
					{Event: "Open", Handle: OpenDoor, Candidates: []string{"Opened"}},
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

	// invalid event error
	state, err := e.DoFSM("lock", true)
	if err != nil {
		if errors.Is(err, fsmerror.ErrEvent) {
			fmt.Printf("Event ERROR: %s\n", err.Error())
		} else if errors.Is(err, fsmerror.ErrState) {
			fmt.Printf("State ERROR: %s\n", err.Error())
		} else if errors.Is(err, fsmerror.ErrHandle) {
			fmt.Printf("Handle ERROR: %s\n", err.Error())
		} else {
			fmt.Printf("ERROR: %s\n", err.Error())
		}
	} else {
		fmt.Printf("%s new state: %s\n", door.name, state.State)
	}

	// Closed -> Opened
	e.DoFSM("Open", true)

	// Opened -> Opened
	state, err = e.DoFSM("Open", true)
	if err != nil {
		if errors.Is(err, fsmerror.ErrEvent) {
			fmt.Printf("Event ERROR: %s\n", err.Error())
		} else if errors.Is(err, fsmerror.ErrState) {
			fmt.Printf("State ERROR: %s\n", err.Error())
		} else if errors.Is(err, fsmerror.ErrHandle) {
			fmt.Printf("Handle ERROR: %s\n", err.Error())
		} else {
			fmt.Printf("ERROR: %s\n", err.Error())
		}
	} else {
		fmt.Printf("%s new state: %s\n", door.name, state.State)
	}

	e.PrintLog(0)
}