# Comparative Delivery Estimate: AI Agents vs Non-AI Teams

## Scope Used for Comparison

This comparison assumes the delivered scope included the following work:

- Stable single-user admin panel baseline for simple Docker Compose stacks
- Safe app lifecycle operations: create, deploy, stop, restart, logs, delete
- Read-only Scanner / Audit flow for discovered Docker resources
- Public Git repository import with Compose-first flow and Dockerfile fallback
- Managed environment variables and app configuration persistence
- App domain, proxy port, `Nginx`, and `Certbot` integration for the stable baseline
- Safe deletion preflight, shared-resource protection, and certificate cleanup rules
- Tests, rollout notes, stable/backlog planning, and production-oriented documentation

## Actual AI-Assisted Delivery

### Time Window

- Start: `2026-03-12 17:47`
- End: `2026-03-14 13:03`
- Wall-clock elapsed time: `43h 16m`
- Estimated active work: `~12 hours`
  - Based on the provided assumption of about `4 hours/day` across 3 calendar dates

### Direct Cost

- Estimated token cost: `~$50`

### Delivery Model

- Human-directed AI execution
- Rapid iteration with low coordination overhead
- Immediate code, tests, refactors, docs, and backlog updates in one loop

## Comparison Table

| Scenario | Typical team shape | Expected organization model | Estimated calendar time | Estimated active effort | Estimated direct cost |
|---|---|---|---:|---:|---:|
| AI-assisted delivery | 1 human operator + AI agents | Continuous direct iteration, no formal cross-team handoffs | ~2 days elapsed | ~12 active hours | ~$50 in tokens |
| Silicon Valley-style team (no AI) | PM/Product, EM or Tech Lead, 1 Senior Backend, 1 Mid Backend, 1 Frontend, shared QA, shared DevOps/SRE | Discovery, planning, architecture review, sprint implementation, QA pass, release review | ~4-7 weeks | ~18-28 person-weeks | ~$140k-$240k |
| Small funded startup (no AI) | 1 Senior + 2 Mid engineers, founder/product acting as PM, part-time manual QA/ops | Short planning, direct build, lightweight review, manual QA, one release cycle | ~3-6 weeks | ~10-18 person-weeks | ~$30k-$60k |

## Realistic Non-AI Execution

### 1. Silicon Valley-style Team

This is how the work would likely be organized in a more formal product-engineering environment:

| Area | Likely setup |
|---|---|
| Product definition | PM or product owner writes scope, acceptance criteria, and release boundaries |
| Architecture | Senior/Lead engineer reviews lifecycle, routing, storage, and cleanup safety |
| Backend work | 2 backend engineers split lifecycle/import/scanner/routing/TLS work |
| Frontend/UI | 1 frontend engineer handles settings/forms/pages/feedback states |
| QA | Shared QA runs regression, API checks, and release checklist |
| DevOps/SRE | Shared ops engineer reviews deployment, rollback, host dependencies, and runtime safety |
| Sign-off | Architecture review, PR review, QA sign-off, release approval |

#### Likely Timeline

| Phase | Typical duration |
|---|---:|
| Scope clarification and planning | 2-4 business days |
| Architecture and risk review | 2-5 business days |
| Implementation | 2-4 weeks |
| QA, bug fixing, and release hardening | 1-2 weeks |
| Total | ~4-7 weeks |

#### Likely Budget Logic

This estimate assumes fully loaded internal cost, not just salary:

- PM/Product: partial allocation
- EM/Lead: partial allocation
- Senior Backend: full allocation
- Mid Backend: full allocation
- Frontend: partial to medium allocation
- QA: partial allocation
- DevOps/SRE: partial allocation

Expected realistic range:

- `~$140k-$240k`

This is the most process-heavy path, but also the safest for governance, release approval, and cross-functional visibility.

### 2. Small Startup Team Without AI

This is a more realistic lean execution model for a private-invested startup:

| Area | Likely setup |
|---|---|
| Product and prioritization | Founder or senior engineer defines scope directly |
| Architecture | Senior engineer owns technical direction |
| Implementation | 2 mid-level engineers + 1 senior engineer ship the backend/UI changes |
| QA | Mostly manual testing by engineers and founder |
| Release process | One staging pass, one production cutover, limited formal approvals |

#### Likely Timeline

| Phase | Typical duration |
|---|---:|
| Scope alignment and technical decomposition | 1-3 business days |
| Implementation | 2-4 weeks |
| Stabilization, manual QA, fixes | 1-2 weeks |
| Total | ~3-6 weeks |

#### Likely Budget Logic

This estimate assumes a lean internal burn model:

- 1 senior engineer full-time
- 2 mid-level engineers full-time
- founder/product time not fully priced as a dedicated PM
- manual QA and ops mostly absorbed by the engineering team

Expected realistic range:

- `~$30k-$60k`

This path is much cheaper than a large formal team, but it carries more execution risk, more key-person dependency, and less process safety.

## Interpretation

### Delivery Compression

The AI-assisted result compressed a project that would normally take:

- `~4-7 weeks` in a formal, well-staffed non-AI environment
- `~3-6 weeks` in a lean startup environment

into:

- `~43 hours of elapsed time`
- `~12 active work hours`
- `~$50` of direct token cost

### Important Caveat

This does **not** mean AI fully replaces a real team in accountability, production ownership, or long-term maintenance. It means that:

- AI can dramatically reduce implementation and documentation cost
- AI can compress iteration cycles when scope is actively directed by a human operator
- human supervision is still required for final architectural judgment, deployment safety, and production validation

## Bottom Line

For this specific scope, a realistic non-AI team would probably have delivered the same result:

- in `weeks`, not `days`
- with `multiple people`, not one human + AI agents
- and at a budget measured in `tens of thousands` or `hundreds of thousands of dollars`, not `~$50`

That makes this project a strong example of how an autonomous admin panel product can be built and iterated much faster with AI-assisted execution, while still following a reasonably structured layered architecture.
