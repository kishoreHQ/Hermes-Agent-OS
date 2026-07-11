# Runtime plugins

Runtimes **execute work**. They are not model providers.

Contract: `kernel/pkg/runtime.Runtime`  
Context: `ContextEnvelope` (prompt is one field among many).

Example layout (future):

```
runtimes/
  process-pty/
    plugin.yaml
  sandbox-agent-host/
    plugin.yaml
  echo/                 # test double
    plugin.yaml
```

Sandbox tier is part of the descriptor (`micro-vm` | `container` | `process-pty`).
