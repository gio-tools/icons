# gio-icon-browser

[![GitHub CI](https://github.com/steverusso/gio-icon-browser/actions/workflows/ci.yaml/badge.svg)](https://github.com/steverusso/gio-icon-browser/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/steverusso/gio-icon-browser)](https://goreportcard.com/report/github.com/steverusso/gio-icon-browser)
[![Go Reference](https://pkg.go.dev/badge/github.com/steverusso/gio-icon-browser.svg)](https://pkg.go.dev/github.com/steverusso/gio-icon-browser)

```
go install github.com/steverusso/gio-icon-browser@latest
```

A browser of every icon in the
[golang.org/x/exp/shiny/materialdesign/icons](https://pkg.go.dev/golang.org/x/exp/shiny/materialdesign/icons)
package, built with and for [Gio](https://gioui.org/).

## Development

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

## Acknowledgements

The idea and name are inspired by the `gtk3-icon-browser` and
`gtk4-icon-browser`. The use of `golang.org/x/tool/go/packages` to generate the
icon data was from looking at [pierrec's
iconx](https://git.sr.ht/~pierrec/giox/tree/main/item/cmd/iconx).

## License

This is free and unencumbered software released into the public domain. Please
see the [UNLICENSE](./UNLICENSE) file for more information.
