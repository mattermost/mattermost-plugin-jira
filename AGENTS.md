# Agents

## Build commands

- `make apply` — Propagate plugin manifest into server/ and webapp/
- `MM_DEBUG=true MM_SERVICESETTINGS_ENABLEDEVELOPER=1 make dist` — Build plugin bundle (single platform, faster for local/cloud-agent dev)
- `make check-style` — Run all linters (Go + webapp)
- `make test` — Run all tests (Go + webapp)
- `make dist` — Build all platform assets
