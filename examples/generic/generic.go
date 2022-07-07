package main

import (
	"errors"
	"flag"
	"fmt"

	fsm "github.com/HaesungSeo/goFSM"
	fsmerror "github.com/HaesungSeo/goFSM/internal/fsmerrors"
)

//////////////////////////////////////////////////
// coding convention
// 1) define FSM event specific userData
type Key struct {
	id string
}

// 2) define FSM Entry Owner, MUST HAVE entry shape of `Entry[*OWNER, *USERDATA]`
type Door struct {
	name  string
	entry *fsm.Entry[*Door, *Key]
}

// 3) define Link function to be used inside NewEntry()
func mylinker(d *Door, e *fsm.Entry[*Door, *Key]) {
	d.entry = e
}

// 4) define Callback functions
func OpenDoor(door *Door, event fsm.Event, _ *Key) (fsm.State, error) {
	entry := door.entry

	fmt.Printf("Door %s: State=%s, Event=%s, Action=OpenDoor\n",
		door.name, entry.State, event.Name)
	return fsm.State{Name: "Opened"}, nil
}

func LockDoor(door *Door, event fsm.Event, key *Key) (fsm.State, error) {
	entry := door.entry
	if key != nil {
		fmt.Printf("Door %s: State=%s, Event=%s, Key=%s, Action=LockDoor\n",
			door.name, entry.State, event.Name, key.id)
	} else {
		fmt.Printf("Door %s: State=%s, Event=%s, NOKEY Action=LockDoor\n",
			door.name, entry.State, event.Name)

	}
	return fsm.State{Name: "Opened"}, nil
}

func main() {
	user := flag.String("k", "", "key id")
	flag.Parse()

	// 6) define FSM descriptor
	d := &fsm.TableDesc[*Door, *Key]{
		InitState: "Closed",
		LogMax:    20,
		States: []fsm.StateDesc[*Door, *Key]{
			{
				State: "Closed",
				Events: []fsm.EventDesc[*Door, *Key]{
					{Event: "Open", Func: OpenDoor, Candidates: []string{"Opened"}},
					{Event: "Lock", Func: LockDoor, Candidates: []string{"Closed"}},
				},
			},
		},
	}

	// 7) define FSM Instance
	fsmCtl, err := fsm.NewTable(d)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}

	// 8) define FSM Entry
	door := Door{name: "myDoor"}
	e, err := fsmCtl.NewEntry(&door, mylinker)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}

	// 9) Transit() !
	var key *Key = nil
	if *user != "" {
		key = &Key{id: *user}
	}
	state, err := e.TransitWithData("Lock", key)
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
