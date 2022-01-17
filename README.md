# entangle

A distributed peer-to-peer filesystem.

[![Go Reference](https://pkg.go.dev/badge/github.com/alphahorizonio/entangle.svg)](https://pkg.go.dev/github.com/alphahorizonio/entangle)

## Overview

`entangle` is a file-sharing and storing solution built on top of [`stfs`](https://github.com/pojntfx/stfs), [`sile-fystem`](https://github.com/JakWai01/sile-fystem) and [`libentangle`](https://github.com/alphahorizonio/libentangle).

## Installation

```bash
go install github.com/alphahorizonio/entangle@latest
```

## Usage 

Start a server containing the remote file which is used as a backend.

```shell
entangle server
```

Start a client to mount the fuse and access the remote backend.

```shell
entangle client --metadata /tmp/stfs-metadata-$(date +%s).sqlite --mountpoint $HOME/Downloads/mount
```

For more information, consider using `entangle --help`.

## Contributing

1. Fork it
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am "feat: Add something"`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create Pull Request

## License 

entangle (c) 2022 Jakob Waibel and contributors

SPDX-License-Identifier: AGPL-3.0
