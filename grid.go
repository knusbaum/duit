package duit

import (
	"fmt"
	"image"

	"9fans.net/go/draw"
)

type Grid struct {
	Kids    []*Kid
	Columns int
	Valign  []Valign
	Halign  []Halign
	Padding []Space // in low DPI pixels
	Width   int     // -1 means full width, 0 means automatic width, >0 means exactly that many lowdpi pixels

	widths  []int
	heights []int
	size    image.Point
}

var _ UI = &Grid{}

func (ui *Grid) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout("Grid", self)
	if kidsLayout(dui, self, ui.Kids, force) {
		return
	}

	if ui.Valign != nil && len(ui.Valign) != ui.Columns {
		panic(fmt.Sprintf("len(valign) = %d, should be ui.Columns = %d", len(ui.Valign), ui.Columns))
	}
	if ui.Halign != nil && len(ui.Halign) != ui.Columns {
		panic(fmt.Sprintf("len(halign) = %d, should be ui.Columns = %d", len(ui.Halign), ui.Columns))
	}
	if ui.Padding != nil && len(ui.Padding) != ui.Columns {
		panic(fmt.Sprintf("len(padding) = %d, should be ui.Columns = %d", len(ui.Padding), ui.Columns))
	}
	if len(ui.Kids)%ui.Columns != 0 {
		panic(fmt.Sprintf("len(kids) = %d, should be multiple of ui.Columns = %d", len(ui.Kids), ui.Columns))
	}

	scaledWidth := dui.Scale(ui.Width)
	if scaledWidth > 0 && scaledWidth < sizeAvail.X {
		ui.size.X = scaledWidth
	}

	ui.widths = make([]int, ui.Columns) // widths include padding
	spaces := make([]Space, ui.Columns)
	if ui.Padding != nil {
		for i, pad := range ui.Padding {
			spaces[i] = dui.ScaleSpace(pad)
		}
	}
	width := 0                       // total width so far
	x := make([]int, len(ui.widths)) // x offsets per column
	x[0] = 0

	// first determine the column widths
	for col := 0; col < ui.Columns; col++ {
		if col > 0 {
			x[col] = x[col-1] + ui.widths[col-1]
		}
		ui.widths[col] = 0
		newDx := 0
		space := spaces[col]
		for i := col; i < len(ui.Kids); i += ui.Columns {
			k := ui.Kids[i]
			k.UI.Layout(dui, k, image.Pt(sizeAvail.X-width-space.Dx(), sizeAvail.Y-space.Dy()), true)
			if k.R.Dx()+space.Dx() > newDx {
				newDx = k.R.Dx() + space.Dx()
			}
		}
		ui.widths[col] = newDx
		width += ui.widths[col]
	}
	if scaledWidth < 0 && width < sizeAvail.X {
		leftover := sizeAvail.X - width
		given := 0
		for i, _ := range ui.widths {
			x[i] += given
			var dx int
			if i == len(ui.widths)-1 {
				dx = leftover - given
			} else {
				dx = leftover / len(ui.widths)
			}
			ui.widths[i] += dx
			given += dx
		}
	}

	// now determine row heights
	ui.heights = make([]int, (len(ui.Kids)+ui.Columns-1)/ui.Columns)
	height := 0                       // total height so far
	y := make([]int, len(ui.heights)) // including padding
	y[0] = 0
	for i := 0; i < len(ui.Kids); i += ui.Columns {
		row := i / ui.Columns
		if row > 0 {
			y[row] = y[row-1] + ui.heights[row-1]
		}
		rowDy := 0
		for col := 0; col < ui.Columns; col++ {
			space := spaces[col]
			k := ui.Kids[i+col]
			k.UI.Layout(dui, k, image.Pt(ui.widths[col]-space.Dx(), sizeAvail.Y-y[row]-space.Dy()), true)
			offset := image.Pt(x[col], y[row]).Add(space.Topleft())
			k.R = k.R.Add(offset) // aligned in top left, fixed for halign/valign later on
			if k.R.Dy()+space.Dy() > rowDy {
				rowDy = k.R.Dy() + space.Dy()
			}
		}
		ui.heights[row] = rowDy
		height += ui.heights[row]
	}

	// now shift the kids for right valign/halign
	for i, k := range ui.Kids {
		row := i / ui.Columns
		col := i % ui.Columns
		space := spaces[col]

		valign := ValignTop
		halign := HalignLeft
		if ui.Valign != nil {
			valign = ui.Valign[col]
		}
		if ui.Halign != nil {
			halign = ui.Halign[col]
		}
		cellSize := image.Pt(ui.widths[col], ui.heights[row]).Sub(space.Size())
		spaceX := 0
		switch halign {
		case HalignLeft:
		case HalignMiddle:
			spaceX = (cellSize.X - k.R.Dx()) / 2
		case HalignRight:
			spaceX = cellSize.X - k.R.Dx()
		}
		spaceY := 0
		switch valign {
		case ValignTop:
		case ValignMiddle:
			spaceY = (cellSize.Y - k.R.Dy()) / 2
		case ValignBottom:
			spaceY = cellSize.Y - k.R.Dy()
		}
		k.R = k.R.Add(image.Pt(spaceX, spaceY))
	}

	ui.size = image.Pt(width, height)
	if ui.Width < 0 && ui.size.X < sizeAvail.X {
		ui.size.X = sizeAvail.X
	}
	self.R = rect(ui.size)
}

func (ui *Grid) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	kidsDraw("Grid", dui, self, ui.Kids, ui.size, img, orig, m, force)
}

func (ui *Grid) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	return kidsMouse(dui, self, ui.Kids, m, origM, orig)
}

func (ui *Grid) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	return kidsKey(dui, self, ui.Kids, k, m, orig)
}

func (ui *Grid) FirstFocus(dui *DUI) *image.Point {
	return kidsFirstFocus(dui, ui.Kids)
}

func (ui *Grid) Focus(dui *DUI, o UI) *image.Point {
	return kidsFocus(dui, ui.Kids, o)
}

func (ui *Grid) Mark(self *Kid, o UI, forLayout bool, state State) (marked bool) {
	return kidsMark(self, ui.Kids, o, forLayout, state)
}

func (ui *Grid) Print(self *Kid, indent int) {
	PrintUI(fmt.Sprintf("Grid columns=%d padding=%v", ui.Columns, ui.Padding), self, indent)
	kidsPrint(ui.Kids, indent+1)
}
