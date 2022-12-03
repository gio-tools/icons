//go:generate go run cmd/gen/main.go

package main

import (
	"flag"
	"image"
	"image/color"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/font/opentype"
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
	"github.com/steverusso/gio-fonts/vegur/vegurbold"
	"github.com/steverusso/gio-fonts/vegur/vegurregular"
)

const copyNotifDuration = time.Second * 3

// CLI flags.
var (
	printFrameTimes  = flag.Bool("print-frame-times", false, "Print out how long each frame takes.")
	printSearchTimes = flag.Bool("print-search-times", false, "Print out how long each search takes.")
)

var (
	allIndices  [numEntries]int
	entryClicks [numEntries]gesture.Click
)

func init() {
	for i := 0; i < numEntries; i++ {
		allIndices[i] = i
	}
}

type iconEntry struct {
	name    string
	key     string // The name but all lowercase for search matching.
	varName string
	icon    *widget.Icon
}

type iconBrowser struct {
	win             *app.Window
	th              *material.Theme
	searchResponses chan searchResponse
	searchCurSeq    int
	searchInput     widget.Editor
	resultList      widget.List
	iconSize        int
	matchedIndices  []int
	copyNotif       copyNotif
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

func (ib *iconBrowser) layout(gtx C) {
	for _, e := range ib.searchInput.Events() {
		if _, ok := e.(widget.ChangeEvent); ok {
			ib.runSearch()
			break
		}
	}
	if ib.matchedIndices == nil {
		ib.matchedIndices = allIndices[:]
	}
	paint.Fill(gtx.Ops, ib.th.Bg)
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(ib.layHeader),
		layout.Rigid(rule{color: ib.th.Fg}.layout),
		layout.Flexed(1, ib.layResults),
	)
	if time.Now().Sub(ib.copyNotif.at) > copyNotifDuration {
		ib.copyNotif = copyNotif{}
	}
	if ib.copyNotif.msg != "" {
		layout.S.Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = 0
			return layout.Inset{Bottom: 20}.Layout(gtx, func(gtx C) D {
				return ib.copyNotif.layout(gtx, ib.th)
			})
		})
	}
}

func (ib *iconBrowser) layHeader(gtx C) D {
	searchEd := material.Editor(ib.th, &ib.searchInput, "Search...")
	numLbl := material.Body2(ib.th, strconv.Itoa(len(ib.matchedIndices)))
	numLbl.Font.Weight = text.Bold
	return layout.UniformInset(16).Layout(gtx, func(gtx C) D {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return iconSearch.Layout(gtx, ib.th.Fg)
			}),
			layout.Rigid(layout.Spacer{Width: 16}.Layout),
			layout.Flexed(1, searchEd.Layout),
			layout.Rigid(numLbl.Layout),
			layout.Rigid(material.Caption(ib.th, " icons").Layout),
		)
	})
}

func (ib *iconBrowser) layResults(gtx C) D {
	const weight = 1.0 / 3.0
	var rows []layout.Widget
	for i := 0; i < len(ib.matchedIndices); i += 3 {
		var cells []layout.FlexChild
		for n := 0; n < 3; n++ {
			indexIndex := i + n
			if indexIndex >= len(ib.matchedIndices) {
				cells = append(cells, layout.Flexed(weight, emptyWidget))
				continue
			}
			entryIndex := ib.matchedIndices[indexIndex]
			cells = append(cells, layout.Flexed(weight, func(gtx C) D {
				return ib.layEntry(gtx, entryIndex)
			}))
		}
		rows = append(rows, func(gtx C) D {
			return layout.Flex{}.Layout(gtx, cells...)
		})
	}
	return material.List(ib.th, &ib.resultList).Layout(gtx, len(rows), func(gtx C, i int) D {
		return rows[i](gtx)
	})
}

func (ib *iconBrowser) layEntry(gtx C, index int) D {
	en := &allEntries[index]
	click := &entryClicks[index]
	var clicked bool
	for _, e := range click.Events(gtx) {
		if e.Type == gesture.TypeClick {
			clicked = true
		}
	}
	if clicked {
		varPath := "icons." + en.varName
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
	case click.Hovered():
		bg = color.NRGBA{50, 50, 50, 255}
	}
	nameLbl := material.Body2(ib.th, en.name)
	nameLbl.Alignment = text.Middle
	m := op.Record(gtx.Ops)
	dims := layout.Inset{Top: 25, Right: 10, Bottom: 25, Left: 10}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Alignment: layout.Middle, Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				gtx.Constraints.Max.X = ib.iconSize
				gtx.Constraints.Max.Y = ib.iconSize
				return en.icon.Layout(gtx, color.NRGBA{210, 210, 210, 255})
			}),
			layout.Rigid(layout.Spacer{Height: 10}.Layout),
			layout.Rigid(nameLbl.Layout),
		)
	})
	call := m.Stop()
	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: dims.Size}.Op())
	defer clip.Rect(image.Rectangle{Max: dims.Size}).Push(gtx.Ops).Pop()
	click.Add(gtx.Ops)
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
			if *printSearchTimes {
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
			if strings.Contains(e.key, input) || strings.Contains(strings.ToLower(e.name), input) {
				resp.indices = append(resp.indices, i)
			}
		}
	}()
}

func mustFace(data []byte) text.Face {
	face, err := opentype.Parse(data)
	if err != nil {
		panic("failed to parse font: " + err.Error())
	}
	return face
}

const topLevelKeySet = "Ctrl-[L,U," + key.NameSpace + "]" +
	"|/" +
	"|" + key.NameEscape +
	"|" + key.NameHome +
	"|" + key.NameEnd

func run() error {
	win := app.NewWindow(
		app.Size(900, 800),
		app.Title("Gio Icon Browser"),
	)
	win.Perform(system.ActionCenter)

	th := material.NewTheme([]text.FontFace{
		{Font: text.Font{Typeface: "Vegur"}, Face: mustFace(vegurregular.OTF)},
		{Font: text.Font{Typeface: "Vegur", Weight: text.Bold}, Face: mustFace(vegurbold.OTF)},
	})
	th.TextSize = 18
	th.Palette = material.Palette{
		Bg:         color.NRGBA{15, 15, 15, 255},
		Fg:         color.NRGBA{230, 230, 230, 255},
		ContrastFg: color.NRGBA{251, 251, 251, 255},
		ContrastBg: color.NRGBA{50, 180, 205, 255},
	}

	ib := iconBrowser{
		win:             win,
		th:              th,
		searchResponses: make(chan searchResponse),
		searchInput:     widget.Editor{SingleLine: true, Submit: true},
		resultList:      widget.List{List: layout.List{Axis: layout.Vertical}},
		iconSize:        48,
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
				ib.layout(gtx)
				areaStack.Pop()
				e.Frame(gtx.Ops)
				if *printFrameTimes {
					log.Println(time.Now().Sub(start))
				}
			case system.DestroyEvent:
				return e.Err
			}
		}
	}
}

func main() {
	flag.Parse()

	go func() {
		if err := run(); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	app.Main()
}
