goFSM
-----
Finite State Machine(FSM) for go

- [import in your project](#import-in-your-project)
- [example code](#example-code)
  - [simple](#simple)
  - [generic](#generic)

# import in your project
```go
import (
    "github.com/HaesungSeo/goFSM"
)
```

# example code
## simple 
from example/door/door.go
```go
package main

import (
    "fmt"

    fsm "github.com/HaesungSeo/goFSM/internal/fsm"
)

type Door struct {
    entry *fsm.Entry
    name     string
}

func OpenDoor(owner interface{}, event fsm.Event, _ interface{}) (fsm.State, error) {
    door := owner.(*Door)
    entry := door.entry
    fmt.Printf("%s: State=%s, Event=%s, Action=OpenDoor\n", door.name, entry.State, event.Event)
    return fsm.State{"Opened"}, nil
}

func CloseDoor(owner interface{}, event fsm.Event, _ interface{}) (fsm.State, error) {
    door := owner.(*Door)
    entry := door.entry
    fmt.Printf("%s: State=%s, Event=%s, Action=CloseDoor\n", door.name, entry.State, event.Event)
    return fsm.State{"Closed"}, nil
}

func main() {
    d := fsm.TableDesc{
        InitState: "Closed",
        LogMax:    20,
        States: []fsm.StateDesc{
            {
                State: "Closed",
                Events: []fsm.EventDesc{
                    {Event: "Open", Func: OpenDoor, Candidates: []string{"Opened"}},
                },
            },
            {
                State: "Opened",
                Events: []fsm.EventDesc{
                    {Event: "Close", Func: CloseDoor, Candidates: []string{"Closed"}},
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
    door.entry = e

    _, err = e.Transit("Open", true)
    if err != nil {
        fmt.Printf("ERROR: %s\n", err.Error())
    }

    e.PrintLog(0)
}
```

execute result
```bash
$ ./test 
myDoor: State={Closed}, Event=Open, Action=OpenDoor
2022-07-04 18:28:46 KST State=[Closed] Event=[Open] Func=[main.OpenDoor] Return=true NextState=[Opened] Err=[]
$ 
```

NOTE: 
MUST link Owner Data with FSM Entry Data, After NewEntry()
```go
entry, _ := fsmCtl.NewEntry(door)
door.entry = entry
```

MUST TYPE CAST, inside callback function
```go
door := owner.(*Door)
```

use fsm generic version, to simplify above convension

## generic
from example/generic/generic.go
```go
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
        door.name, entry.State, event.Event)
    return fsm.State{State: "Opened"}, nil
}

func LockDoor(door *Door, event fsm.Event, key *Key) (fsm.State, error) {
    entry := door.entry
    if key != nil {
        fmt.Printf("Door %s: State=%s, Event=%s, Key=%s, Action=LockDoor\n",
            door.name, entry.State, event.Event, key.id)
    } else {
        fmt.Printf("Door %s: State=%s, Event=%s, NOKEY Action=LockDoor\n",
            door.name, entry.State, event.Event)

    }
    return fsm.State{State: "Opened"}, nil
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
        fmt.Printf("%s Next State: %s\n", door.name, state.State)
    }

    e.PrintLog(0)
}
```

execute result
```bash
$ ./generic
Door myDoor: State={Closed}, Event=Lock, NOKEY Action=LockDoor
ERROR: invalid next state: State Closed Event Lock nextState Opened
2022-07-06 16:14:04 KST State=[Closed] Event=[Lock] Func=[main.LockDoor] Return=false NextState=[Closed] Msg=[] Err=[invalid next state: State Closed Event Lock nextState Opened]
$ ./generic -k root
Door myDoor: State={Closed}, Event=Lock, Key=root, Action=LockDoor
ERROR: invalid next state: State Closed Event Lock nextState Opened
2022-07-06 16:14:10 KST State=[Closed] Event=[Lock] Func=[main.LockDoor] Return=false NextState=[Closed] Msg=[] Err=[invalid next state: State Closed Event Lock nextState Opened]
$
```

callbacks can have more clear data types with generic type<p>
simple version
```go
func OpenDoor(owner interface{}, event fsm.Event, userData interface{}) (fsm.State, error)
```

generic version
```go
func OpenDoor(door *Door, event fsm.Event, key *Key) (fsm.State, error)
```
