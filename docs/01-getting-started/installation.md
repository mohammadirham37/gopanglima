# Installation

goPanglima requires **Go 1.23 or newer**. Check your toolchain:

```bash
go version
```

Create a module and add goPanglima:

```bash
mkdir myapp && cd myapp
go mod init example.com/myapp
go get github.com/mohammadirham37/gopanglima@latest
```

> The `gopanglima` CLI and `gopanglima new` project generator are planned for
> a later milestone and do not exist yet. For now, create applications by hand
> as shown in [Your first application](first-app.md).

Related: [Your first application](first-app.md), [Configuration](../02-foundation/configuration.md).
