module github.com/ThreeDotsLabs/cli

go 1.17

require (
	github.com/BurntSushi/toml v0.4.1
	github.com/ThreeDotsLabs/cli/course/genproto v0.0.0-00010101000000-000000000000
	github.com/fatih/color v1.12.0
	github.com/golang/protobuf v1.5.2
	github.com/hexops/gotextdiff v1.0.3
	github.com/pkg/errors v0.8.1
	github.com/sergi/go-diff v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1
	google.golang.org/grpc v1.40.0
)

require (
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4 // indirect
	golang.org/x/sys v0.0.0-20210510120138-977fb7262007 // indirect
	golang.org/x/text v0.3.5 // indirect
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)

replace github.com/ThreeDotsLabs/cli/course/genproto => ./course/genproto
