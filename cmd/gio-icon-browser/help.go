package main

import (
	"image"
	"image/color"

	"gio.tools/icons"
	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

var allShortcuts = [...]shortcutInfo{
	{[]string{"Ctrl", "["}, "Decrease font and icon size."},
	{[]string{"Ctrl", "]"}, "Increase font and icon size."},
	{[]string{"Ctrl", "Space"}, "Focus search field."},
	{[]string{"Ctrl", "L"}, "Focus search field."},
	{[]string{"/"}, "Focus search field."},
	{[]string{"Ctrl", "U"}, "Clear search field."},
	{[]string{"↑"}, "Scroll up by one row."},
	{[]string{"↓"}, "Scroll down by one row."},
	{[]string{"Page Up"}, "Scroll up by one page."},
	{[]string{"Page Down"}, "Scroll down by one page."},
	{[]string{"Ctrl", "H"}, "Open this help info."},
}

type shortcutInfo struct {
	keys []string
	desc string
}

type helpInfo struct {
	state    helpInfoState
	portion  int
	click    gesture.Click
	closeBtn widget.Clickable
}

type helpInfoState byte

const (
	helpInfoClosed helpInfoState = iota
	helpInfoClosing
	helpInfoOpened
	helpInfoOpening
)

func (h *helpInfo) layout(gtx C, th *material.Theme) D {
	const drawerWidth = 500
	// Take care of animation progress.
	switch h.state {
	case helpInfoOpening:
		if h.portion == drawerWidth {
			h.state = helpInfoOpened
		} else {
			h.portion += 50
		}
		gtx.Execute(op.InvalidateCmd{})
	case helpInfoClosing:
		if h.portion == 0 {
			h.state = helpInfoClosed
		} else {
			h.portion -= 50
		}
		gtx.Execute(op.InvalidateCmd{})
	}
	var overlayClicked bool
	for {
		ce, ok := h.click.Update(gtx.Source)
		if !ok {
			break
		}
		if ce.Kind == gesture.KindPress && ce.Position.X < gtx.Constraints.Max.X-drawerWidth {
			overlayClicked = true
			break
		}
	}
	if h.closeBtn.Clicked(gtx) || overlayClicked {
		h.state = helpInfoClosing
	}
	originMax := gtx.Constraints.Max
	// Draw the overlay and clip the entire window area for click events so they don't
	// fall through.
	paint.Fill(gtx.Ops, color.NRGBA{20, 20, 20, 240})
	clipOp := clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops)
	h.click.Add(gtx.Ops)
	clipOp.Pop()
	// Offset to the drawer and fill in the background.
	drawerX := originMax.X - drawerWidth + (drawerWidth - h.portion)
	defer op.Offset(image.Pt(drawerX, 0)).Push(gtx.Ops).Pop()
	gtx.Constraints.Max.X = drawerWidth
	paint.FillShape(gtx.Ops, th.Bg, clip.Rect{Max: gtx.Constraints.Max}.Op())
	// The drawer content.
	return layout.Inset{Top: 16, Right: 16, Bottom: 16, Left: 32}.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.Y = 0
		height := 0
		{
			lbl := material.H5(th, "Keyboard Shortcuts")
			lbl.Font.Weight = font.Bold
			btn := material.IconButton(th, &h.closeBtn, icons.ActionExitToApp, "")
			btn.Inset = layout.UniformInset(4)
			dims := layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, lbl.Layout),
				layout.Rigid(btn.Layout),
			)
			height = dims.Size.Y + 24
			// Horizontal rule after the heading.
			offOp := op.Offset(image.Pt(0, height)).Push(gtx.Ops)
			hrDims := rule{color: th.Fg}.layout(gtx)
			offOp.Pop()
			height += hrDims.Size.Y + 24
		}
		const leftInset = 8
		for i := range allShortcuts {
			offOp := op.Offset(image.Pt(leftInset, height)).Push(gtx.Ops)
			dims := layShortcutRow(gtx, th, i)
			height += dims.Size.Y + 10
			offOp.Pop()
		}
		return D{Size: originMax}
	})
}

func layShortcutRow(gtx C, th *material.Theme, idx int) D {
	// Draw the keystroke text in the first column.
	height := 0
	{
		xOffset := 0
		for i := range allShortcuts[idx].keys {
			offOp := op.Offset(image.Pt(xOffset, 0)).Push(gtx.Ops)
			dims := widget.Border{
				Color:        th.Fg,
				Width:        1,
				CornerRadius: 1,
			}.Layout(gtx, func(gtx C) D {
				return layout.Inset{Top: 3, Right: 8, Bottom: 3, Left: 8}.Layout(gtx, func(gtx C) D {
					gtx.Constraints.Min.X = 0
					lbl := material.Body2(th, allShortcuts[idx].keys[i])
					lbl.Font.Typeface = "monospace"
					return lbl.Layout(gtx)
				})
			})
			offOp.Pop()
			xOffset += dims.Size.X + 8
			if dims.Size.Y > height {
				height = dims.Size.Y
			}
		}
	}
	// Move to draw the description text in the 2nd column.
	fullWidth := gtx.Constraints.Max.X
	xOffset := int(float32(fullWidth)*0.4) + 16
	offOpCol2 := op.Offset(image.Pt(xOffset, 0)).Push(gtx.Ops)
	gtx.Constraints.Max.X -= xOffset
	dims := material.Body1(th, allShortcuts[idx].desc).Layout(gtx)
	if dims.Size.Y > height {
		height = dims.Size.Y
	}
	offOpCol2.Pop()
	return D{Size: image.Pt(fullWidth, height)}
}
