# PERSONA_ROUTER — Who Should Act Now?

Read this to decide which persona to activate.

---

## Decision Tree

```
What do I need to do?
│
├── Research competitors / UX patterns / discover features
│   └── → researcher (NO-CODE; hands off to pm/feature)
│
├── Define or clarify requirements / scope / acceptance criteria
│   └── → pm
│
├── Design module structure, API contracts, or ADR
│   └── → architect
│
├── Design a specific feature (UI + state + hooks)
│   └── → feature (then hands off to developer)
│
├── Write or change any code / config / file
│   └── → developer  ← ONLY this persona writes code
│
├── Review code that developer wrote
│   └── → dev_supervisor
│
├── Review auth / secrets / crypto / security boundaries
│   └── → security
│
├── Run tests / build / check pipeline
│   └── → qa
│
├── Approve test evidence and close a task
│   └── → qa_supervisor
│
├── Write or update documentation
│   └── → docs
│
└── CI/CD pipeline / versioning / release
    └── → release
```

---

## Handoff Protocol

When a NO-CODE persona finishes their work, they must:

1. Write a clear **Handoff block** at the bottom of their output:
   ```
   ## Handoff → developer
   - File to create/edit: ...
   - Exact change: ...
   - Acceptance criteria: ...
   ```

2. Set the workflow task to `READY` if it was in `BACKLOG`.

3. The `developer` picks up the handoff and implements.

4. After implementation, `developer` hands off to `dev_supervisor` for review.

---

## Voltana-Specific Persona Assignments

| Area | Primary Persona | Reviewer |
|---|---|---|
| Competitive / UX research, feature discovery | researcher | pm (then feature) |
| Go API (handler/service/repo) | developer | dev_supervisor |
| Database migrations | developer (after architect designs) | dev_supervisor |
| React feature components | developer (after feature designs) | dev_supervisor |
| JWT / auth flow | developer + security review | security → dev_supervisor |
| Docker Compose / Nginx | developer (after architect designs) | dev_supervisor |
| Battery health algorithm | architect → developer | dev_supervisor + qa |
| OBD integration | architect + security → developer | security + qa_supervisor |
| API documentation | docs | dev_supervisor |
| CI/CD pipeline | release → developer | dev_supervisor |
