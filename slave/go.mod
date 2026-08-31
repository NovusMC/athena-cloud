module slave

go 1.25.0

require (
	common v0.0.0
	github.com/fatih/color v1.19.0
	google.golang.org/protobuf v1.36.0
	protocol v0.0.0
)

require (
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/goccy/go-yaml v1.15.13 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.42.0 // indirect
)

// needed for renovate to work
replace (
	common => ../common
	protocol => ../protocol
)
