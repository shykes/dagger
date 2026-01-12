# Dagger Messaging Brief

## Proposed Identity

> **Dagger is an incremental artifact engine.**

Three words. Each carries weight:

- **Incremental** — Only re-run what changed. Cached by content.
- **Artifact** — Typed objects (containers, directories, files), not bytes.
- **Engine** — A runtime that executes computation.

### A Note on "Incremental" and Determinism

"Incremental" signals caching and efficiency, but does not explicitly signal:
- **Deterministic** (same inputs → same outputs)
- **Hermetic** (sandboxed, isolated)
- **Repeatable** (reproducible across environments)

These properties are *prerequisites* for incrementality (you can't cache non-deterministic results), but they're not explicit in the word itself.

**Solution:** Lead with "incremental artifact engine" as the memorable identity, then immediately anchor trust with properties:

```markdown
Dagger is an incremental artifact engine.

**Deterministic.** Same inputs produce same outputs. Always.
**Hermetic.** Every function runs in a container.
**Incremental.** Cached by content. Only re-run what changed.
```

This makes "deterministic" the first thing readers see after the identity — the trust anchor that positions us as the Yin to AI's Yang.

---

## Primary Use Case: End-to-End Testing

On top of the core identity, we lead with a concrete use case:

> **Dagger is the complete solution for end-to-end testing.**

Why e2e testing is the right wedge:

| Reason | Detail |
|--------|--------|
| **Painful** | E2e tests are slow, flaky, hard to debug |
| **Underserved** | No dominant "e2e testing platform" exists |
| **Showcases all properties** | Hermetic (spin up services), incremental (cache setup), local-first (debug before push) |
| **Not AI hype** | Grounded, practical, relatable |
| **Expandable** | Once you're in for e2e, you use it for builds, generators, etc. |

### Messaging Layers

```
┌─────────────────────────────────────────────────┐
│  "The complete solution for end-to-end testing" │  ← Hook (use case)
├─────────────────────────────────────────────────┤
│  "Powered by an incremental artifact engine"    │  ← Foundation (identity)
├─────────────────────────────────────────────────┤
│  Deterministic. Hermetic. Cached.               │  ← Properties (trust)
└─────────────────────────────────────────────────┘
```

---

## Why This Identity

### The Problem with Current Positioning

The current README leads with "runtime for composable workflows" and emphasizes AI agents prominently. This creates two issues:

1. **"Composable workflows" is generic** — Could describe Airflow, Temporal, or any orchestration tool.

2. **AI-forward positioning feels like hype** — System engineers are skeptical. In a world of probabilistic AI, they crave determinism.

### What Makes Dagger Actually Different

Dagger is not just another function platform. The key insight:

**Other platforms (Lambda, Workers):**
```
function(bytes) → bytes
```
The runtime doesn't understand the data.

**Dagger:**
```
function(Container, Directory) → Container
```
The runtime deeply understands typed artifacts. They have methods. They persist state. They cache by content.

This enables:
- Intelligent caching (knows when a Directory changed by content, not timestamp)
- Cross-language composition (TypeScript calls Go function, gets a Container back with full method access)
- Lazy evaluation (the graph executes only what's needed)

"Incremental artifact engine" captures this unique architecture in three words.

---

## Brand Values

We want engineers to associate Dagger with:

| Value | Expression |
|-------|------------|
| **Trust** | Deterministic. Same inputs, same outputs. Always. |
| **Reliability** | The solid foundation you build on. |
| **Simplicity** | Like a Lego brick — small, sturdy, composable. |
| **Authenticity** | Technical substance, not marketing claims. |

### The Yin-Yang Positioning

In an era of generative AI, Dagger is the verification layer:

- AI generates code → Dagger checks if it works
- AI proposes changes → Dagger runs the tests
- AI is probabilistic → Dagger is deterministic

We are the **Yin to AI's Yang**: opposite and complementary. AI needs something trustworthy to act on. Dagger is that thing.

This lets us acknowledge AI capabilities without joining the hype. Dagger is essential *infrastructure* for the AI era, not another AI product.

---

## Architecture: Plumbing and Porcelain

```
┌─────────────────────────────────────────┐
│  dagger check    dagger generate        │  ← Porcelain (conventions)
├─────────────────────────────────────────┤
│  dagger call <function>                 │  ← Plumbing (primitives)
├─────────────────────────────────────────┤
│  Typed artifacts with methods           │  ← Core model
├─────────────────────────────────────────┤
│  Incremental, cached, hermetic          │  ← Properties
└─────────────────────────────────────────┘
```

**Functions** are the primitive. Every function:
- Runs in a container (hermetic)
- Passes typed artifacts, not bytes (typed)
- Is cached by content hash (incremental)
- Works the same locally and in CI (portable)

**Checks and generators** are conventions built on functions:
- `dagger check` — run all validations (tests, linters)
- `dagger generate` — run all generators (bindings, docs)

**Toolchains** package functions for common workflows:
- `dagger toolchain install github.com/dagger/dagger/toolchains/go`
- Provides build, test, lint out of the box
- User doesn't write code — just installs and runs

---

## Proposed User Journey

### Default Path (No Code Required)

1. `dagger init`
2. `dagger toolchain install <language>`
3. `dagger check` — validate project
4. `dagger generate` — generate artifacts
5. Add `dagger check` to CI — done

### Advanced Path (Power Users)

1. Write custom toolchain with functions
2. Mark functions as `// +check` or `// +generator`
3. They appear in `dagger check -l` and `dagger generate -l`
4. Share via Daggerverse

This mirrors how most tools work: use conventions first, customize later.

---

## Key Messages by Audience

### For DevOps/Platform Engineers
> Dagger is an incremental artifact engine. Install a toolchain, run `dagger check`, get the same results locally and in CI. No YAML. No "works on my machine."

### For Developers
> Run your CI checks locally before you push. Same containers. Same result. Debug failures on your laptop, not in CI logs.

### For Architects
> Functions pass typed artifacts — containers, directories, files — not bytes. The runtime understands each type, enabling content-addressed caching and cross-language composition.

### For AI/Agent Builders
> AI can generate code. Dagger tells you if it works. Hermetic execution, deterministic results, full observability. The reliable substrate for agentic workflows.

---

## README Structure (Proposed)

```markdown
## What is Dagger?

Dagger is an incremental artifact engine.

Functions pass typed artifacts — containers, directories, files — not
bytes. Artifacts have methods. Results are cached by content. Pipelines
run the same locally and in CI.

## Quick Start

[Install → Add toolchain → Run checks → Add to CI]

## How It Works

[Functions, artifacts, caching, porcelain/plumbing]

## Properties

- Hermetic
- Incremental
- Portable
- Composable

## Toolchains

[List of available toolchains]

## Documentation / Community
```

---

## Open Questions for Discussion

1. **"Artifact" ambiguity** — Could be confused with artifact storage (Artifactory). Do we need to clarify "artifacts with methods" immediately?

2. **AI positioning** — How prominently should AI/agents appear? Current proposal: acknowledge as a use case, don't lead with it.

3. **Visual identity** — The Lego brick metaphor suggests: solid colors, simple shapes, satisfying "click" interactions. Does this resonate?

4. **Standalone runners** — How soon can we message "bypass CI entirely"? This is a strong differentiator but needs to be real.

---

## Next Steps

- [ ] Review and align on identity ("incremental artifact engine")
- [ ] Validate technical accuracy with engineering
- [ ] Draft new README
- [ ] Update docs landing page
- [ ] Design visual assets that reinforce trust/reliability
- [ ] Plan quickstart around toolchain-first journey
