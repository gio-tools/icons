# gio-icon-browser

A browser of every icon in the
[golang.org/x/exp/shiny/materialdesign/icons](https://pkg.go.dev/golang.org/x/exp/shiny/materialdesign/icons)
package, built with and for [Gio](https://gioui.org/).

## Development

To build the app, run `go build` (or just `go build -tags nowayland` for no Wayland
support).

If you have [`task`](https://github.com/go-task/task),
[`goimports`](https://pkg.go.dev/golang.org/x/tools/cmd/goimports) and
[`gofumpt`](https://github.com/mvdan/gofumpt) installed, you can simply run
`task` (or `task nowayland`) to fmt, lint and build the project.

## Acknowledgements

The idea and name are inspired by the `gtk3-icon-browser` and
`gtk4-icon-browser`. The use of `golang.org/x/tool/go/packages` to generate the
icon data was from looking at [pierrec's
iconx](https://git.sr.ht/~pierrec/giox/tree/main/item/cmd/iconx).

## License

This is free and unencumbered software released into the public domain. Please
see the [UNLICENSE](./UNLICENSE) file for more information.
