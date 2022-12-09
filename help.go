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

type helpOverlay struct {
	active   bool
	click    gesture.Click
	closeBtn widget.Clickable
}

func (h *helpOverlay) layout(gtx C, th *material.Theme) D {
	if h.closeBtn.Clicked() {
		h.active = false
	}
	const inset = 16
	maxSize := gtx.Constraints.Max
	// Draw the overlay and add the clip the entire area so click events don't fall
	// through.
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, color.NRGBA{20, 20, 20, 240})
	h.click.Add(gtx.Ops) // Capture mouse events so they don't fall through the overlay.

	gtx.Constraints.Max.X /= 2
	gtx.Constraints.Min.Y = 0
	offOpHalf := op.Offset(image.Pt(gtx.Constraints.Max.X, 0)).Push(gtx.Ops)
	paint.FillShape(gtx.Ops, th.Bg, clip.Rect{Max: gtx.Constraints.Max}.Op())

	defer offOpHalf.Pop()
	return layout.Inset{Top: 16, Right: 16, Bottom: 16, Left: 32}.Layout(gtx, func(gtx C) D {
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
		return D{Size: maxSize}
	})
}

func layShortcutRow(gtx C, th *material.Theme, idx int) D {
	sc := material.Body1(th, allShortcuts[idx][0])
	desc := material.Body1(th, allShortcuts[idx][1])
	return layout.Flex{}.Layout(gtx,
		layout.Flexed(0.45, sc.Layout),
		layout.Rigid(layout.Spacer{Width: 20}.Layout),
		layout.Flexed(0.5, desc.Layout),
	)
}
