## What is Dagger?

Dagger is an open-source, agent-ready platform for end-to-end testing.

AI coding agents are fast but unreliable — they need an external system for trusted feedback on every code change. Dagger provides that feedback: repeatable test execution that agents can call as they develop, and CI can verify during review. It runs locally, in CI, or directly in the cloud.

```
brew install dagger/tap/dagger
```

## Why Dagger?


### Repeatable

With LLMs, code changes are nearly instant. The bottleneck is now trusted feedback: you can only ship as fast as you can get repeatable test results.

Dagger is designed for repeatability: tests run in containers; orchestration logic runs in sandboxed functions; host dependencies are explicit and strictly typed; intermediate artifacts are built just-in-time; everything is cached by default with fine-grained control. Same inputs, same outputs.

### Local-first

Local execution is a core feature. Once configured, Dagger runs your tests reliably on any supported system. The only dependency is a recent Linux kernel. On non-Linux systems, Docker Desktop and similar products are supported out of the box.

### Programmable

Shell scripts and proprietary YAML are no longer acceptable for test orchestration. Dagger provides a complete platform: a container and function runtime, system API, cross-language type system, SDKs for 8 languages, and an interactive REPL.

```go
// Example: test a Go project
func (m *MyModule) Test(ctx context.Context, source *dagger.Directory) (string, error) {
    return dag.Container().
        From("golang:1.23").
        WithDirectory("/src", source).
        WithWorkdir("/src").
        WithExec([]string{"go", "test", "./..."}).
        Stdout(ctx)
}
```

### Observable

Built-in tracing, logs, and metrics show exactly what's happening at every step. Debug complex workflows immediately instead of guessing from a wall of text.

<p align="center"><img src="docs/static/img/readme/cloud-trace.gif" width="60%"></p>

### Open

The engine, CLI, and SDKs are open-source. We use open standards: OpenTelemetry for observability, OCI for containers, GraphQL for the API. Our [commercial product](https://dagger.io/cloud) enhances the open ecosystem rather than competing with it.


## Getting started

- [Documentation](https://docs.dagger.io)
- [Quickstart](https://docs.dagger.io/quickstart)
- [AI Agents Guide](https://docs.dagger.io/ai-agents)

## Community

- [Discord](https://discord.gg/dagger-io) — ~5,000 members
- [GitHub Discussions](https://github.com/dagger/dagger/discussions)
- [Twitter](https://twitter.com/dagger_io)
- [Community Page](https://dagger.io/community)

## Contributing

See [CONTRIBUTING.md](https://github.com/dagger/dagger/blob/main/CONTRIBUTING.md).
