package main

import (
	"fmt"

	fsm "github.com/HaesungSeo/goFSM"
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

func CloseDoor(data interface{}, event fsm.Event) (fsm.State, error) {
	door := data.(*Door)
	entry := door.fmsEntry
	fmt.Printf("%s: State=%s, Event=%s, Action=CloseDoor\n", door.name, entry.State, event.Event)
	return fsm.State{"Closed"}, nil
}

func main() {
	d := fsm.FSMDesc{
		InitState: "Closed",
		States: fsm.StateDesc{
			{
				State: "Closed",
				Events: fsm.EventDesc{
					{Event: "Open", Handle: OpenDoor, Candidates: []string{"Opened"}},
				},
			},
			{
				State: "Opened",
				Events: fsm.EventDesc{
					{Event: "Close", Handle: CloseDoor, Candidates: []string{"Closed"}},
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

	_, err = e.DoFSM("Open", true)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
	}

	e.PrintLog(0)
}
