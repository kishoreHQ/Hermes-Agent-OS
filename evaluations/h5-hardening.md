# H5 Production Hardening Evaluation

**Gate:** `make prove-h5`  
**ADRs:** 0010 (monorepo), 0011 (security)

## Composite checks

1. **H4 matrix** — 2×2 provider×runtime interchangeability  
2. **Evaluation suite** — happy path, observe, assist+external, steps runtime  
3. **Performance** — mission path p50/p99 baselines (soft pathological fail)  
4. **Security invariants** — observe no-exec, assist approval, sandbox ranking  

## Sign-off

| Persona | Focus | Verdict target |
|---------|--------|----------------|
| Security Engineer | modes, scopes, signing, credentials | PASS |
| Performance Engineer | baselines | PASS (soft CI) |
| Platform Engineer | monorepo retention | PASS (ADR-0010) |
| Principal Architect | no vendor lock-in regression | PASS |
