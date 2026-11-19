---
title: "valetkey"
description: "Saved API requests your AI agents can run without ever seeing the keys"
layout: hextra-home
type: docs
cascade:
  - type: docs
---

{{< hextra/hero-badge link="https://github.com/bsreeram08/valetkey/releases" >}}
  <span>Formerly gurl — now with an MCP server: valetkey mcp</span>
{{< /hextra/hero-badge >}}

<div class="hx:mt-6 hx:mb-6">
{{< hextra/hero-headline >}}
  Your agents run the request,&nbsp;<br class="hx:sm:block hx:hidden" />not the key
{{< /hextra/hero-headline >}}
</div>

<div class="hx:mb-8">
{{< hextra/hero-subtitle >}}
  Save API requests once and keep secrets in environments.&nbsp;<br class="hx:sm:block hx:hidden" />Agents run them by name, from the CLI or as MCP tools, and never see a credential.
{{< /hextra/hero-subtitle >}}
</div>

<div class="hx:mb-8 valetkey-hero-code">

```bash
brew tap bsreeram08/valetkey https://github.com/bsreeram08/valetkey && brew install valetkey
valetkey save "health" https://api.example.com/health && valetkey run "health"
```

</div>

<div class="hx:mb-12 hx:flex hx:flex-wrap hx:gap-3">
{{< hextra/hero-button text="Get started" link="docs/quickstart/" >}}
{{< hextra/hero-button text="For AI agents" link="ai/agent-mode/" style="background: transparent; color: inherit; border: 1px solid currentColor;" >}}
</div>

{{< hextra/feature-grid >}}
  {{< hextra/feature-card
    title="Named requests"
    subtitle="Save any request, or paste a curl command, and replay it by name. Inspect with show, generate code with codegen."
    link="docs/quickstart/"
  >}}
  {{< hextra/feature-card
    title="Environments & secrets"
    subtitle="`{{placeholders}}` filled at send time from environments. Secrets encrypted with AES-256-GCM and shown as `*****`."
    link="guides/environments/"
  >}}
  {{< hextra/feature-card
    title="Collections & runs"
    subtitle="Group requests, chain them with extracted values, feed CSV/JSON rows, and emit JUnit reports for CI."
    link="guides/collection-runner/"
  >}}
  {{< hextra/feature-card
    title="Diagnose failures"
    subtitle="Every run is recorded. `valetkey diag` shows the request as sent, variable origins, and what changed since the last success."
    link="guides/diagnosing/"
  >}}
  {{< hextra/feature-card
    title="Agent mode"
    subtitle="`--agent`: JSON-only output, no prompts, secrets masked everywhere. Agents start with `valetkey docs`."
    link="ai/agent-mode/"
  >}}
  {{< hextra/feature-card
    title="Code generation"
    subtitle="Turn any saved request into curl, Go, Python or JavaScript, with credentials masked by default."
    link="cli/codegen/"
  >}}
{{< /hextra/feature-grid >}}
