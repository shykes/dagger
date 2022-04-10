---
slug: /1204/secrets
displayed_sidebar: europa
---

# The Dagger Secrets API

Secrets are a crucial part of any CI/CD pipeline, which is why Dagger introduced native support for secrets from the very beginning. The Dagger Secrets API makes it easy to incorporate secrets into your actions and plans, in a way that is *secure by default*:

* The core type `dagger.#Secret` holds a reference to a secret, not its contents, whih helps prevent credentials leaks.
* A plan may load secrets from virtually any source: a file, environment variable, KMS, vault service, local keychain, etc.
* Actions may receive secrets as inputs, and inject them in containers as temporary files, or environment variables.
* Acions may create new secrets with `core.#NewSecret`, and export them as outputs.

* Secrets And just in case you accidentally print a secret’s contents in your action logs… Dagger will automatically detect it and scrub it. You’re welcome!
Actions can now dynamically create their own secrets, using the #NewSecret core action.
Actions can now perform common operations against the contents of a secret, securely, without accessing the contents directly (to avoid leaks). Operations include extracting a secret from a JSON- or YAML-encoded envelope, and trimming space characters.
Secrets can now be injected into containers as environment variables, in addition to temporary files



## Loading secrets from the client

Most operations in `client` support handling secrets (see [Interacting with the client](./1203-client.md)). More specifically, you can:

- Write a secret to a file;
- Read a secret from a file;
- Read a secret from an environment variable;
- Read a secret from the output of a command;
- Use a secret as the input of a command.

## Environment

The simplest use case is reading from an environment variable:

```cue
dagger.#Plan & {
    client: env: GITHUB_TOKEN: dagger.#Secret
}
```

## File

You may need to trim the whitespace, especially when reading from a file:

```cue file=../tests/core-concepts/secrets/plans/file.cue
```

## SOPS

There’s many ways to store encrypted secrets in your git repository. If you use [SOPS](https://github.com/mozilla/sops), here's a simple example where you can access keys from an encrypted yaml file:

```yaml title="secrets.yaml"
myToken: ENC[AES256_GCM,data:AlUz7g==,iv:lq3mHi4GDLfAssqhPcuUIHMm5eVzJ/EpM+q7RHGCROU=,tag:dzbT5dEGhMnHbiRTu4bHdg==,type:str]
sops:
    ...
```

```cue file=../tests/core-concepts/secrets/plans/sops.cue title="main.cue"
```
