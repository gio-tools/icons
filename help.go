package main

import (
	"image"
	"image/color"

	"gioui.org/gesture"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/widget/material"
)

var allShortcuts = [...][2]string{
	{"Ctrl-[", "Decrease font and icon size."},
	{"Ctrl-]", "Increase font and icon size."},
	{"/", "Focus search field."},
	{"Ctrl-L", "Focus search field."},
	{"Ctrl-Space", "Focus search field."},
	{"Ctrl-U", "Clear search field."},
	{"Escape", "Unfocus the search field."},
	{"Arrow Up & Down", "Scroll by row."},
	{"Page Up & Down", "Scroll by page."},
}

type shortcutGroup struct {
	title  string
	values [2]string
}

type helpOverlay struct {
	active bool
	click  gesture.Click
}

func (h *helpOverlay) layout(gtx C, th *material.Theme) D {
	gtx.Constraints.Min.Y = 0
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, color.NRGBA{20, 20, 20, 253})
	height := 0
	{
		dims := layout.UniformInset(20).Layout(gtx, func(gtx C) D {
			lbl := material.H5(th, "Shortcuts")
			lbl.Alignment = text.Middle
			return lbl.Layout(gtx)
		})
		height = dims.Size.Y
	}
	for _, pair := range allShortcuts {
		offOp := op.Offset(image.Pt(0, height)).Push(gtx.Ops)
		dims := layShortcutRow(gtx, th, pair)
		height += dims.Size.Y + 10
		offOp.Pop()
	}
	h.click.Add(gtx.Ops) // Capture mouse events so they don't fall through the overlay.
	return D{Size: gtx.Constraints.Max}
}

func layShortcutRow(gtx C, th *material.Theme, pair [2]string) D {
	sc := material.Body1(th, pair[0])
	sc.Alignment = text.End
	desc := material.Body1(th, pair[1])
	return layout.Flex{}.Layout(gtx,
		layout.Flexed(0.45, sc.Layout),
		layout.Rigid(layout.Spacer{Width: 20}.Layout),
		layout.Flexed(0.5, desc.Layout),
	)
}
