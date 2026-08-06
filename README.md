# tdl

This is the `tdl` CLI tool used for running the interactive trainings on the [Three Dots Labs Academy](https://academy.threedots.tech).

## Install

### Brew (macOS) - recommended

```sh
brew install ThreeDotsLabs/tap/tdl
```

### Script (macOS, Linux) - recommended

```sh
sudo /bin/sh -c "$(curl -fsSL https://raw.githubusercontent.com/ThreeDotsLabs/cli/master/install.sh)" -- -b /usr/local/bin
```

### Nix (macOS, Linux)

```sh
nix profile add github:ThreeDotsLabs/cli
```

### Script (Windows)

Install to your home directory:

```sh
iwr https://raw.githubusercontent.com/ThreeDotsLabs/cli/master/install.ps1 -useb | iex
```

Or install in any chosen path:

```sh
$env:TDL_INSTALL = 'bin\'
iwr https://raw.githubusercontent.com/ThreeDotsLabs/cli/master/install.ps1 -useb | iex
```

### Binaries

Download the latest binary from GitHub and move it to a directory in your `$PATH`.

[See Releases](https://github.com/ThreeDotsLabs/cli/releases)

### From source

```sh
go install github.com/ThreeDotsLabs/cli/tdl@latest
```

