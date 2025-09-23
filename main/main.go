package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"time"
	"weilong/connection"
	"weilong/controller"
	"weilong/cube"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type Config struct {
	MACAddress string `json:"mac_address"`
	Model      string `json:"model"`
}

const (
	configFile      = "config.json"
	appSettingsFile = "app_settings.json"
)

func saveConfig(config Config) error {
	file, err := os.Create(configFile)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	return encoder.Encode(config)
}

func loadConfig() (Config, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	var config Config
	err = decoder.Decode(&config)
	return config, err
}

type AppSettings struct {
	ABind       string `json:"a_bind"`
	AHold       int    `json:"a_hold"`
	UpdateState bool   `json:"update_state"`
	UpdateDelay int    `json:"update_delay"`
}

func saveAppSettings(settings AppSettings) error {
	file, err := os.Create(appSettingsFile)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	return encoder.Encode(settings)
}

func loadAppSettings() (AppSettings, error) {
	file, err := os.Open(appSettingsFile)
	if err != nil {
		return AppSettings{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	var settings AppSettings
	err = decoder.Decode(&settings)
	return settings, err
}

var doUpdate bool
var updateDelay time.Duration

// My cube's mac:        CF:30:16:00:DE:1D

// 		CURRENT PROBLEMS
// * After pressing "Disconnect" something is not being reset, causing duplicating of request on next conenction
// (that is why app closes on button press)
// * Base of the uinput controller emulation module is done, but it is unfinished
// * There should be a pop up if cube is disconencted
// * Something strange with connection: (Part of log)
//  2025/03/14 14:01:38 Error sending data: In Progress
//  2025/03/14 14:01:38 Retrying in 100 ms
//  2025/03/14 14:01:39 Still failed: In Progress
//  Battery level: 86 <- means, that connected
// (Error reported, but still connected)

func main() {
	cube_ := cube.Cube{}
	myApp := app.New()
	w := myApp.NewWindow("Smart Cube App")
	macEntry := widget.NewEntry()
	macEntry.SetPlaceHolder("MAC-адрес")
	errorLabel := widget.NewLabel("")
	modelRadio := widget.NewRadioGroup([]string{"Weilong V10 AI"}, func(s string) {})
	config, err := loadConfig()
	if err == nil {
		macEntry.SetText(config.MACAddress)
		modelRadio.SetSelected(config.Model)
	}
	var showStartScreen func()
	showStartScreen = func() {
		connectButton := widget.NewButton("Connect", func() {
			if modelRadio.Selected == "Weilong V10 AI" {
				cube_.Type = cube.WeilongV10AI
			}
			err := cube_.FindAndConnect(macEntry.Text)
			if err != nil {
				errorLabel.SetText(err.Error())
				errorLabel.Refresh()
				return
			}
			saveConfig(Config{
				MACAddress: macEntry.Text,
				Model:      modelRadio.Selected,
			})
			
			c := controller.Controller{}
			if err = c.Init(); err != nil {log.Println(err)}
			
			doUpdate = false
			options := []string{"R", "R'", "L", "L'", "U", "U'", "D", "D'", "F", "F'", "B", "B'"}
			
			// Complexes of button settings
			btnALabel := widget.NewLabel("A Button")
			btnABind := widget.NewSelect(options, func(option string) {})
			holdATime := widget.NewSlider(40, 150)
			holdATime.Step = 1
			holdATime.Value = 50
			holdALabel := widget.NewLabel("Hold(ms): 50")
			holdATime.OnChangeEnded = func(value float64) {
				holdALabel.SetText("Hold(ms): " + strconv.Itoa(int(value)))
				holdATime.Refresh()
				*cube.GetCallback(options[btnABind.SelectedIndex()]) = func() {c.PressA(time.Duration(int(value)))}
			}
			btnABind.OnChanged = func(option string) {*cube.GetCallback(option) = func() {c.PressA(time.Duration(int(holdATime.Value)))}}
			btnAComplex := container.NewHBox(btnALabel, btnABind, holdALabel, holdATime)
			
			btnBLabel := widget.NewLabel("B Button")
		    btnBBind := widget.NewSelect(options, func(option string) {})
		    holdBTime := widget.NewSlider(40, 150)
		    holdBTime.Step = 1
		    holdBTime.Value = 50
		    holdBLabel := widget.NewLabel("Hold(ms): 50")
		    holdBTime.OnChangeEnded = func(value float64) {
		        holdBLabel.SetText("Hold(ms): " + strconv.Itoa(int(value)))
		        holdBTime.Refresh()
		        *cube.GetCallback(options[btnBBind.SelectedIndex()]) = func() { c.PressB(time.Duration(int(value))) }
		    }
		    btnBBind.OnChanged = func(option string) { *cube.GetCallback(option) = func() { c.PressB(time.Duration(int(holdBTime.Value))) } }
		    btnBComplex := container.NewHBox(btnBLabel, btnBBind, holdBLabel, holdBTime)
		
		    btnXLabel := widget.NewLabel("X Button")
		    btnXBind := widget.NewSelect(options, func(option string) {})
		    holdXTime := widget.NewSlider(40, 150)
		    holdXTime.Step = 1
		    holdXTime.Value = 50
		    holdXLabel := widget.NewLabel("Hold(ms): 50")
		    holdXTime.OnChangeEnded = func(value float64) {
		        holdXLabel.SetText("Hold(ms): " + strconv.Itoa(int(value)))
		        holdXTime.Refresh()
		        *cube.GetCallback(options[btnXBind.SelectedIndex()]) = func() { c.PressX(time.Duration(int(value))) }
		    }
		    btnXBind.OnChanged = func(option string) { *cube.GetCallback(option) = func() { c.PressX(time.Duration(int(holdXTime.Value))) } }
		    btnXComplex := container.NewHBox(btnXLabel, btnXBind, holdXLabel, holdXTime)
		
		    btnYLabel := widget.NewLabel("Y Button")
		    btnYBind := widget.NewSelect(options, func(option string) {})
		    holdYTime := widget.NewSlider(40, 150)
		    holdYTime.Step = 1
		    holdYTime.Value = 50
		    holdYLabel := widget.NewLabel("Hold(ms): 50")
		    holdYTime.OnChangeEnded = func(value float64) {
		        holdYLabel.SetText("Hold(ms): " + strconv.Itoa(int(value)))
		        holdYTime.Refresh()
		        *cube.GetCallback(options[btnYBind.SelectedIndex()]) = func() { c.PressY(time.Duration(int(value))) }
		    }
		    btnYBind.OnChanged = func(option string) { *cube.GetCallback(option) = func() { c.PressY(time.Duration(int(holdYTime.Value))) } }
		    btnYComplex := container.NewHBox(btnYLabel, btnYBind, holdYLabel, holdYTime)


			solvedCheck := widget.NewLabel("Solved: true")
			batteryLevel := widget.NewLabel("Battery: 100%")
			last5moves := widget.NewLabel("Last 5 moves: ")
			leftContent := container.NewVBox(
				widget.NewLabel(modelRadio.Selected),
				batteryLevel,
				solvedCheck,
				last5moves,
			)

			var toggleInfoUpdater func()
			toggleInfoUpdater = func() {
				if !doUpdate {
					doUpdate = true
					solvedCheck.Show()
					batteryLevel.Show()
					go func() {
						for {
							if !doUpdate {
								break
							}
							cube_.UpdateState()
							cube_.UpdatePowerInfo()
							solved := "true"
							if !cube_.IsSolved() {
								solved = "false"
							}
							batteryLevel.SetText("Battery: " + strconv.Itoa(cube_.Power) + "%")
							solvedCheck.SetText("Solved: " + solved)
							batteryLevel.Refresh()
							solvedCheck.Refresh()
							time.Sleep(updateDelay * time.Second)
						}
					}()
				} else {
					doUpdate = false
					solvedCheck.Hide()
					batteryLevel.Hide()
				}
			}
			
			
			// creating update settings
			updateDelayCheck := widget.NewCheck("Update cube state", func(bal bool) { toggleInfoUpdater() })
			updateDelayCheck.SetChecked(false)
			solvedCheck.Hide()
			batteryLevel.Hide()
			updateDelaySlider := widget.NewSlider(1, 60)
			updateDelaySlider.Value = 30
			updateDelay = 30
			updateDelaySlider.Step = 1
			updateDelaySliderLabel := widget.NewLabel("Delay(s): 30")
			updateDelaySlider.OnChangeEnded = func(val float64) {
				updateDelay = time.Duration(val)
				updateDelaySliderLabel.SetText("Delay(s): " + strconv.Itoa(int(val)))
				updateDelaySliderLabel.Refresh()
			}

			// apply saved data
			settings, err := loadAppSettings()
			if err != nil {
				log.Println("Error loading settings:", err)
			}
			btnABind.SetSelected(settings.ABind)
			holdATime.Value = float64(settings.AHold)
			holdALabel.SetText("Hold(ms): " + strconv.Itoa(settings.AHold))
			updateDelayCheck.SetChecked(settings.UpdateState)
			if settings.UpdateState && !doUpdate {toggleInfoUpdater()}
			updateDelaySlider.Value = float64(settings.UpdateDelay)
			updateDelay = time.Duration(settings.UpdateDelay)
			updateDelaySliderLabel.SetText("Delay(s): " + strconv.Itoa(settings.UpdateDelay))

			rightContent := container.NewVBox(
				container.NewHBox(
					widget.NewLabel("Control Panel"),
					widget.NewButton("Save Settings", func() {
						settings := AppSettings{
							ABind:       btnABind.Selected,
							AHold:       int(holdATime.Value),
							UpdateState: updateDelayCheck.Checked,
							UpdateDelay: int(updateDelaySlider.Value),
						}
						if err := saveAppSettings(settings); err != nil {
							log.Println("Error saving settings:", err)
						}
					}),
				),
				widget.NewButton("Disconnect(Exit)", func() {
					err := connection.Disconnect()
					if err != nil {
						errorLabel.SetText(err.Error())
						log.Println(err)
					} else {
						errorLabel.SetText("")
					}
					doUpdate = false
					myApp.Quit()
					showStartScreen()
				}),
				btnAComplex,
				btnBComplex,
				btnXComplex,
				btnYComplex,
				container.NewHBox(
					updateDelayCheck,
					updateDelaySliderLabel,
					updateDelaySlider,
				),
			)
			split := container.NewHSplit(leftContent, rightContent)
			w.SetContent(split)
			w.Resize(fyne.NewSize(800, 600))

			cube_.GreetCube()
			time.Sleep(100 * time.Millisecond)
			go func() {
				for {
					last5moves.SetText("Last 5 moves: " + cube.GetLastMoves())
					last5moves.Refresh()
					time.Sleep(100 * time.Millisecond)
				}
			}()
		})

		content := container.NewVBox(
			macEntry,
			modelRadio,
			connectButton,
			errorLabel,
		)
		w.SetContent(content)
		w.Resize(fyne.NewSize(350, 300))
	}
	showStartScreen()
	w.ShowAndRun()
}
