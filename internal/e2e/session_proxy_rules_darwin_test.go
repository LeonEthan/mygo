//go:build darwin && !ios

package e2e

import (
	"testing"

	"github.com/egoist/mygo"
)

func TestProxiedSessionRequiresCompiledRules(t *testing.T) {
	config := mygo.SessionProxyConfig{HTTPProxy: "http://127.0.0.1:12345", ContentRulesJSON: "[]"}
	if session, err := mygo.NewProxiedSession("rule-validation", config); err == nil || session != nil {
		t.Fatalf("invalid WebKit rules produced session %v, error %v", session, err)
	}
	config.ContentRulesJSON = `[{"trigger":{"url-filter":"private"},"action":{"type":"block"}}]`
	if _, err := mygo.NewProxiedSession("rule-validation", config); err != nil {
		t.Fatal(err)
	}
}
