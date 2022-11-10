package main

import (
	"flag"
	"image"
	"image/color"
	"log"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/gesture"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

var iconSearch = mi(icons.ActionSearch)

type iconEntry struct {
	name  string
	icon  *widget.Icon
	click gesture.Click
}

// mi (stands for `must icon`) returns a new `*widget.Icon` for the given byte
// slice or panics on error. It's primarily used within `data.go` and
// abbreviating the name like this reduces that file size by about 6kb.
func mi(data []byte) *widget.Icon {
	ic, err := widget.NewIcon(data)
	if err != nil {
		panic(err)
	}
	return ic
}

type iconBrowser struct {
	searchInput    widget.Editor
	resultList     widget.List
	matchedIndices []int
}

func (ib *iconBrowser) layout(gtx C, th *material.Theme) D {
	paint.Fill(gtx.Ops, th.Bg)
	var rows []layout.Widget
	for i := 0; i < len(allEntries); i += 3 {
		var cells []layout.FlexChild
		for n := 0; n < 3; n++ {
			index := i + n
			cells = append(cells, layout.Flexed(1.0/3.0, func(gtx C) D {
				if index >= len(allEntries) {
					return D{}
				}
				icData := &allEntries[index]
				name := material.Body2(th, icData.name)
				name.Alignment = text.Middle
				return layout.Flex{Alignment: layout.Middle, Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						gtx.Constraints.Max.X = 48
						gtx.Constraints.Max.Y = 48
						return icData.icon.Layout(gtx, th.Fg)
					}),
					layout.Rigid(layout.Spacer{Height: 10}.Layout),
					layout.Rigid(name.Layout),
				)
			}))
		}
		rows = append(rows, func(gtx C) D {
			return layout.Inset{Top: 20, Bottom: 20}.Layout(gtx, func(gtx C) D {
				return layout.Flex{}.Layout(gtx, cells...)
			})
		})
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return D{} // TODO the search input
		}),
		layout.Flexed(1, func(gtx C) D {
			return material.List(th, &ib.resultList).Layout(gtx, len(rows), func(gtx C, i int) D {
				return rows[i](gtx)
			})
		}),
	)
}

func run(showFrameTimes bool) error {
	updates := make(chan any)

	win := app.NewWindow(
		app.Size(900, 800),
		app.Title("GioUI Icon Browser"),
	)
	win.Perform(system.ActionCenter)

	th := material.NewTheme(gofont.Collection())
	th.TextSize = 17
	th.Palette = material.Palette{
		Bg:         color.NRGBA{17, 21, 24, 255},
		Fg:         color.NRGBA{230, 230, 230, 255},
		ContrastFg: color.NRGBA{251, 251, 251, 255},
		ContrastBg: color.NRGBA{50, 180, 205, 255},
	}

	ib := iconBrowser{
		searchInput: widget.Editor{SingleLine: true, Submit: true},
		resultList:  widget.List{List: layout.List{Axis: layout.Vertical}},
	}

	var ops op.Ops
	for {
		select {
		case u := <-updates:
			_ = u
			win.Invalidate()
		case e := <-win.Events():
			switch e := e.(type) {
			case system.FrameEvent:
				start := time.Now()
				gtx := layout.NewContext(&ops, e)
				// Process any key events since the previous frame.
				for _, ke := range gtx.Events(win) {
					if ke, ok := ke.(key.Event); ok {
						// a.handleKeyEvent(ke)
						_ = ke
					}
				}
				// Gather key input on the entire window area.
				areaStack := clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops)
				key.InputOp{Tag: win, Keys: key.NameSpace + "|/"}.Add(gtx.Ops)
				ib.layout(gtx, th)
				areaStack.Pop()
				e.Frame(gtx.Ops)
				if showFrameTimes {
					log.Println(time.Now().Sub(start))
				}
			case system.DestroyEvent:
				return e.Err
			}
		}
	}
}

func main() {
	showFrameTimes := flag.Bool("print-frame-times", false, "Print out how long each frame takes.")
	flag.Parse()

	go func() {
		if err := run(*showFrameTimes); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	app.Main()
}
