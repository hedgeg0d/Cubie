package main

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"time"

	"cubie/cube"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type Config struct {
	MACAddress string `json:"mac_address"`
	Model      string `json:"model"`
}

const configFile = "config.json"

func saveConfig(config Config) error {
	file, err := os.Create(configFile)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(config)
}

func loadConfig() (Config, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()
	var config Config
	err = json.NewDecoder(file).Decode(&config)
	return config, err
}

func main() {
	myApp := app.New()
	w := myApp.NewWindow("Cubie")

	macEntry := widget.NewEntry()
	macEntry.SetPlaceHolder("MAC address")
	errorLabel := widget.NewLabel("")
	modelRadio := widget.NewRadioGroup([]string{"Weilong V10 AI"}, func(string) {})

	if config, err := loadConfig(); err == nil {
		macEntry.SetText(config.MACAddress)
		modelRadio.SetSelected(config.Model)
	}

	var showConnectScreen func()
	var showSessionScreen func(c *cube.Cube)

	showSessionScreen = func(c *cube.Cube) {
		batteryLabel := widget.NewLabel("Battery: --%")
		movesLabel := widget.NewLabel("Last 5 moves: ")

		ctx, cancel := context.WithCancel(context.Background())

		c.OnMove = func(string) {
			movesLabel.SetText("Last 5 moves: " + c.LastMoves())
			movesLabel.Refresh()
		}

		disconnectButton := widget.NewButton("Disconnect", func() {
			cancel()
			c.OnMove = nil
			if err := c.Disconnect(); err != nil {
				errorLabel.SetText(err.Error())
			} else {
				errorLabel.SetText("")
			}
			showConnectScreen()
		})

		content := container.NewVBox(
			widget.NewLabel(string(modelRadio.Selected)),
			batteryLabel,
			movesLabel,
			disconnectButton,
		)
		w.SetContent(content)
		w.Resize(fyne.NewSize(400, 300))

		c.GreetCube()
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				c.UpdatePowerInfo()
				batteryLabel.SetText("Battery: " + strconv.Itoa(c.Power) + "%")
				batteryLabel.Refresh()
				select {
				case <-ctx.Done():
					return
				case <-time.After(30 * time.Second):
				}
			}
		}()
	}

	showConnectScreen = func() {
		connectButton := widget.NewButton("Connect", func() {
			var t cube.CubeType
			if modelRadio.Selected == "Weilong V10 AI" {
				t = cube.WeilongV10AI
			}
			c := cube.New(t)
			if err := c.FindAndConnect(macEntry.Text); err != nil {
				errorLabel.SetText(err.Error())
				return
			}
			errorLabel.SetText("")
			saveConfig(Config{MACAddress: macEntry.Text, Model: modelRadio.Selected})
			showSessionScreen(c)
		})

		content := container.NewVBox(macEntry, modelRadio, connectButton, errorLabel)
		w.SetContent(content)
		w.Resize(fyne.NewSize(350, 300))
	}

	showConnectScreen()
	w.ShowAndRun()
}
