//go:generate go run cmd/gen/main.go

package main

import (
	"flag"
	"image"
	"image/color"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"gio.tools/fonts/vegur"
	"gio.tools/icons"
	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

const copyNotifDuration = time.Second * 3

var (
	printFrameTimes  = flag.Bool("print-frame-times", false, "Print out how long each frame takes.")
	printSearchTimes = flag.Bool("print-search-times", false, "Print out how long each search takes.")
)

var (
	allIndices  [numEntries]int
	entryClicks [numEntries]clickState
)

type clickState struct {
	lastPressAt time.Time
	gesture.Click
}

func init() {
	for i := 0; i < numEntries; i++ {
		allIndices[i] = i
	}
}

type iconEntry struct {
	name    string // The human readable name.
	varName string // The actual variable name in the icons package.
	key     string // The variable name, but all lowercase for search matching.
	icon    *widget.Icon
}

type iconBrowser struct {
	win *app.Window
	th  *material.Theme

	searchResponses chan searchResponse
	searchCurSeq    int
	searchInput     widget.Editor
	resultList      widget.List
	matchedIndices  []int
	copyNotif       copyNotif
	helpInfo        helpInfo
	openHelpBtn     widget.Clickable

	textSize   unit.Sp
	iconSize   image.Point
	maxWidth   int
	numPerRow  int
	entryWidth int
}

type searchResponse struct {
	indices []int
	seq     int
}

var topLevelKeyFilters = []event.Filter{
	key.Filter{Name: "/"},
	key.Filter{Name: "[", Required: key.ModCtrl},
	key.Filter{Name: "[", Required: key.ModCtrl},
	key.Filter{Name: "]", Required: key.ModCtrl},
	key.Filter{Name: "H", Required: key.ModCtrl},
	key.Filter{Name: "L", Required: key.ModCtrl},
	key.Filter{Name: "U", Required: key.ModCtrl},
	key.Filter{Name: key.NameSpace, Required: key.ModCtrl},
	key.Filter{Name: key.NameEscape},
	key.Filter{Name: key.NameUpArrow},
	key.Filter{Name: key.NameDownArrow},
	key.Filter{Name: key.NamePageUp},
	key.Filter{Name: key.NamePageDown},
	key.Filter{Name: key.NameHome},
	key.Filter{Name: key.NameEnd},
}

func (ib *iconBrowser) frame(gtx C) {
	for {
		event, ok := gtx.Event(topLevelKeyFilters...)
		if !ok {
			break
		}
		if ke, ok := event.(key.Event); ok {
			ib.handleKeyEvent(gtx, ke)
		}
	}
	// Gather key input on the entire window area.
	areaStack := clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops)
	event.Op(gtx.Ops, ib.win)
	ib.layout(gtx)
	areaStack.Pop()
}

func (ib *iconBrowser) handleKeyEvent(gtx C, e key.Event) {
	if e.State != key.Press {
		return
	}
	switch e.Modifiers {
	case key.ModCtrl:
		switch e.Name {
		case "[":
			if ib.th.TextSize > 5 {
				ib.th.TextSize--
			}
		case "]":
			if ib.th.TextSize < 65 {
				ib.th.TextSize++
			}
		case "L", key.NameSpace:
			gtx.Execute(key.FocusCmd{Tag: &ib.searchInput})
			ib.searchInput.SetCaret(ib.searchInput.Len(), 0)
		case "U":
			if ed := &ib.searchInput; gtx.Focused(ed) {
				ed.SetText("")
				ib.runSearch()
			}
		case "H":
			ib.helpInfo.state = helpInfoOpening
		}
	case 0:
		switch e.Name {
		case "/":
			gtx.Execute(key.FocusCmd{Tag: &ib.searchInput})
		case key.NameEscape:
			switch {
			case gtx.Focused(&ib.searchInput):
				gtx.Execute(key.FocusCmd{Tag: nil})
			case ib.helpInfo.state == helpInfoOpened:
				ib.helpInfo.state = helpInfoClosing
			}
		case key.NameUpArrow:
			ib.resultList.Position.First--
		case key.NameDownArrow:
			ib.resultList.Position.First++
		case key.NamePageUp:
			ib.resultList.Position.First -= ib.resultList.Position.Count
		case key.NamePageDown:
			ib.resultList.Position.First += ib.resultList.Position.Count
		case key.NameHome:
			ib.scrollResultListTop()
		case key.NameEnd:
			// The number of results plus one will always be greater than the number
			// of children managed by the list (even if it were a single column),
			// thus ensuring this will always bring it to the very end.
			ib.resultList.List.Position.First = len(ib.matchedIndices) + 1
		}
	}
}

