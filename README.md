# oci-systemd-hook-go

A *partial* Go implementation of the [oci-systemd-hook][oci-systemd-hook]. A
notable difference is this hook is designed to be run in the [`createContainer
Hook`][hook], instead of the `prestart Hook`.

## Build and Install

```
go build .
```

## Usage

See [usage](usage.md).

[oci-systemd-hook]: https://github.com/projectatomic/oci-systemd-hook
[hook]: https://github.com/opencontainers/runtime-spec/blob/master/config.md#posix-platform-hooks
[systemd-in-docker]: https://github.com/pierrchen/systemd-in-docker
