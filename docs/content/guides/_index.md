---
title: "Guides"
description: "Task-by-task walkthroughs: environments, placeholders, chaining, collections, diagnosing failures, scripting."
weight: 2
---

Each guide covers one task end to end. Read them in order the first time; later, jump to the one you need.

| Guide | Read it when you want to… |
|-------|---------------------------|
| [Environments & secrets]({{< relref "/guides/environments" >}}) | keep hosts and credentials out of saved requests, switch dev/staging/prod |
| [Variables & placeholders]({{< relref "/guides/variables" >}}) | understand `{{placeholders}}`, typed and dynamic values, scope precedence, `--persist` |
| [Chaining & extraction]({{< relref "/guides/chaining" >}}) | pull a token or id out of one response and use it in the next request |
| [Collections & data-driven runs]({{< relref "/guides/collection-runner" >}}) | run a group of requests in order, feed CSV/JSON rows, produce CI reports |
| [Diagnosing failures]({{< relref "/guides/diagnosing" >}}) | find out why a run failed and what changed since it last worked |
| [Scripting & assertions]({{< relref "/guides/scripting" >}}) | check responses, or run JavaScript before and after a request |
| [Other protocols]({{< relref "/guides/protocols" >}}) | call GraphQL, gRPC, WebSocket or SSE endpoints |
