package main

import (
	"image"
	"image/color"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/widget/material"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

type copyNotif struct {
	msg string
	at  time.Time
}

func (n *copyNotif) layout(gtx C, th *material.Theme) D {
	lbl := material.Body1(th, n.msg)
	lbl.Alignment = text.Middle
	lbl.Color = color.NRGBA{255, 255, 255, 255}
	lbl.Font.Weight = font.SemiBold
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
