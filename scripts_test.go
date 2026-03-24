package sprout_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestRunBrowserLLMScriptFailsClearlyWithoutKey(t *testing.T) {
	cmd := exec.Command("bash", "scripts/run_browser_llm.sh", ":8080")
	cmd.Env = append(cmd.Env, "SPROUT_DRY_RUN=1")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected missing-key invocation to fail")
	}
	text := string(output)
	if !strings.Contains(text, "OPENAI_API_KEY is required for browser LLM mode.") {
		t.Fatalf("missing clear key error: %s", text)
	}
}

func TestRunBrowserLLMScriptDryRunPrintsLaunchCommand(t *testing.T) {
	cmd := exec.Command("bash", "scripts/run_browser_llm.sh", ":9090")
	cmd.Env = append(cmd.Env,
		"SPROUT_DRY_RUN=1",
		"OPENAI_API_KEY=test-key",
		"OPENAI_BASE_URL=https://api.openai.com",
		"SPROUT_LLM_MODEL=gpt-4.1-mini",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dry run failed: %v output=%s", err, string(output))
	}
	text := string(output)
	for _, fragment := range []string{
		"Starting sprout-web in LLM mode on :9090 with model gpt-4.1-mini",
		"Provider base URL: https://api.openai.com",
		"go run ./cmd/sprout-web --addr :9090 --model gpt-4.1-mini",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("output missing %q: %s", fragment, text)
		}
	}
}
