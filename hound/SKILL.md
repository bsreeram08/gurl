---
name: hound
description: Brutal code review skill that spawns subagents for security, architecture, and code quality checks.
---

# Hound Code Review Skill

**Bloodhound for Bugs and Issues**

This skill triggers brutal, thorough code review via subagent.

## When to Use

Use when:
- User says "review this code"
- User says "swarm review"
- User says "lint test"
- Before any code merge
- After any major refactor

## ALWAYS SPAWN A SUBAGENT

When this skill is activated, **ALWAYS spawn a worker subagent**.

```
Agent: Hound
Task: Code Review
Scope: [files from request]
```

## Subagent Prompt Template

```
## Hound Code Review

**Project:** [project path]
**Files to Review:** [files or "all"]

### Security Audit (CRITICAL)
- Shell injection (exec.Command string concat)
- Command injection (user input in system calls)
- Path traversal (unvalidated file paths)
- Race conditions (concurrent map access)
- Secrets exposure (API keys in code)

### Architecture Review (CRITICAL)
- Randomized control flow (map iteration)
- Non-atomic operations (multi-step writes)
- Global mutable state
- Memory leaks
- Deadlocks

### Reliability Check (HIGH)
- Error swallowing (_ = err)
- Missing timeouts
- Resource exhaustion

### Output Format
```
| Aspect | Status |
|--------|--------|
| Security | 🟢/🟡/🔴 |
| Architecture | 🟢/🟡/🔴 |
| Code Quality | 🟢/🟡/🔴 |

### 🔴 CRITICAL ISSUES
[Issue, file, evidence, fix]

### 🟡 WARNINGS
[Issue, severity, description]

### Hound's Verdict
[REJECT / REQUEST CHANGES / MERGE]
```
