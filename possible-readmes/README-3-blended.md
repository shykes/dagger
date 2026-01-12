## What is Dagger?

Dagger is an incremental artifact engine.

Functions pass typed artifacts — containers, directories, files. Each has
methods. Each result is cached by content. Runs the same locally and in CI.

**Hermetic.** **Incremental.** **Local-first.** **Programmable.**

Perfect for end-to-end tests: spin up services, run your suite, cache
the setup. Debug before you push.

## Quick start

Install Dagger:

```
curl -fsSL https://dl.dagger.io/install.sh | sh
```

Initialize your project and install a toolchain:

```
dagger init
dagger toolchain install github.com/dagger/dagger/toolchains/go
```

Run all checks:

```
dagger check
```

First run executes. Second run returns cached results if inputs unchanged.

## CI integration

```yaml
- uses: actions/checkout@v4
- run: curl -fsSL https://dl.dagger.io/install.sh | sh
- run: dagger check
```

Same functions. Same containers. Same result.

## How it works

Toolchains expose functions. Some functions are checks. Some are generators.
All are callable directly:

```
dagger check -l                    # list checks
dagger check                       # run all checks
dagger check go:lint               # run one check

dagger call go build --source=.    # call a function directly
```

`dagger check` and `dagger generate` are conventions.
`dagger call` is the primitive.

## Available toolchains

- [Go](https://github.com/dagger/dagger/toolchains/go) — build, test, lint, tidy

Browse more at [daggerverse.dev](https://daggerverse.dev)

## Writing custom toolchains

Toolchains are Dagger modules with functions. Mark a function as a check:

```go
// +check
func (m *MyToolchain) Lint(ctx context.Context) error {
    // ...
}
```

It appears in `dagger check -l` automatically.

See [Building Toolchains](https://docs.dagger.io/toolchains).

## Documentation

- [Quickstart](https://docs.dagger.io/quickstart)
- [API reference](https://docs.dagger.io/api)

## Community

- [Discord](https://discord.gg/dagger-io)
- [GitHub](https://github.com/dagger/dagger)
