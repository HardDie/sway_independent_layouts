## sway_independent_layouts
Saves the keyboard layout for each launched application in the sway environment and automatically changes it when switching between applications.

Also if you use a **waybar** for widgets, you can set the **"signal": 3** for the language widget and it will be updated every time the daemon switches the layout.
Example of waybar config:
```
{
	"output": "HDMI",
	"modules-left": ["sway/workspaces", "sway/mode"],
	"modules-right": ["layout"],
	"layout": {
		"exec": "layout_widget",
		"interval": 1,
		"signal": 3
	}
}
```

## How to install
```
go install github.com/HardDie/sway_independent_layouts
```

## How to run
Run this application as a daemon.
```
export PATH=$PATH:$HOME/go/bin
sway_independent_layouts&
```
