# Hound Code Review Skill

**ALWAYS SPAWN A SUBAGENT WHEN THIS SKILL IS CALLED.**

This skill triggers a brutal, thorough code review via subagent. Named Hound - bloodhound for bugs, relentless tracker of issues.

## When This Skill Is Activated

When user says:
- "review this code"
- "swarm review"  
- "lint test this"
- "code review"
- "check my code"
- "find issues"
- ANY request to review code

## Subagent Invocation

**ALWAYS** spawn a worker subagent with this exact prompt template:

```
## Hound Code Review Task

**Project:** [Get project path from context]
**Files to Review:** [Determine from user's request or diff]

### Your Task
Conduct a brutal, thorough code review following Hound methodology:

1. **Security Audit** (CRITICAL)
   - Shell injection (exec.Command string concat)
   - Command injection (user input in system calls)
   - Path traversal (unvalidated file paths)
   - Race conditions (concurrent map access)
   - Secrets exposure (API keys in code)

2. **Architecture Review** (CRITICAL)
   - Randomized control flow (map iteration)
   - Non-atomic operations (multi-step writes)
   - Global mutable state
   - Memory leaks
   - Deadlocks

3. **Reliability Check** (HIGH)
   - Error swallowing (_ = err)
   - Missing timeouts
   - Resource exhaustion

4. **Correctness** (HIGH)
   - Off-by-one errors
   - Integer overflow
   - Time zone bugs

### Output Format
Return a structured review:

```
## Hound Code Review

| Aspect | Status |
|--------|--------|
| Security | 🟢/🟡/🔴 |
| Architecture | 🟢/🟡/🔴 |
| Code Quality | 🟢/🟡/🔴 |
| Blast Radius | Low/Medium/High |
| Semantic Changes | Yes/No |

### 🔴 CRITICAL ISSUES
[Each issue with: severity, type, file, evidence, fix]

### 🟡 WARNINGS
[Each warning with: severity, type, description]

### 🟢 NOTES
[Minor improvements]

---

## Hound's Verdict
[REJECT / REQUEST CHANGES / MERGE]
```

### Files to Review
- If PR: Review changed files only
- If --all: Review all .go files
- If specific: Review listed files

### Subagent Type
Always use: `worker`

### Description
"Hound Review - [brief scope]"

## Checklist (For Subagent)

- [ ] Spawn as subagent (NEVER skip)
- [ ] Review ALL files in scope
- [ ] Check security first (critical)
- [ ] Check architecture second (critical)
- [ ] Check reliability third
- [ ] Check correctness
- [ ] Output in Hound format
- [ ] Provide specific fixes
- [ ] Give verdict (REJECT/REQUEST CHANGES/MERGE)

