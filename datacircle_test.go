package datacircle

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlpodyssey/openai-agents-go/agents"
)

func TestDeclarations(t *testing.T) {
	names := []string{ListAPIsName, BestPlansName, SearchPlansName, ProviderName}
	one := []agents.Tool{NewListAPIsTool(nil), NewBestPlansTool(nil), NewSearchPlansTool(nil), NewProviderTool(nil)}
	for _, tools := range [][]agents.Tool{NewTools(nil), one} {
		if len(tools) != 4 {
			t.Fatalf("%d tools, want 4", len(tools))
		}
		for i, tl := range tools {
			ft, ok := tl.(agents.FunctionTool)
			if !ok {
				t.Fatalf("tool %d is a %T, want an agents.FunctionTool", i, tl)
			}
			if ft.Name != names[i] || ft.ToolName() != names[i] {
				t.Errorf("tool %d is %s, want %s", i, ft.Name, names[i])
			}
			if ft.Description == "" || !strings.Contains(ft.Description, "https://datacircle.dev") {
				t.Errorf("%s's description, which the model reads, is %q", ft.Name, ft.Description)
			}
			if !ft.StrictJSONSchema.Or(false) {
				t.Errorf("%s's schema is not strict", ft.Name)
			}
			schema, _ := json.Marshal(ft.ParamsJSONSchema)
			if !strings.Contains(string(schema), `"additionalProperties":false`) {
				t.Errorf("%s's schema %s allows other properties", ft.Name, schema)
			}
			props, _ := ft.ParamsJSONSchema["properties"].(map[string]any)
			required, _ := ft.ParamsJSONSchema["required"].([]any)
			if len(props) == 0 || len(required) != len(props) {
				t.Errorf("%s's schema %s: strict needs every property required", ft.Name, schema)
			}
			for name, p := range props {
				if d, _ := p.(map[string]any)["description"].(string); d == "" {
					t.Errorf("%s's %s has no description", ft.Name, name)
				}
			}
		}
	}
	schema, _ := json.Marshal(NewSearchPlansTool(nil).ParamsJSONSchema)
	if !strings.Contains(string(schema), `"sort":{"description":"score (best first, the default), provider (A to Z) or price ($ per 1,000, lowest first). Empty for score","type":"string"}`) {
		t.Errorf("datacircle_search_plans' schema %s lost its sort's description", schema)
	}
}

func TestCalls(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		switch {
		case r.URL.Path == "/apis":
			w.Write([]byte(`[{"api":"Employee profile","slug":"linkedin-profile"},{"api":"Employee email","slug":"email-finder","plans":3}]`))
		case r.URL.Path == "/apis/email-finder/best":
			w.Write([]byte(`{"api":"Employee email","plans":[{"provider":"A","rank":1,"terms":"long"}]}`))
		case r.URL.Path == "/plans" && (r.URL.Query().Get("api") == "Bright Data" || r.URL.Query().Get("api") == "bright-data"):
			w.WriteHeader(400)
			w.Write([]byte(`{"error":"api must be one of: Employee email"}`))
		case r.URL.Path == "/plans" && strings.Contains(r.URL.Query().Get("query"), "-"):
			w.Write([]byte(`{"total":0,"plans":[]}`))
		case r.URL.Path == "/plans":
			w.Write([]byte(`{"total":1,"plans":[{"provider":"Bright Data","rank":2}]}`))
		case r.URL.Path == "/providers/bright-data":
			w.Write([]byte(`{"provider":"Bright Data","apis":[{"api":"Employee profile","plans":[{"plan":"C"}]},{"api":"Employee email","rank":4,"plans":[{"plan":"B","rank":9},{"plan":"A","rank":4}]}]}`))
		default:
			w.WriteHeader(404)
			w.Write([]byte(`{"error":"no API has the slug nope"}`))
		}
	}))
	defer server.Close()
	cfg := &Config{BaseURL: server.URL}
	ctx := context.Background()
	for _, call := range []struct {
		tool agents.FunctionTool
		args string
		has  string
	}{
		{NewListAPIsTool(cfg), `{"query":"email","limit":0}`, `"total":1`},
		{NewBestPlansTool(cfg), `{"slug":"Email Finder","limit":2}`, `"provider":"A"`},
		{NewSearchPlansTool(cfg), `{"api":"Bright Data","query":"","sort":"","limit":0,"offset":0}`, `No API is named`},
		{NewSearchPlansTool(cfg), `{"api":"bright-data","query":"","sort":"","limit":0,"offset":0}`, `these are the plans whose name holds \"bright data\".","plans":[{"provider":"Bright Data"`},
		{NewSearchPlansTool(cfg), `{"api":"","query":"bright-data","sort":"price","limit":0,"offset":0}`, `"plans":[{"provider":"Bright Data"`},
		{NewProviderTool(cfg), `{"slug":"Bright Data"}`, `"apis":[{"api":"Employee email","best":{"plan":"A"`},
		{NewProviderTool(cfg), `{"slug":"bright-data"}`, `"ranked":1,"total":2`},
		{NewBestPlansTool(cfg), `{"slug":"nope","limit":0}`, `{"error":"no API has the slug nope. datacircle_list_apis gives every API's slug. A provider's plans: datacircle_search_plans`},
		{NewBestPlansTool(cfg), `{"slug":"","limit":2}`, `"error":"Give an API's slug, like email-finder: datacircle_list_apis gives every slug. A provider's plans:`},
		{NewListAPIsTool(cfg), `{"query":"Bright Data","limit":0}`, `"note":"No API's name holds every word of \"Bright Data\". A provider's plans: datacircle_search_plans`},
	} {
		got, err := call.tool.OnInvokeTool(ctx, call.args)
		out, _ := json.Marshal(got)
		if err != nil || !strings.Contains(string(out), call.has) {
			t.Errorf("%s(%s) = %s, %v; want it to hold %s", call.tool.Name, call.args, out, err, call.has)
		}
		if strings.Contains(string(out), "terms") {
			t.Errorf("%s gave a plan's every key: %s", call.tool.Name, out)
		}
	}
	if got, _ := NewListAPIsTool(cfg).OnInvokeTool(ctx, `{"query":"employee email","limit":0}`); got.(map[string]any)["note"] != nil {
		t.Errorf("datacircle_list_apis notes a query an API's name holds every word of: %v", got)
	}
	if want := "/apis/email-finder/best?limit=2"; paths[1] != want {
		t.Errorf("called %s, want %s", paths[1], want)
	}
	if want := "/plans?query=bright+data&sort=price"; paths[8] != want {
		t.Errorf("called %s, want %s", paths[8], want)
	}
}

func TestDown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer server.Close()
	_, err := NewListAPIsTool(&Config{BaseURL: server.URL}).OnInvokeTool(context.Background(), `{"query":"","limit":0}`)
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Errorf("an API that is down gave %v, want the tool's error", err)
	}
}
