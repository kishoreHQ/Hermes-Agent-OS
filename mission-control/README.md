# Mission Control (product home)

Operator UI host for Hermes Agent OS.

**Status:** Product home reserved.  
Shipped Mission Control currently lives in AESP-Reference-Implementation (`ui/`) until Phase H3 re-home (see [docs/PLAN.md](../docs/PLAN.md)).

Rules when ported here:

- Bind only to Hermes Host API (`/api/v1` + events)  
- Zero vendor SDKs in the UI  
- Preserve UI-SPEC / Command Deck contracts from the RI track  
