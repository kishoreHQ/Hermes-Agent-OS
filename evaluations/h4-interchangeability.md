# H4 Interchangeability Evaluation

**Gate:** `make prove-h4`  
**ADR:** [docs/adr/0009-interchangeability.md](../docs/adr/0009-interchangeability.md)

## Matrix

| Case | Labels | Expected provider | Expected runtime |
|------|--------|-------------------|------------------|
| default | — | `provider.example.echo` | `runtime.example.echo` |
| prefer steps | `route.preferRuntime=runtime.example.steps` | echo | steps |
| exclude local | `route.excludeProvider=provider.example.echo`, `route.preferLocal=false` | budget | echo |
| budget+steps | exclude echo provider + prefer steps | budget | steps |

## Pass criteria

1. All four cases `state=succeeded`  
2. Each has `route.decided` with `required` + `reason`  
3. No kernel source change between cases  
4. ≥2 providers and ≥2 runtimes registered  

## Reviewer sign-off

| Persona | Verdict |
|---------|---------|
| Principal Architect | PASS if matrix green |
| Platform | PASS if plugins only |
| Protocol | PASS if capabilities primary |