func (ib *iconBrowser) scrollResultListTop() {
	ib.resultList.List.Position.First = 0
	ib.resultList.List.Position.Offset = 0
}

func (ib *iconBrowser) layout(gtx C) {
	for {
		e, ok := ib.searchInput.Update(gtx)
		if !ok {
			break
		}
		if _, ok := e.(widget.ChangeEvent); ok {
			ib.runSearch()
			break
		}
	}
	if ib.openHelpBtn.Clicked(gtx) {
		ib.helpInfo.state = helpInfoOpening
	}
	if ib.matchedIndices == nil {
		ib.matchedIndices = allIndices[:]
	}
	ib.ensure(gtx)
	paint.Fill(gtx.Ops, ib.th.Bg)

	rigidHeights := 0
	{
		gtx1 := gtx
		gtx1.Constraints.Min.Y = 0
		headerDims := ib.layHeader(gtx1)
		rigidHeights += headerDims.Size.Y
	}
	{
		offOp := op.Offset(image.Pt(0, rigidHeights)).Push(gtx.Ops)
		hrDims := rule{color: ib.th.Fg}.layout(gtx)
		rigidHeights += hrDims.Size.Y
		offOp.Pop()
	}
	{
		offOp := op.Offset(image.Pt(0, rigidHeights)).Push(gtx.Ops)
		gtx1 := gtx
		gtx1.Constraints.Max.Y -= rigidHeights
		_ = ib.layResults(gtx1)
		offOp.Pop()
	}

	if time.Since(ib.copyNotif.at) > copyNotifDuration {
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
	if ib.helpInfo.state != helpInfoClosed {
		ib.helpInfo.layout(gtx, ib.th)
	}
}

func (ib *iconBrowser) ensure(gtx C) {
	if ib.textSize != ib.th.TextSize || ib.maxWidth != gtx.Constraints.Max.X {
		ib.textSize = ib.th.TextSize
		icSize := int(ib.textSize * 2.67)
		ib.iconSize = image.Pt(icSize, icSize)
		ib.maxWidth = gtx.Constraints.Max.X
		entryMinWidth := (icSize * 4)
		ib.numPerRow = ib.maxWidth / entryMinWidth
		if ib.numPerRow == 0 {
			ib.numPerRow = 1
		}
		ib.entryWidth = ib.maxWidth / ib.numPerRow
	}
}

func (ib *iconBrowser) layHeader(gtx C) D {
	searchEd := material.Editor(ib.th, &ib.searchInput, "Search...")
	numLbl := material.Body2(ib.th, strconv.Itoa(len(ib.matchedIndices)))
	numLbl.Font.Weight = font.Bold
	return layout.UniformInset(16).Layout(gtx, func(gtx C) D {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return icons.ActionSearch.Layout(gtx, ib.th.Fg)
			}),
			layout.Rigid(layout.Spacer{Width: 16}.Layout),
			layout.Flexed(1, searchEd.Layout),
			layout.Rigid(numLbl.Layout),
			layout.Rigid(material.Caption(ib.th, " icons").Layout),
			layout.Rigid(layout.Spacer{Width: 16}.Layout),
			layout.Rigid(func(gtx C) D {
				btn := material.IconButton(ib.th, &ib.openHelpBtn, icons.ActionHelpOutline, "")
				btn.Size = 28
				btn.Inset = layout.UniformInset(2)
				return btn.Layout(gtx)
			}),
		)
	})
}

