goFSM
-----
Finite State Machine(FSM) for go

- [import in your project](#import-in-your-project)
- [example code](#example-code)

# import in your project
```go
import (
    "github.com/HaesungSeo/goFSM/v2"
)
```

# example code
from example/door/door.go
```go
package main

import (
	"errors"
	"flag"
	"fmt"

	fsm "github.com/HaesungSeo/goFSM/v2"
	fsmerror "github.com/HaesungSeo/goFSM/v2/internal/fsmerrors"
)

// ////////////////////////////////////////////////
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

// 3) define Callback functions
func OpenDoor(door *Door, event fsm.Event, _ *Key) (fsm.HandleRetCode, error) {
	entry := door.entry

	fmt.Printf("Door %s: State=%s, Event=%s, Action=OpenDoor\n",
		door.name, entry.State, event.Name)
	return fsm.ExitOK, nil
}

type NoKeyError struct {
	State string // current state
	Event string // input event
	Err   error
}

func (e *NoKeyError) Error() string {
	return e.Err.Error() + ": State " + e.State + " Event " + e.Event + " No key"
}

func (e *NoKeyError) Unwrap() error { return e.Err }

func LockDoor(door *Door, event fsm.Event, key *Key) (fsm.HandleRetCode, error) {
	entry := door.entry
	if key != nil {
		fmt.Printf("Door %s: State=%s, Event=%s, Key=%s, Action=LockDoor\n",
			door.name, entry.State, event.Name, key.id)
		return fsm.ExitOK, nil
	}

	fmt.Printf("Door %s: State=%s, Event=%s, NOKEY Action=LockDoor\n",
		door.name, entry.State, event.Name)
	return fsm.ExitFail, &NoKeyError{
		State: entry.State.Name,
		Event: event.Name,
		Err:   fsmerror.ErrInvalidUserData,
	}
}

func main() {
	user := flag.String("k", "", "key id")
	flag.Parse()

	// 4) define FSM descriptor
	d := &fsm.TableDesc[*Door, *Key]{
		InitState:   "Closed",
		FinalStates: []string{"Opned", "Closed", "Locked"},
		LogMax:      20,
		States: []fsm.StateDesc[*Door, *Key]{
			{
				State: "Closed",
				Events: []fsm.EventDesc[*Door, *Key]{
					{Event: "Open", Func: OpenDoor, CandList: []string{"Opened", "Closed"}},
					{Event: "Lock", Func: LockDoor, CandList: []string{"Locked", "Closed"}},
				},
			},
			{
				State: "Opened",
				Events: []fsm.EventDesc[*Door, *Key]{
					{Event: "Open", Func: OpenDoor, CandMap: map[fsm.HandleRetCode]string{
						fsm.ExitOK:   "Opened",
						fsm.ExitFail: "Opened",
					}},
					{Event: "Lock", Func: LockDoor, CandMap: map[fsm.HandleRetCode]string{
						fsm.ExitOK:   "Opened",
						fsm.ExitFail: "Opened",
					}},
				},
			},
		},
	}

	// 5) define FSM Instance
	fsmCtl, err := fsm.NewTable(d)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}

	// 6) define FSM Entry
	door := Door{name: "myDoor"}
	door.entry = fsmCtl.NewEntry(&door)

	// 7) Transit() !
	var key *Key = nil
	if *user != "" {
		key = &Key{id: *user}
	}
	isRun := true
	for isRun {
		state, eot, err := door.entry.TransitWithData("Lock", key)
		if err != nil {
			switch {
			case errors.Is(err, fsmerror.ErrInvalidEvent):
				fmt.Printf("ERROR1: %s\n", err.Error())
			case errors.Is(err, fsmerror.ErrHandleNotExists):
				fmt.Printf("ERROR2: %s\n", err.Error())
			default:
				fmt.Printf("ERROR3: %s\n", err.Error())
			}
			break
		} else {
			fmt.Printf("### %s Next State: %s\n", door.name, state.Name)
		}
		if eot {
			isRun = false
		}
	}
	door.entry.PrintLog(0)
}
```

execute result
```bash
$ cd door/
$ ls
door.go
$ go build
$ ls
door  door.go
$ ./door
Door myDoor: State={Closed}, Event=Lock, NOKEY Action=LockDoor
ERROR3: invalid userdata: State Closed Event Lock No key
2023-01-09 17:13:02 KST State=[Closed] Event=[Lock] Func=[main.LockDoor] NextState=[Closed] Err=[invalid userdata: State Closed Event Lock No key]
$ ./door -h
Usage of ./door:
  -k string
    	key id
$ ./door -k any
Door myDoor: State={Closed}, Event=Lock, Key=any, Action=LockDoor
### myDoor Next State: Locked
2023-01-09 17:13:09 KST State=[Closed] Event=[Lock] Func=[main.LockDoor] NextState=[Locked]
$
```
