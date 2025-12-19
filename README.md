## What is Dagger?

Dagger is an incremental computation engine for CI/CD. It lets you
write pipelines as code and runs them in containers.

## Properties

**Hermetic** — Every operation runs in a container. No implicit
dependencies on host state.

**Incremental** — Operations are cached by the SHA of their inputs.
Unchanged subgraphs are skipped entirely.

**Local-first** — Pipelines run the same on your laptop as in CI.
Debug locally, push when it works.

**Programmable** — Pipelines are functions in Go, Python, or
TypeScript. Use variables, loops, conditionals, and tests.

## Use cases

**End-to-end testing** — Spin up databases, queues, and services as
containers. Run your test suite against them. Tear down automatically.
Cache the setup so subsequent runs are fast.

**Build pipelines** — Multi-stage builds with dependency caching.
Matrix builds across platforms. Publish to registries.

**Dev environments** — Define your environment as code. Share it
across the team. Run it locally or in CI.

## Example

An end-to-end test that spins up Postgres, runs migrations, and
tests against it:

```go
func Test(ctx context.Context, src *dagger.Directory) (string, error) {
    // Start Postgres as a service
    db := dag.Container().
        From("postgres:16").
        WithEnvVariable("POSTGRES_PASSWORD", "test").
        WithExposedPort(5432).
        AsService()

    // Run tests with database attached
    return dag.Container().
        From("golang:1.21").
        WithDirectory("/src", src).
        WithWorkdir("/src").
        WithServiceBinding("db", db).
        WithEnvVariable("DATABASE_URL", "postgres://postgres:test@db:5432/postgres").
        WithExec([]string{"go", "run", "./cmd/migrate"}).
        WithExec([]string{"go", "test", "./..."}).
        Stdout(ctx)
}
```

```
dagger call test --src=.
```

First run: pulls images, starts Postgres, runs migrations, runs tests.
Second run: if `src` hasn't changed, returns cached result instantly.

## Install

```
curl -fsSL https://dl.dagger.io/install.sh | sh
```

## Documentation

- [Quickstart](https://docs.dagger.io/quickstart)
- [API reference](https://docs.dagger.io/api)
- [Module registry](https://daggerverse.dev)

## Community

- [Discord](https://discord.gg/dagger-io)
- [GitHub](https://github.com/dagger/dagger)
