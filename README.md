# icons

[![GitHub CI](https://github.com/gio-tools/icons/actions/workflows/ci.yaml/badge.svg)](https://github.com/gio-tools/icons/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/gio.tools/icons)](https://goreportcard.com/report/gio.tools/icons)
[![Go Reference](https://pkg.go.dev/badge/gio.tools/icons.svg)](https://pkg.go.dev/gio.tools/icons)

```
go get gio.tools/icons
```

This package contains all of the icons in
[golang.org/x/exp/shiny/materialdesign/icons](https://pkg.go.dev/golang.org/x/exp/shiny/materialdesign/icons)
as [Gio](https://gioui.org) icon widgets.

## Icon Browser

```
go install gio.tools/icons/cmd/gio-icon-browser@latest
```

This project also has a browser of every icon in the
[golang.org/x/exp/shiny/materialdesign/icons](https://pkg.go.dev/golang.org/x/exp/shiny/materialdesign/icons)
package, built with and for [Gio](https://gioui.org/).

#### Native

To build the app, run `go build` (or just `go build -tags nowayland` for no Wayland
support).

If you have [`task`](https://github.com/go-task/task),
[`goimports`](https://pkg.go.dev/golang.org/x/tools/cmd/goimports),
[`gofumpt`](https://github.com/mvdan/gofumpt) and
[`staticcheck`](https://github.com/dominikh/go-tools) installed, you can simply run `task`
(or `task nowayland`) to fmt, lint and build the project.

#### WebAssembly

If you have [`gogio` installed](https://gioui.org/doc/install/wasm), you can run `task
wasm` to build the web assembly assets. To view these in a browser, serve up the files in
the `wasm_assets` directory.

### Acknowledgements

The idea and name are inspired by the `gtk3-icon-browser` and
`gtk4-icon-browser`. The use of `golang.org/x/tool/go/packages` to generate the
icon data was from looking at [pierrec's
iconx](https://git.sr.ht/~pierrec/giox/tree/main/item/cmd/iconx).

## License

This is free and unencumbered software released into the public domain. Please
see the [UNLICENSE](./UNLICENSE) file for more information.
