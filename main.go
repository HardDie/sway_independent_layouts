package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os/exec"

	"github.com/joshuarubin/go-sway"
)

type (
	// Pair (input dev - layout id)  map[input.Identifier]input.XKBActiveLayoutIndex
	InputsInfo map[string]int64

	Runtime struct {
		// ID of previous active window
		PreviousContainerId int64
		// Collection of layouts for each window
		InputsCollection map[int64]InputsInfo
	}

	Handler struct {
		sway.EventHandler
		client sway.Client
	}
)

var (
	Debug *bool
)

func GetInputs(ctx context.Context, client sway.Client) (res InputsInfo) {
	// Get all active inputs
	inputs, err := client.GetInputs(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Backup inputs settings
	res = make(InputsInfo)
	for _, val := range inputs {
		if val.XKBActiveLayoutIndex != nil {
			res[val.Identifier] = *val.XKBActiveLayoutIndex
		}
	}
	return
}

func SetInputs(ctx context.Context, client sway.Client, layout InputsInfo) error {
	cmd := "input %s xkb_switch_layout %d"
	for Identifier, XKBActiveLayoutIndex := range layout {
		_, err := client.RunCommand(ctx, fmt.Sprintf(cmd, Identifier, XKBActiveLayoutIndex))
		if err != nil {
			return err
		}
	}
	return nil
}

// Send signal to waybar and update layout widget
func UpdateBarStatus() error {
	_, err := exec.Command("pkill", "-SIGRTMIN+3", "waybar").Output()
	return err
}

func Deref[T any](val *T) T {
	var res T
	if val == nil {
		return res
	}
	return *val
}

func (h Handler) Window(ctx context.Context, e sway.WindowEvent) {
	run := ctx.Value("runtime").(*Runtime)

	if *Debug {
		defer func() {
			data, err := json.MarshalIndent(run.InputsCollection, "", "	")
			if err == nil {
				log.Println("[INFO] Dump of all layouts: " + string(data))
			}
		}()
	}

	switch e.Change {
	case "focus":
		if *Debug {
			log.Printf("[INFO] Got focus event. ID: %v, Name: %v, app_id: %v, pid: %v\n",
				e.Container.ID, e.Container.Name, Deref(e.Container.AppID), Deref(e.Container.PID))
		}
		defer func() {
			// Change previous id
			run.PreviousContainerId = e.Container.ID
		}()

		// Save current layout for previous window
		if run.PreviousContainerId != 0 {
			// Get inputs layout
			inputsMap := GetInputs(ctx, h.client)

			if *Debug {
				data, err := json.MarshalIndent(inputsMap, "", "	")
				if err == nil {
					log.Printf("[INFO] Save layout for %v container: "+string(data), run.PreviousContainerId)
				} else {
					log.Printf("[INFO] Save layout for %v container (error marshal)", run.PreviousContainerId)
				}
			}
			// Save layout for previous window
			run.InputsCollection[run.PreviousContainerId] = inputsMap
		}

		// Get layout for current window
		if layout, ok := run.InputsCollection[e.Container.ID]; ok {
			if *Debug {
				data, err := json.MarshalIndent(layout, "", "	")
				if err == nil {
					log.Printf("[INFO] Restore layout for %v container: "+string(data), e.Container.ID)
				} else {
					log.Printf("[INFO] Restore layout for %v container (error marshal)", e.Container.ID)
				}
			}

			// Update active layout
			if err := SetInputs(ctx, h.client, layout); err != nil {
				log.Println("Error set layout:", err.Error())
			}

			// Update bar
			if err := UpdateBarStatus(); err != nil {
				log.Println("Error signal:", err.Error())
			}
		}
	case "close":
		if *Debug {
			log.Printf("[INFO] Got close event. ID: %v, Name: %v, app_id: %v, pid: %v\n",
				e.Container.ID, e.Container.Name, Deref(e.Container.AppID), Deref(e.Container.PID))
		}
		// Remove app from cache
		if _, ok := run.InputsCollection[e.Container.ID]; ok {
			delete(run.InputsCollection, e.Container.ID)
		}
		run.PreviousContainerId = 0
	}
}

func main() {
	Debug = flag.Bool("debug", false, "print debug messages")
	flag.Parse()

	if *Debug {
		log.SetFlags(log.Lshortfile | log.Ltime)
		log.Println("Debug enabled")
	}

	ctx := context.WithValue(context.Background(), "runtime", &Runtime{
		InputsCollection: make(map[int64]InputsInfo),
	})

	client, err := sway.New(ctx)
	if err != nil {
		log.Fatal(err)
	}

	th := Handler{
		EventHandler: sway.NoOpEventHandler(),
		client:       client,
	}

	if err = sway.Subscribe(ctx, th, sway.EventTypeWindow); err != nil {
		log.Fatal(err)
	}
}
