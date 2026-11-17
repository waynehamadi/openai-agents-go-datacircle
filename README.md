# Datacircle tools for the OpenAI Agents Go SDK

Find the best GTM APIs. We benchmark all the GTM APIs and we put the results on our website: https://datacircle.dev

This Go module gives the [OpenAI Agents Go SDK](https://github.com/nlpodyssey/openai-agents-go) four function tools on Datacircle's open API, https://api.datacircle.dev/v1. You need no key and no account.

| Tool | What the tool gives | Call |
| --- | --- | --- |
| `datacircle_list_apis` | Every GTM API on Datacircle, with the API's slug, number of plans, number of providers and best plan | `GET /apis` |
| `datacircle_best_plans` | One API's plans, sorted by score out of 100, best first. Each plan comes with a rank and a price per 1,000 calls | `GET /apis/{slug}/best` |
| `datacircle_search_plans` | Plans that match a provider or plan name, on all APIs or on one API. Sort by score, provider or price | `GET /plans` |
| `datacircle_provider` | The APIs a provider sells, and the provider's rank on each API | `GET /providers/{slug}` |

## Install

```bash
go get github.com/waynehamadi/openai-agents-go-datacircle
```

It works with the SDK's v0.1.0 and its later versions.

## Use the tools

```go
import (
	"github.com/nlpodyssey/openai-agents-go/agents"
	datacircle "github.com/waynehamadi/openai-agents-go-datacircle"
)

agent := agents.New("GTM assistant").
	WithInstructions("Answer from the datacircle tools.").
	WithModel("gpt-4o-mini").
	WithTools(datacircle.NewTools(nil)...)

result, err := agents.Run(ctx, agent, "Who is the best provider for the email-finder API?")
fmt.Println(result.FinalOutput)
```

Each tool has its own constructor: `NewListAPIsTool`, `NewBestPlansTool`, `NewSearchPlansTool` and `NewProviderTool`. Give a model only the ones it needs. Each one is an `agents.FunctionTool`, so you can also call it yourself:

```go
answer, err := datacircle.NewBestPlansTool(nil).OnInvokeTool(ctx, `{"slug":"email-finder","limit":3}`)
```

The tools' schemas are strict, as the SDK makes them: the model sends every input, an empty string or 0 for one it leaves out.

Each tool gives back a JSON object. Each plan comes with its rank, provider, plan, score, price per 1,000 (`perThousand`, in dollars) and its page on datacircle.dev. That page lists every field. When the API gives no data, like for a slug it doesn't know, the tool's result is `{"error": "..."}` with the API's own words, so the model can call again. A network failure is the tool's error: the SDK gives the model its text and the run goes on.

`datacircle.Config` sets the `HTTPClient` the tools use (a timeout, a proxy). `nil` uses `http.DefaultClient`.

## Network

Each tool call makes one HTTPS GET request to https://api.datacircle.dev/v1 and nothing else. The request holds only the tool's own arguments: no key, no config, nothing from your chats. The API takes up to 60 calls a minute from one IP address. The tool waits once, for a minute at most, when the API asks the tool to wait. The module depends only on the SDK and `golang.org/x/text`, and stores nothing.

## Links

- Datacircle: https://datacircle.dev
- The API: https://datacircle.dev/api
- The API's spec: https://api.datacircle.dev/v1/openapi.json
- What each key means: https://datacircle.dev/fields

MIT license.
