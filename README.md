<img align="right" width="100" height="100" src="images/ieddata-icon-150x150.png" style="padding: 0 0 1ex 0.8em">

[![Siemens](https://img.shields.io/badge/github-siemens-009999?logo=github)](https://github.com/siemens)
[![Industrial Edge](https://img.shields.io/badge/github-industrial%20edge-e39537?logo=github)](https://github.com/industrial-edge)
[![Edgeshark](https://img.shields.io/badge/github-Edgeshark-003751?logo=github)](https://github.com/siemens/edgeshark)

# IED App Engine Data Access

[![PkgGoDev](https://pkg.go.dev/badge/github.com/siemens/ieddata)](https://pkg.go.dev/github.com/siemens/ieddata)
[![GitHub](https://img.shields.io/github/license/siemens/ieddata)](https://img.shields.io/github/license/siemens/ieddata)
![build and test](https://github.com/siemens/ieddata/workflows/build%20and%20test/badge.svg?branch=main)
![file descriptors](https://img.shields.io/badge/file%20descriptors-not%20leaking-success)
[![Go Report Card](https://goreportcard.com/badge/github.com/siemens/ieddata)](https://goreportcard.com/report/github.com/siemens/ieddata)
![Coverage](https://img.shields.io/badge/Coverage-91.4%25-brightgreen)

`ieddata` is part of the "Edgeshark" project that consist of several
repositories:
- [Edgeshark Hub repository](https://github.com/siemens/edgeshark)
- [G(h)ostwire discovery service](https://github.com/siemens/ghostwire)
- [Packetflix packet streaming service](https://github.com/siemens/packetflix)
- [Containershark Extcap plugin for
  Wireshark](https://github.com/siemens/cshargextcap)
- support modules:
  - [csharg (CLI)](https://github.com/siemens/csharg)
  - [mobydig](https://github.com/siemens/mobydig)
  - 🖝 **ieddata** 🖜

## Installation

```sh
go get github.com/siemens/ieddata@latest
```

Note: `ieddata` supports versions of Go 1 that are noted by the [Go release
policy](https://golang.org/doc/devel/release.html#policy), that is, major
versions _N_ and _N_-1 (where _N_ is the current major version).

## Usage

This example queries a Siemens Industrial Edge (virtual) device's...
- ...device name and its owner,
- ...and the list of installed applications.

Error handling has been left out for brevity.

```go
db, _ := ieddata.Open("platformbox.db")
defer db.Close()

di, _ := db.DeviceInfo()
fmt.Printf("device name: %s\nowner name: %s\n", di["deviceName"], di["ownerName"])

apps, _ := db.Apps()
slices.SortFunc(apps, func(a, b ieddata.App) bool { return a.Title < b.Title })
for _, app := range apps {
   fmt.Printf("app: %q %s\n", app.Title, app.Id)
}
```

> _Nota bene:_ the IED's `platformbox.db` is opened always in read-only mode.
> Thus, the underlying database connection on purpose does not allow callers to
> modify the IED's databases using this module.

## View Package Documentation Locally

```sh
make pkgsite
```

Then navigate to http://localhost:6060/github.com/siemens/ieddata.

## Deployment

Please note that an application using this Go module needs to have capability
`CAP_SYS_PTRACE` requested in its composer file, as well as your binary needs to
be executed with `CAP_SYS_PTRACE` also _effective_, not just permitted. If you
run your binary under a non-UID0 user, then you need to assign file capabilities
to your binary (see also
[setcap(8)](https://man7.org/linux/man-pages/man8/setcap.8.html)).

Additionally, you need to deploy any container that leverages this Go module
with `pid:host` in order to access the file system view (mount namespaces) of
other containers.

## Unit Tests

This Go module comes with unit tests and some of these unit tests need to be run
as root (UID 0). The unit tests additionally require an operational Docker
container engine to be up and running (for simulating a fake IED edge core
runtime container). To run the tests, simply do:

```sh
make test
```

The unit tests also include checks for file descriptor leaks.

# Contributing

Please see [CONTRIBUTING.md](CONTRIBUTING.md).

## License and Copyright

(c) Siemens AG 2023

[SPDX-License-Identifier: MIT](LICENSE)
