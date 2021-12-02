# hedgedgrpc

[![build-img]][build-url]
[![pkg-img]][pkg-url]
[![reportcard-img]][reportcard-url]
[![coverage-img]][coverage-url]
[![version-img]][version-url]

Hedged Go GRPC client which helps to reduce tail latency at scale.

## Rationale

See paper [Tail at Scale](https://cacm.acm.org/magazines/2013/2/160173-the-tail-at-scale/fulltext) by Jeffrey Dean, Luiz André Barroso. In short: the client first sends one request, but then sends an additional request after a timeout if the previous hasn't returned an answer in the expected time. The client cancels remaining requests once the first result is received.

## Note

See also [hedgedhttp](https://github.com/cristalhq/hedgedhttp) for Go `net/http`.

## Features

* Simple API.
* Easy to integrate.
* Optimized for speed.
* Clean and tested code.

## Install

Go version 1.17+

```
go get github.com/cristalhq/hedgedgrpc
```

## Example

```go
```

Also see examples: [examples_test.go](https://github.com/cristalhq/hedgedgrpc/blob/main/examples_test.go).

## Documentation

See [these docs][pkg-url].

## License

[MIT License](LICENSE).

[build-img]: https://github.com/cristalhq/hedgedgrpc/workflows/build/badge.svg
[build-url]: https://github.com/cristalhq/hedgedgrpc/actions
[pkg-img]: https://pkg.go.dev/badge/cristalhq/hedgedgrpc
[pkg-url]: https://pkg.go.dev/github.com/cristalhq/hedgedgrpc
[reportcard-img]: https://goreportcard.com/badge/cristalhq/hedgedgrpc
[reportcard-url]: https://goreportcard.com/report/cristalhq/hedgedgrpc
[coverage-img]: https://codecov.io/gh/cristalhq/hedgedgrpc/branch/main/graph/badge.svg
[coverage-url]: https://codecov.io/gh/cristalhq/hedgedgrpc
[version-img]: https://img.shields.io/github/v/release/cristalhq/hedgedgrpc
[version-url]: https://github.com/cristalhq/hedgedgrpc/releases