func (ib *iconBrowser) layResults(gtx C) D {
	numRows := len(ib.matchedIndices) / ib.numPerRow
	if len(ib.matchedIndices)%ib.numPerRow != 0 {
		numRows++
	}
	return material.List(ib.th, &ib.resultList).Layout(gtx, numRows, func(gtx C, i int) D {
		first := i * ib.numPerRow
		w := gtx.Constraints.Max.X / ib.numPerRow
		h := 0
		for n := 0; n < ib.numPerRow; n++ {
			idx := first + n
			if idx >= len(ib.matchedIndices) {
				break
			}
			xOffsetOp := op.Offset(image.Point{X: n * w}).Push(gtx.Ops)
			dims := ib.layEntry(gtx, ib.matchedIndices[idx])
			if dims.Size.Y > h {
				h = dims.Size.Y
			}
			xOffsetOp.Pop()
		}
		return D{Size: image.Point{X: gtx.Constraints.Max.X, Y: h}}
	})
}

func (ib *iconBrowser) layEntry(gtx C, index int) D {
	en := &allEntries[index]
	click := &entryClicks[index]
	var pressed bool
	for {
		ce, ok := click.Update(gtx.Source)
		if !ok {
			break
		}
		if ce.Kind == gesture.KindPress {
			pressed = true
			break
		}
	}
	if pressed {
		click.lastPressAt = gtx.Now
		varPath := "icons." + en.varName
		gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(varPath))})
		ib.copyNotif = copyNotif{
			msg: varPath,
			at:  time.Now(),
		}
		gtx.Execute(op.InvalidateCmd{})
		go func() {
			time.Sleep(copyNotifDuration + time.Millisecond*100)
			ib.win.Invalidate()
		}()
	}

	const inset = 10   // The outer inset that serves as space between entries.
	const spacing = 15 // The space before and after each inner element of an entry.

	gtx.Constraints.Max.X = ib.entryWidth - inset*2
	insetOffOp := op.Offset(image.Point{inset, inset}).Push(gtx.Ops)

	// We need to determine the entry's inner dimensions for two reasons: 1) to properly
	// fill in the background if needed and 2) to know this entry's total height when we
	// return the overall dimensions.
	innerDims := D{Size: image.Point{
		X: gtx.Constraints.Max.X,
		Y: spacing,
	}}
	// Record the ops for drawing this entry so we can replay them after we fill in the
	// background and add the click gesture.
	m := op.Record(gtx.Ops)
	{
		// Draw the horizontally centered icon.
		x := gtx.Constraints.Max.X/2 - ib.iconSize.X/2
		offOp := op.Offset(image.Pt(x, spacing)).Push(gtx.Ops)
		gtx1 := gtx
		gtx1.Constraints.Max = ib.iconSize
		iconDims := en.icon.Layout(gtx1, color.NRGBA{210, 210, 210, 255})
		innerDims.Size.Y += iconDims.Size.Y + spacing
		offOp.Pop()
	}
	{
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		// Offset down (to after the icon) to draw the name label.
		offOp := op.Offset(image.Pt(0, innerDims.Size.Y)).Push(gtx.Ops)
		name := material.Body2(ib.th, en.name)
		name.Alignment = text.Middle
		nameDims := name.Layout(gtx)
		innerDims.Size.Y += nameDims.Size.Y + spacing
		offOp.Pop()
	}
	drawEntry := m.Stop()

	// We animate click presses by scaling the entry down and back up over a certain time
	// frame.
	const animTimeFrame = 200
	sinceLastPress := gtx.Now.Sub(click.lastPressAt).Milliseconds()
	isAnimating := sinceLastPress < animTimeFrame
	if isAnimating {
		const halfMillis = animTimeFrame / 2
		// The scaling factor is some percentage between 70% - 100%, based on where we are
		// in the animation time frame (70% being halfway through). Since the animation is
		// "shrink down and expand back to normal size," we need the same scale factor on
		// both halves of the time frame. In other words, if the animation time frame is
		// 200ms, we should scale the entry to 85% (halfway scaled down) for both 50ms
		// (halfway there) and 150ms (halfway back).
		pct := 1 - (float32(sinceLastPress) / halfMillis)
		if sinceLastPress > halfMillis {
			pct = (float32(sinceLastPress) - halfMillis) / halfMillis
		}
		// The origin point is the center of the entry.
		origin := f32.Pt(float32(innerDims.Size.X)/2, float32(innerDims.Size.Y)/2)
		scale := 0.7 + (0.3 * pct)
		af := f32.Affine2D{}.Scale(origin, f32.Pt(scale, scale))
		op.Affine(af).Add(gtx.Ops)
		gtx.Execute(op.InvalidateCmd{})
	}

	rrOp := clip.UniformRRect(image.Rectangle{Max: innerDims.Size}, 6).Push(gtx.Ops)
	if click.Hovered() || isAnimating {
		paint.LinearGradientOp{
			Stop1:  layout.FPt(image.Point{}),
			Stop2:  layout.FPt(innerDims.Size),
			Color1: color.NRGBA{32, 32, 32, 255},
			Color2: color.NRGBA{65, 65, 65, 255},
		}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
	}
	click.Add(gtx.Ops)
	drawEntry.Add(gtx.Ops)
	rrOp.Pop()

	insetOffOp.Pop()
	return D{Size: image.Point{
		X: gtx.Constraints.Max.X + inset*2,
		Y: innerDims.Size.Y + inset*2,
	}}
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
				log.Println(time.Since(start))
			}
		}()
		input := strings.ToLower(ib.searchInput.Text())
		if input == "" {
			return
		}
		resp.indices = make([]int, 0, len(allEntries)/3)
		for i := range allEntries {
			e := &allEntries[i]
			if strings.Contains(e.key, input) || strings.Contains(strings.ToLower(e.name), input) {
				resp.indices = append(resp.indices, i)
			}
		}
	}()
}

