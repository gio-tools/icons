package icons

import "gioui.org/widget"

// The function `MustIcon` is primarily used to parse each icon's source. Shortening the
// name to `mi` reduces the generated file size by about 6kb.
var mi = MustIcon

// MustIcon returns a new `*widget.Icon` for the given byte slice or panics on error.
func MustIcon(data []byte) *widget.Icon {
	ic, err := widget.NewIcon(data)
	if err != nil {
		panic(err)
	}
	return ic
}
