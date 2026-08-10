# Contributing

Thanks for considering a contribution to context-debug.

## Reporting bugs / requesting features

Open an issue using the appropriate template. Include a minimal reproduction
where possible.

## Development

```
go test ./...              # run the package tests (root module)
cd example && go run .     # run the example HTTP service (needs POSTGRES_DSN)
```

There is no separate lint config; rely on `go vet` and standard `gofmt`
(`gofmt -l .` should report nothing before you open a PR).

## Submitting changes

1. Fork the repo and create a branch off `main`.
2. Make your change, keeping it focused — unrelated cleanups belong in a
   separate PR.
3. Ensure `go vet ./...`, `gofmt -l .`, and `go test ./...` are clean.
4. Open a pull request using the PR template, describing the motivation and
   what changed.

By contributing, you agree your contributions will be licensed under this
project's [license](LICENSE).