func run() error {
	var win app.Window
	win.Option(app.Title("Gio Icon Browser"))
	win.Option(app.MinSize(600, 600))
	win.Option(app.Size(980, 770))

	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(vegur.Collection()))
	th.TextSize = 18
	th.Palette = material.Palette{
		Bg:         color.NRGBA{15, 15, 15, 255},
		Fg:         color.NRGBA{230, 230, 230, 255},
		ContrastFg: color.NRGBA{251, 251, 251, 255},
		ContrastBg: color.NRGBA{89, 173, 196, 255},
	}

	ib := iconBrowser{
		win:             &win,
		th:              th,
		searchResponses: make(chan searchResponse),
		searchInput:     widget.Editor{SingleLine: true, Submit: true},
		resultList:      widget.List{List: layout.List{Axis: layout.Vertical}},
	}
	// ib.searchInput.Focus()

	events := make(chan event.Event)
	acks := make(chan struct{})

	go func() {
		for {
			ev := win.Event()
			events <- ev
			<-acks
			if _, ok := ev.(app.DestroyEvent); ok {
				return
			}
		}
	}()

	var ops op.Ops
	for {
		select {
		case r := <-ib.searchResponses:
			if r.seq == ib.searchCurSeq {
				ib.matchedIndices = r.indices
				ib.searchCurSeq = 0
				ib.scrollResultListTop()
			}
			ib.win.Invalidate()
		case e := <-events:
			switch e := e.(type) {
			case app.FrameEvent:
				start := time.Now()
				gtx := app.NewContext(&ops, e)
				ib.frame(gtx)
				e.Frame(gtx.Ops)
				if *printFrameTimes {
					log.Println(time.Since(start))
				}
			case app.DestroyEvent:
				acks <- struct{}{}
				return e.Err
			}
			acks <- struct{}{}
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
