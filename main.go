//go:generate go run cmd/gen/main.go

package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"strings"
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

// CLI flags.
var (
	printFrameTimes  bool
	printSearchTimes bool
)

type iconEntry struct {
	name  string
	key   string // The name but all lowercase for search matching.
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
	searchResponses chan searchResponse
	searchCurSeq    int
	searchInput     widget.Editor
	resultList      widget.List
	matchedIndices  []int
}

type searchResponse struct {
	indices []int
	seq     int
}

func (ib *iconBrowser) handleKeyEvent(gtx C, e key.Event) {
	if e.State != key.Press {
		return
	}
	switch e.Modifiers {
	case key.ModCtrl:
		switch e.Name {
		case "L", key.NameSpace:
			ib.searchInput.Focus()
		case "U":
			if ed := &ib.searchInput; ed.Focused() {
				ed.SetText("")
				ib.runSearch()
			}
		}
	case 0:
		switch e.Name {
		case "/":
			ib.searchInput.Focus()
		case key.NameEscape:
			if ib.searchInput.Focused() {
				key.FocusOp{Tag: nil}.Add(gtx.Ops)
			}
		case key.NameHome:
			ib.resultList.List.Position.First = 0
			ib.resultList.List.Position.Offset = 0
		case key.NameEnd:
			// The number of results plus one will always be greater than the number
			// of children managed by the list (even if it were a single column),
			// thus ensuring this will always bring it to the very end.
			ib.resultList.List.Position.First = len(ib.matchedIndices) + 1
		}
	}
}

func (ib *iconBrowser) layout(gtx C, th *material.Theme) D {
	for _, e := range ib.searchInput.Events() {
		if _, ok := e.(widget.ChangeEvent); ok {
			ib.runSearch()
		}
	}
	if ib.matchedIndices == nil {
		ib.matchedIndices = allIndices
	}
	paint.Fill(gtx.Ops, th.Bg)
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return ib.layHeader(gtx, th, len(ib.matchedIndices))
		}),
		layout.Rigid(rule{color: th.Fg}.layout),
		layout.Flexed(1, func(gtx C) D {
			return ib.layResults(gtx, th, ib.matchedIndices)
		}),
	)
}

func (ib *iconBrowser) layHeader(gtx C, th *material.Theme, n int) D {
	searchEd := material.Editor(th, &ib.searchInput, "Search...")
	numLbl := material.Body2(th, fmt.Sprintf("%d", n))
	numLbl.Font.Weight = text.Bold
	iconsLbl := material.Caption(th, " icons")
	return layout.UniformInset(16).Layout(gtx, func(gtx C) D {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return iconSearch.Layout(gtx, th.Fg)
			}),
			layout.Rigid(layout.Spacer{Width: 16}.Layout),
			layout.Flexed(1, searchEd.Layout),
			layout.Rigid(numLbl.Layout),
			layout.Rigid(iconsLbl.Layout),
		)
	})
}

func (ib *iconBrowser) layResults(gtx C, th *material.Theme, indices []int) D {
	const weight = 1.0 / 3.0
	var rows []layout.Widget
	for i := 0; i < len(indices); i += 3 {
		var cells []layout.FlexChild
		for n := 0; n < 3; n++ {
			indexIndex := i + n
			if indexIndex >= len(indices) {
				cells = append(cells, layout.Flexed(weight, emptyWidget))
				continue
			}
			entry := &allEntries[indices[indexIndex]]
			cells = append(cells, layout.Flexed(weight, func(gtx C) D {
				nameLbl := material.Body2(th, entry.name)
				nameLbl.Alignment = text.Middle
				return layout.Flex{Alignment: layout.Middle, Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						gtx.Constraints.Max.X = 48
						gtx.Constraints.Max.Y = 48
						return entry.icon.Layout(gtx, th.Fg)
					}),
					layout.Rigid(layout.Spacer{Height: 10}.Layout),
					layout.Rigid(nameLbl.Layout),
				)
			}))
		}
		rows = append(rows, func(gtx C) D {
			return layout.Inset{Top: 20, Bottom: 20}.Layout(gtx, func(gtx C) D {
				return layout.Flex{}.Layout(gtx, cells...)
			})
		})
	}
	return material.List(th, &ib.resultList).Layout(gtx, len(rows), func(gtx C, i int) D {
		return rows[i](gtx)
	})
}

func (ib *iconBrowser) runSearch() {
	ib.searchCurSeq++
	resp := searchResponse{
		indices: nil,
		seq:     ib.searchCurSeq,
	}
	go func() {
		start := time.Now()
		defer func() {
			ib.searchResponses <- resp
			if printSearchTimes {
				log.Println(time.Now().Sub(start))
			}
		}()
		input := strings.ToLower(ib.searchInput.Text())
		if input == "" {
			return
		}
		resp.indices = make([]int, 0, len(allEntries)/2)
		for i := range allEntries {
			e := &allEntries[i]
			if strings.Contains(e.key, input) {
				resp.indices = append(resp.indices, i)
			}
		}
	}()
}

type rule struct {
	width int
	color color.NRGBA
	axis  layout.Axis
}

func (rl rule) layout(gtx C) D {
	if rl.width == 0 {
		rl.width = 1
	}
	size := image.Point{gtx.Constraints.Max.X, rl.width}
	if rl.axis == layout.Vertical {
		size = image.Point{rl.width, gtx.Constraints.Max.Y}
	}
	rect := clip.Rect{Max: size}.Op()
	paint.FillShape(gtx.Ops, rl.color, rect)
	return D{Size: size}
}

func emptyWidget(gtx C) D { return D{} }

const topLevelKeySet = "Ctrl-[L,U," + key.NameSpace + "]" +
	"|/" +
	"|" + key.NameEscape +
	"|" + key.NameHome +
	"|" + key.NameEnd

func run() error {
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
		searchResponses: make(chan searchResponse),
		searchInput:     widget.Editor{SingleLine: true, Submit: true},
		resultList:      widget.List{List: layout.List{Axis: layout.Vertical}},
	}

	var ops op.Ops
	for {
		select {
		case r := <-ib.searchResponses:
			if r.seq == ib.searchCurSeq {
				ib.matchedIndices = r.indices
				ib.searchCurSeq = 0
			}
			win.Invalidate()
		case e := <-win.Events():
			switch e := e.(type) {
			case system.FrameEvent:
				start := time.Now()
				gtx := layout.NewContext(&ops, e)
				// Process any key events since the previous frame.
				for _, ke := range gtx.Events(win) {
					if ke, ok := ke.(key.Event); ok {
						ib.handleKeyEvent(gtx, ke)
					}
				}
				// Gather key input on the entire window area.
				areaStack := clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops)
				key.InputOp{Tag: win, Keys: topLevelKeySet}.Add(gtx.Ops)
				ib.layout(gtx, th)
				areaStack.Pop()
				e.Frame(gtx.Ops)
				if printFrameTimes {
					log.Println(time.Now().Sub(start))
				}
			case system.DestroyEvent:
				return e.Err
			}
		}
	}
}

func main() {
	flag.BoolVar(&printFrameTimes, "print-frame-times", false, "Print out how long each frame takes.")
	flag.BoolVar(&printSearchTimes, "print-search-times", false, "Print out how long each search run takes.")
	flag.Parse()

	go func() {
		if err := run(); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	app.Main()
}
