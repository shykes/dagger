# EnvFile Cache Coherency: Re-enabling System Environment Variable Expansion

## Context

**Branch**: `dotenv-system`
**Related PR**: [#11034](https://github.com/dagger/dagger/pull/11034)
**Discussion**: [Comment by jedevc](https://github.com/dagger/dagger/pull/11034#discussion_r2401382370)

## The Problem

EnvFile supports bash-style variable expansion in `.env` files:
```bash
# .env
MY_TOKEN=$HOME/secrets/token    # Expands $HOME from host environment
MY_PATH=${PWD}/data             # Expands ${PWD} from host environment
```

This expansion happens via `Host{}.GetEnv(ctx, name)` in:
- `core/envfile.go:116` - `variables()` method
- `core/envfile.go:194` - `Lookup()` method

**Currently disabled** (returns `""`) due to caching issues. This branch re-enables it.

### Why It Was Disabled

Without proper cache scoping:
1. **Cache collision**: Client A with `$HOME=/Users/alice` shares cache with Client B with `$HOME=/Users/bob`
2. **Wrong values**: Client B gets Alice's expanded values from cache
3. **Security risk**: Malicious module could cache-sniff other clients' env vars
4. **Inconsistency**: Same EnvFile returns different values depending on where it's evaluated

### The Cache Coherency Issue

**Critical location**: `core/modulesource.go:432`

```go
// ModuleSource digest includes expanded env var values
vars, err := src.UserDefaults.Variables(ctx, false) // false = expand variables

// Variables with $HOME get expanded here, but without client-scoped caching
for _, v := range vars {
    inputs = append(inputs, fmt.Sprintf("env:%s=%s", v.Name, v.Value))
}
```

Without `CachePerClient`, different clients' env vars pollute shared cache.

## The Solution

Apply `dagql.CachePerClient` pattern (already used by 19+ operations: `git()`, `host.directory()`, `currentModule()`, etc.)

### Required Changes

#### 1. Add Cache Keys to GraphQL Resolvers

**File**: `core/schema/envfile.go`

```diff
 // Line ~102
-dagql.NodeFunc("variables", s.variables,
+dagql.NodeFuncWithCacheKey("variables", s.variables,
+    dagql.CachePerClient[*core.EnvFile, variablesArgs],
     dagql.Args(
```

```diff
 // Line ~122
-dagql.NodeFunc("get", s.get,
+dagql.NodeFuncWithCacheKey("get", s.get,
+    dagql.CachePerClient[*core.EnvFile, getArgs],
     dagql.Args(
```

**Why**: Ensures each client's env var expansion is cached separately.

#### 2. Fix ModuleSource Digest Calculation

**File**: `core/modulesource.go:430-448`

**Option A** - Include client context in digest (recommended):
```go
// Include client ID when env vars are expanded
if src.UserDefaults != nil {
    clientMD, err := engine.ClientMetadataFromContext(ctx)
    if err != nil {
        return digest.FromString("")
    }

    vars, err := src.UserDefaults.Variables(ctx, false)
    if err != nil {
        slog.Error("failed to load user defaults", "error", err)
    } else {
        // Client ID becomes part of module identity when using host env vars
        inputs = append(inputs, fmt.Sprintf("client:%s", clientMD.ClientID))

        sort.Slice(vars, func(i, j int) bool {
            return vars[i].Name < vars[j].Name
        })
        for _, v := range vars {
            inputs = append(inputs, fmt.Sprintf("env:%s=%s", v.Name, v.Value))
        }
    }
}
```

**Option B** - Use raw (unexpanded) values only:
```go
// Don't expand variables in digest calculation
vars, err := src.UserDefaults.Variables(ctx, true) // true = raw, no expansion
```

**Trade-off**:
- Option A: Correct but creates per-client module caches (higher storage, lower reuse)
- Option B: Better caching but limits functionality (can't use expanded vars in module identity)

**Recommendation**: Option A for correctness. Users who need better cache reuse can avoid using `$VAR` expansion in their `.env` files.

#### 3. Update FIXME Comments

**File**: `core/envfile.go`

Remove outdated FIXME comments at:
- Line 112-115
- Line 189-193

Replace with explanation:
```go
// Expand variables using host environment via Host.GetEnv
// Cache coherency is maintained via CachePerClient on GraphQL resolvers
// See core/schema/envfile.go and ENVFILE_CACHE_COHERENCY.md
```

### Testing Requirements

**File**: `core/integration/envfile_cache_test.go` (new)

Required test cases:
1. **Multi-client isolation**: Two clients with different `$FOO` get different values
2. **Cache correctness**: Client A's cache doesn't leak to Client B
3. **Module digest stability**: Same inputs produce same digest within a client
4. **Module digest isolation**: Different client env vars produce different digests
5. **No expansion interference**: Variables without `$` syntax work identically across clients

Example test structure:
```go
func TestEnvFileMultiClientCacheIsolation(t *testing.T) {
    // Setup two clients with different env vars
    clientA := setupClientWithEnv(t, "FOO", "valueA")
    clientB := setupClientWithEnv(t, "FOO", "valueB")

    // Create .env with: MY_VAR=$FOO
    envFile := createEnvFile(t, "MY_VAR=$FOO")

    // Both clients query same envFile
    varsA := clientA.EnvFile(envFile).Variables()
    varsB := clientB.EnvFile(envFile).Variables()

    // Verify no cache collision
    assert.Equal(t, "valueA", varsA[0].Value)
    assert.Equal(t, "valueB", varsB[0].Value)
}
```

## Trade-offs

### What We Gain
- ✅ **Correctness**: Each client gets their own env var values
- ✅ **Security**: Cache isolation prevents cross-client leaks
- ✅ **Predictability**: Deterministic behavior

### What We Lose
- ⚠️ **Cache efficiency**: Clients with different env vars can't share module caches
- ⚠️ **Storage**: More cache entries (N clients = up to N× cache for modules using host env vars)

**Magnitude**: Only affects modules that use `$VAR` expansion in `.env` files. Modules with static values are unaffected.

## Implementation Checklist

- [ ] Add `CachePerClient` to `variables` field resolver
- [ ] Add `CachePerClient` to `get` field resolver
- [ ] Decide on ModuleSource digest approach (Option A or B)
- [ ] Implement chosen approach in `modulesource.go`
- [ ] Remove FIXME comments in `envfile.go`
- [ ] Add comprehensive multi-client tests
- [ ] Verify no performance regression (benchmark cache key computation)
- [ ] Update CHANGELOG.md with behavior change

## References

- **Cache key patterns**: `dagql/cachekey.go`
- **Existing CachePerClient usage**: `core/schema/host.go`, `core/schema/git.go`, `core/schema/module.go`
- **EnvFile implementation**: `core/envfile.go`
- **dotenv expansion logic**: `core/dotenv/` package
