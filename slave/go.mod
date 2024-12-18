module slave

go 1.23.2

require (
	common v0.0.0
	github.com/fatih/color v1.18.0
	google.golang.org/protobuf v1.36.0
	protocol v0.0.0
)

require (
	github.com/goccy/go-yaml v1.15.9 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.26.0 // indirect
)

// needed for renovate to work
replace (
	common => ../common
	protocol => ../protocol
)
