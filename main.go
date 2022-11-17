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
	"github.com/steverusso/giofonts/sourcesanspro"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// CLI flags.
var (
	printFrameTimes  bool
	printSearchTimes bool
)

const copyNotifDuration = time.Second * 3

var iconSearch = mi(icons.ActionSearch)

type iconEntry struct {
	name    string
	key     string // The name but all lowercase for search matching.
	varName string
	icon    *widget.Icon
	click   gesture.Click
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
	win             *app.Window
	searchResponses chan searchResponse
	searchCurSeq    int
	searchInput     widget.Editor
	resultList      widget.List
	matchedIndices  []int
	copyNotif       copyNotif
}

type searchResponse struct {
	indices []int
	seq     int
}

type copyNotif struct {
	msg string
	at  time.Time
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

func (ib *iconBrowser) layout(gtx C, th *material.Theme) {
	for _, e := range ib.searchInput.Events() {
		if _, ok := e.(widget.ChangeEvent); ok {
			ib.runSearch()
			break
		}
	}
	if ib.matchedIndices == nil {
		ib.matchedIndices = allIndices
	}
	paint.Fill(gtx.Ops, th.Bg)
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return ib.layHeader(gtx, th, len(ib.matchedIndices))
		}),
		layout.Rigid(rule{color: th.Fg}.layout),
		layout.Flexed(1, func(gtx C) D {
			return ib.layResults(gtx, th, ib.matchedIndices)
		}),
	)
	if time.Now().Sub(ib.copyNotif.at) > copyNotifDuration {
		ib.copyNotif = copyNotif{}
	}
	if ib.copyNotif.msg != "" {
		layout.S.Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = 0
			return layout.Inset{Bottom: 20}.Layout(gtx, func(gtx C) D {
				lbl := material.Body1(th, ib.copyNotif.msg)
				lbl.Alignment = text.Middle
				lbl.Color = color.NRGBA{255, 255, 255, 255}
				lbl.Font.Weight = text.SemiBold
				m := op.Record(gtx.Ops)
				dims := layout.Inset{Top: 20, Right: 25, Bottom: 20, Left: 25}.Layout(gtx, func(gtx C) D {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(lbl.Layout),
						layout.Rigid(material.Body1(th, "  copied!").Layout),
					)
				})
				call := m.Stop()
				paint.FillShape(gtx.Ops, color.NRGBA{4, 222, 113, 255}, clip.RRect{
					NW: 6, NE: 6, SE: 6, SW: 6,
					Rect: image.Rectangle{
						Min: image.Point{-2, -2},
						Max: image.Point{dims.Size.X + 2, dims.Size.Y + 2},
					},
				}.Op(gtx.Ops))
				paint.FillShape(gtx.Ops, color.NRGBA{20, 140, 49, 255}, clip.RRect{
					NW: 5, NE: 5, SE: 5, SW: 5,
					Rect: image.Rectangle{Max: dims.Size},
				}.Op(gtx.Ops))
				call.Add(gtx.Ops)
				return dims
			})
		})
	}
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
				return ib.layEntry(gtx, th, entry)
			}))
		}
		rows = append(rows, func(gtx C) D {
			return layout.Flex{}.Layout(gtx, cells...)
		})
	}
	return material.List(th, &ib.resultList).Layout(gtx, len(rows), func(gtx C, i int) D {
		return rows[i](gtx)
	})
}

func (ib *iconBrowser) layEntry(gtx C, th *material.Theme, en *iconEntry) D {
	var clicked bool
	for _, e := range en.click.Events(gtx) {
		if e.Type == gesture.TypeClick {
			clicked = true
		}
	}
	if clicked {
		varPath := fmt.Sprintf("icons.%s", en.varName)
		ib.win.WriteClipboard(varPath)
		ib.copyNotif = copyNotif{
			msg: varPath,
			at:  time.Now(),
		}
		op.InvalidateOp{}.Add(gtx.Ops)
		go func() {
			time.Sleep(copyNotifDuration + time.Millisecond*100)
			ib.win.Invalidate()
		}()
	}
	var bg color.NRGBA
	switch {
	case clicked:
		bg = color.NRGBA{0, 0, 0, 255}
	case en.click.Hovered():
		bg = color.NRGBA{50, 50, 50, 255}
	}
	nameLbl := material.Body2(th, en.name)
	nameLbl.Alignment = text.Middle
	m := op.Record(gtx.Ops)
	dims := layout.Inset{Top: 25, Bottom: 25}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Alignment: layout.Middle, Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				gtx.Constraints.Max.X = 48
				gtx.Constraints.Max.Y = 48
				return en.icon.Layout(gtx, color.NRGBA{210, 210, 210, 255})
			}),
			layout.Rigid(layout.Spacer{Height: 10}.Layout),
			layout.Rigid(nameLbl.Layout),
		)
	})
	call := m.Stop()
	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: dims.Size}.Op())
	defer clip.Rect(image.Rectangle{Max: dims.Size}).Push(gtx.Ops).Pop()
	en.click.Add(gtx.Ops)
	call.Add(gtx.Ops)
	return dims
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

	th := material.NewTheme(sourcesanspro.Collection())
	th.TextSize = 17
	th.Palette = material.Palette{
		Bg:         color.NRGBA{15, 15, 15, 255},
		Fg:         color.NRGBA{230, 230, 230, 255},
		ContrastFg: color.NRGBA{251, 251, 251, 255},
		ContrastBg: color.NRGBA{50, 180, 205, 255},
	}

	ib := iconBrowser{
		win:             win,
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
			ib.win.Invalidate()
		case e := <-ib.win.Events():
			switch e := e.(type) {
			case system.FrameEvent:
				start := time.Now()
				gtx := layout.NewContext(&ops, e)
				// Process any key events since the previous frame.
				for _, ke := range gtx.Events(ib.win) {
					if ke, ok := ke.(key.Event); ok {
						ib.handleKeyEvent(gtx, ke)
					}
				}
				// Gather key input on the entire window area.
				areaStack := clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops)
				key.InputOp{Tag: ib.win, Keys: topLevelKeySet}.Add(gtx.Ops)
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
