# Contributing to CopyTyGo

Bug reports, documentation fixes, and code contributions are welcome. For framework usage and examples, see the [CopyTyGo v5 documentation](https://copytygo.org/docs/v5/).

## Report a bug or propose a change

Search [existing issues](https://github.com/arfajhf/copytygo/issues) before opening one. For a bug, include the CopyTyGo version, Go version, operating system, steps to reproduce, and what you expected to happen. Include relevant error output with credentials and other private data removed.

For a new feature or a change to a public API or CLI command, consider opening an issue first to discuss the use case and scope.

## Develop locally

Fork the repository, clone your fork, and create a branch from an up-to-date `main`. Use the Go version specified in [go.mod](go.mod). If you are checking a generated frontend, use Node.js 22 or later.

For Go changes, format the edited files with `gofmt` and run these checks from the repository root:

```sh
go test ./...
go vet ./...
go build ./...
```

Add or update tests for behavior changes where applicable. For documentation changes, review the rendered Markdown and check the links. The [CI workflow](.github/workflows/ci.yml) lists the checks GitHub runs.

## Open a pull request

- Target `main` in the upstream repository and keep the pull request focused on one change.
- Describe what changed and why, link any related issue, and list the checks you ran.
- Call out changes to public APIs, CLI behavior, or generated projects so reviewers can assess their impact.
