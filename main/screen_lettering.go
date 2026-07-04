package main

import (
	"context"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const letterCellPx = 46

type stickerCell struct {
	widget.BaseWidget
	key      string
	fill     color.Color
	txtColor color.Color
	editable bool
	selected bool
	onTap    func(string)

	bg  *canvas.Rectangle
	txt *canvas.Text
}

func newStickerCell(key string, fill, txtColor color.Color, symbol string, editable bool, onTap func(string)) *stickerCell {
	c := &stickerCell{key: key, fill: fill, txtColor: txtColor, editable: editable, onTap: onTap}
	c.bg = canvas.NewRectangle(fill)
	c.bg.CornerRadius = 6
	c.txt = canvas.NewText(symbol, txtColor)
	c.txt.Alignment = fyne.TextAlignCenter
	c.txt.TextStyle = fyne.TextStyle{Bold: true}
	c.txt.TextSize = 18
	c.applyBorder()
	c.ExtendBaseWidget(c)
	return c
}

func (c *stickerCell) applyBorder() {
	if c.selected {
		c.bg.StrokeColor = accentColor
		c.bg.StrokeWidth = 3
	} else {
		c.bg.StrokeColor = borderColor
		c.bg.StrokeWidth = 1
	}
}

func (c *stickerCell) Tapped(*fyne.PointEvent) {
	if c.editable && c.onTap != nil {
		c.onTap(c.key)
	}
}

func (c *stickerCell) SetSymbol(s string) {
	c.txt.Text = s
	c.txt.Refresh()
}

func (c *stickerCell) SetSelected(b bool) {
	c.selected = b
	c.applyBorder()
	c.bg.Refresh()
}

func (c *stickerCell) CreateRenderer() fyne.WidgetRenderer {
	content := container.NewGridWrap(
		fyne.NewSize(letterCellPx, letterCellPx),
		container.NewStack(c.bg, container.NewCenter(c.txt)),
	)
	return widget.NewSimpleRenderer(content)
}

func (a *App) showLettering() {
	a.switchScreen(fyne.NewSize(760, 760), func(ctx context.Context) fyne.CanvasObject {
		profiles := loadLetteringProfiles()
		active := profiles.Active
		scheme := profiles.Profiles[active]

		persist := func() {
			profiles.Profiles[active] = scheme
			writeJSON(letteringProfilesFile, profiles)
		}

		cells := map[string]*stickerCell{}
		selected := ""

		symbolEntry := widget.NewEntry()
		symbolEntry.SetPlaceHolder("symbol")
		selectedLabel := caption("Select a sticker to edit")

		selectCell := func(key string) {
			if selected != "" {
				if c, ok := cells[selected]; ok {
					c.SetSelected(false)
				}
			}
			selected = key
			if c, ok := cells[key]; ok {
				c.SetSelected(true)
			}
			selectedLabel.Text = "Editing " + key[:1] + " sticker"
			selectedLabel.Refresh()
			symbolEntry.SetText(scheme[key])
			a.window.Canvas().Focus(symbolEntry)
		}

		symbolEntry.OnChanged = func(s string) {
			if selected == "" {
				return
			}
			scheme[selected] = s
			if c, ok := cells[selected]; ok {
				c.SetSymbol(s)
			}
		}

		faceWidget := func(face string) fyne.CanvasObject {
			objs := make([]fyne.CanvasObject, 9)
			for idx := 0; idx < 9; idx++ {
				if idx == 4 {
					center := newStickerCell(face+"c", faceFill[face], faceTextColor(face), "", false, nil)
					objs[idx] = center
					continue
				}
				key := stickerKey(face, idx)
				cell := newStickerCell(key, faceFill[face], faceTextColor(face), scheme[key], true, selectCell)
				cells[key] = cell
				objs[idx] = cell
			}
			return container.NewGridWrap(
				fyne.NewSize(letterCellPx*3, letterCellPx*3),
				container.NewGridWithColumns(3, objs...),
			)
		}

		blank := func() fyne.CanvasObject {
			r := canvas.NewRectangle(color.RGBA{0, 0, 0, 0})
			return container.NewGridWrap(fyne.NewSize(letterCellPx*3, letterCellPx*3), r)
		}

		net := container.NewVBox(
			container.NewHBox(blank(), faceWidget("U"), blank(), blank()),
			container.NewHBox(faceWidget("L"), faceWidget("F"), faceWidget("R"), faceWidget("B")),
			container.NewHBox(blank(), faceWidget("D"), blank(), blank()),
		)

		save := widget.NewButton("Save", persist)
		save.Importance = widget.HighImportance

		profileSel := widget.NewSelect(profiles.names(), func(name string) {
			if name == active {
				return
			}
			persist()
			profiles.Active = name
			writeJSON(letteringProfilesFile, profiles)
			a.showLettering()
		})
		profileSel.Selected = active

		newBtn := widget.NewButton("New", func() {
			nameEntry := widget.NewEntry()
			nameEntry.SetPlaceHolder("Scheme name")
			dialog.ShowForm("New scheme", "Create", "Cancel",
				[]*widget.FormItem{widget.NewFormItem("Name", nameEntry)},
				func(ok bool) {
					name := nameEntry.Text
					if !ok || name == "" {
						return
					}
					persist()
					profiles.Profiles[name] = LetteringScheme{}
					profiles.Active = name
					writeJSON(letteringProfilesFile, profiles)
					a.showLettering()
				}, a.window)
		})
		delBtn := widget.NewButton("Delete", func() {
			if len(profiles.Profiles) <= 1 {
				return
			}
			delete(profiles.Profiles, active)
			for name := range profiles.Profiles {
				profiles.Active = name
				break
			}
			writeJSON(letteringProfilesFile, profiles)
			a.showLettering()
		})
		delBtn.Importance = widget.DangerImportance

		header := container.NewBorder(nil, nil,
			heading("Lettering scheme", 24),
			container.NewHBox(caption("Profile"), profileSel, newBtn, delBtn),
		)

		editRow := container.NewBorder(nil, nil, selectedLabel, nil, symbolEntry)

		bottom := container.NewVBox(
			container.NewPadded(editRow),
			container.NewPadded(container.NewHBox(save, widget.NewButton("Back", a.showBlind))),
		)

		return container.NewPadded(container.NewBorder(
			container.NewPadded(header),
			bottom,
			nil, nil,
			container.NewScroll(container.NewCenter(net)),
		))
	})
}
