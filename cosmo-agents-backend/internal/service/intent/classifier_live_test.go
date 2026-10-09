//go:build live

package intent

// Measures a classifier prompt change against the previous prompt on a real
// model. Not part of the normal suite: it calls the API and costs credit.
//
//   OPENAI_API_KEY=... OPENAI_BASE_URL=... OPENAI_MODEL=... \
//   LIVE_DATASET=../../../../eval/judge_eval_v2/dataset.json \
//   go test -tags live -run TestLive_ ./internal/service/intent/ -v

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/openai/openai-go"
	"github.com/rs/zerolog"

	"github.com/rockship/cosmo-agents-go/pkg/ai"
)

// previousPrompt is the prompt without the opt-out and wrong-recipient rules.
func previousPrompt(t *testing.T, current string) string {
	start := strings.Index(current, "   - Do Not Contact vs Not Interested:")
	end := strings.Index(current, "\n6. Analysis Process:")
	if start < 0 || end < start {
		t.Fatal("cannot locate the added rules in the prompt")
	}
	return current[:start] + "\n" + current[end:]
}

func TestLive_WrongRecipientAndOptOutRules(t *testing.T) {
	path := os.Getenv("LIVE_DATASET")
	if path == "" {
		t.Skip("LIVE_DATASET not set")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var ds struct {
		Items []struct {
			ID    string `json:"id"`
			Body  string `json:"body"`
			Truth string `json:"ground_truth_intent"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &ds); err != nil {
		t.Fatal(err)
	}
	seeds := strings.Split(os.Getenv("LIVE_SEEDS"), ",")
	if os.Getenv("LIVE_SEEDS") == "" {
		seeds = []string{"e9", "e5", "e7", "e22", "e2", "e12", "e13"}
	}
	keep := map[string]bool{}
	for _, s := range seeds {
		keep[s] = true
	}

	client := openai.NewClient(ai.CompatOptions(os.Getenv("OPENAI_API_KEY"))...)
	log := zerolog.Nop()
	ic := NewIntentClassifier(&client, os.Getenv("OPENAI_MODEL"), &log)
	after := ic.constructPrompt()
	before := previousPrompt(t, after)

	type row struct{ id, seed, truth, before, after string }
	var rows []row
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for _, it := range ds.Items {
		seed := strings.SplitN(it.ID, "v", 2)[0]
		if !keep[seed] {
			continue
		}
		truth := it.Truth
		if truth == "OTHER" {
			truth = "UNKNOWN_INTENT"
		}
		wg.Add(1)
		go func(id, seed, body, truth string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			label := func(p string) string {
				r, err := ic.classifyWithPrompt(context.Background(), p, body)
				if err != nil {
					return "ERROR"
				}
				return canonicalLabel(r.Intent)
			}
			b, a := label(before), label(after)
			mu.Lock()
			rows = append(rows, row{id, seed, truth, b, a})
			mu.Unlock()
		}(it.ID, seed, it.Body, truth)
	}
	wg.Wait()

	sort.Slice(rows, func(i, j int) bool { return rows[i].id < rows[j].id })
	type tally struct{ n, b, a int }
	per := map[string]*tally{}
	for _, r := range rows {
		if per[r.seed] == nil {
			per[r.seed] = &tally{}
		}
		p := per[r.seed]
		p.n++
		if r.before == r.truth {
			p.b++
		}
		if r.after == r.truth {
			p.a++
		}
		mark := ""
		if r.before != r.after {
			mark = "  <- changed"
		}
		fmt.Printf("%-7s %-16s before=%-16s after=%-16s%s\n", r.id, r.truth, r.before, r.after, mark)
	}
	fmt.Println("\nseed     n  before  after")
	var tb, ta, tn int
	for _, s := range seeds {
		if p := per[s]; p != nil {
			fmt.Printf("%-6s %3d  %3d     %3d\n", s, p.n, p.b, p.a)
			tn, tb, ta = tn+p.n, tb+p.b, ta+p.a
		}
	}
	fmt.Printf("total  %3d  %3d     %3d\n", tn, tb, ta)
}
