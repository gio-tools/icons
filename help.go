package main

import (
	"image"
	"image/color"

	"gioui.org/gesture"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

var allShortcuts = [...][2]string{
	{"Ctrl - [", "Decrease font and icon size."},
	{"Ctrl - ]", "Increase font and icon size."},
	{"/", "Focus search field."},
	{"Ctrl - L", "Focus search field."},
	{"Ctrl - Space", "Focus search field."},
	{"Ctrl - U", "Clear search field."},
	{"Escape", "Unfocus the search field."},
	{"Arrow Up & Down", "Scroll by row."},
	{"Page Up & Down", "Scroll by page."},
}

type helpInfo struct {
	active   bool
	click    gesture.Click
	closeBtn widget.Clickable
}

func (h *helpInfo) layout(gtx C, th *material.Theme) D {
	if h.closeBtn.Clicked() {
		h.active = false
	}
	for _, e := range h.click.Events(gtx) {
		// Close the drawer if the user clicks outside it.
		if e.Type == gesture.TypePress && e.Position.X < gtx.Constraints.Max.X-460 {
			h.active = false
		}
	}
	originMax := gtx.Constraints.Max
	// Draw the overlay and add the clip the entire area so click events don't fall
	// through.
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, color.NRGBA{20, 20, 20, 240})
	h.click.Add(gtx.Ops)

	gtx.Constraints.Max.X = 460
	defer op.Offset(image.Pt(originMax.X-gtx.Constraints.Max.X, 0)).Push(gtx.Ops).Pop()
	paint.FillShape(gtx.Ops, th.Bg, clip.Rect{Max: gtx.Constraints.Max}.Op())

	return layout.Inset{Top: 16, Right: 16, Bottom: 16, Left: 32}.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.Y = 0
		height := 0
		{
			lbl := material.H5(th, "Keyboard Shortcuts")
			btn := material.IconButton(th, &h.closeBtn, &iconExitToApp, "")
			btn.Inset = layout.UniformInset(4)
			dims := layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, lbl.Layout),
				layout.Rigid(btn.Layout),
			)
			height = dims.Size.Y + 32
		}
		for i := range allShortcuts {
			offOp := op.Offset(image.Pt(0, height)).Push(gtx.Ops)
			dims := layShortcutRow(gtx, th, i)
			height += dims.Size.Y + 10
			offOp.Pop()
		}
		return D{Size: originMax}
	})
}

func layShortcutRow(gtx C, th *material.Theme, idx int) D {
	// Draw the keystroke text in the first column.
	colWidth1 := int(float32(gtx.Constraints.Max.X) * 0.45)
	gtx1 := gtx
	gtx1.Constraints.Max.X = colWidth1
	dims := material.Body1(th, allShortcuts[idx][0]).Layout(gtx1)
	height := dims.Size.Y
	// Move to draw the description text in the 2nd column.
	gtx2 := gtx
	gtx2.Constraints.Max.X = gtx.Constraints.Max.X - colWidth1 - 16
	offOp := op.Offset(image.Pt(colWidth1+16, 0)).Push(gtx2.Ops)
	dims = material.Body1(th, allShortcuts[idx][1]).Layout(gtx2)
	if dims.Size.Y > height {
		height = dims.Size.Y
	}
	offOp.Pop()
	return D{Size: image.Pt(gtx.Constraints.Max.X, height)}
}
