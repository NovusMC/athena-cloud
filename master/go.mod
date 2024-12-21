module master

go 1.23.2

require (
	common v0.0.0
	github.com/ergochat/readline v0.1.3
	github.com/fatih/color v1.18.0
	github.com/goccy/go-yaml v1.15.13
	github.com/gokrazy/rsync v0.1.0
	github.com/urfave/cli/v3 v3.0.0-beta1
	google.golang.org/protobuf v1.36.0
	protocol v0.0.0
)

require (
	github.com/DavidGamba/go-getoptions v0.23.0 // indirect
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mmcloughlin/md4 v0.1.1 // indirect
	golang.org/x/crypto v0.27.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.18.0 // indirect
)

// needed for renovate to work
replace (
	common => ../common
	protocol => ../protocol
)
