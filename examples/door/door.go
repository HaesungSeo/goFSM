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

func OpenDoor(n *fsm.FSMEntry, e fsm.Event) (fsm.State, error) {
	door := n.Head.(*Door)
	fmt.Printf("%s: State=%s, Event=%s, Action=OpenDoor\n", door.name, n.State, e.Event)
	return fsm.State{"Opened"}, nil
}

func CloseDoor(n *fsm.FSMEntry, e fsm.Event) (fsm.State, error) {
	door := n.Head.(*Door)
	fmt.Printf("%s: State=%s, Event=%s, Action=CloseDoor\n", door.name, n.State, e.Event)
	return fsm.State{"Closed"}, nil
}

func LockDoor(n *fsm.FSMEntry, e fsm.Event) (fsm.State, error) {
	door := n.Head.(*Door)
	fmt.Printf("%s: State=%s, Event=%s, Action=LockDoor\n", door.name, n.State, e.Event)
	return fsm.State{"Locked"}, nil
}

func UnlockDoor(n *fsm.FSMEntry, e fsm.Event) (fsm.State, error) {
	door := n.Head.(*Door)
	fmt.Printf("%s: State=%s, Event=%s, Action=UnlockDoor\n", door.name, n.State, e.Event)
	return fsm.State{"Closed"}, nil
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
					{Event: "Lock", Handle: LockDoor, Candidates: []string{"Closed", "Locked"}},
				},
			},
			{
				State: "Opened",
				Events: fsm.EventDesc{
					{Event: "Close", Handle: CloseDoor, Candidates: []string{"Closed"}},
				},
			},
			{
				State: "Locked",
				Events: fsm.EventDesc{
					{Event: "Unlock", Handle: UnlockDoor, Candidates: []string{"Closed", "Locked"}},
				},
			},
		},
	}

	fsmCtl, err := fsm.New(d, 0)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}
	fsmCtl.DumpTable()

	door := &Door{name: "myDoor"}
	e, err := fsmCtl.NewEntry(door)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}
	door.fmsEntry = e

	state, err := e.DoFSM("Open", true)
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

	state, err = e.DoFSM("Closee", true)
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
