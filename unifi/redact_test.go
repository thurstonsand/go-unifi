package unifi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRedactSensitivePayload guards that secrets embedded in a request body are
// not leaked when the body is included in an error message
// (ubiquiti-community/terraform-provider-unifi#256).
func TestRedactSensitivePayload(t *testing.T) {
	const (
		wgKey     = "fake-wireguard-key-value"
		pass      = "fake-passphrase-value"
		ipsecPSK  = "fake-ipsec-psk-value"
		radiusSec = "fake-radius-secret-value"
		pw        = "fake-password-value"
	)
	body := []byte(`{
		"name": "wgadmin",
		"purpose": "vpn-server",
		"x_wireguard_private_key": "` + wgKey + `",
		"x_passphrase": "` + pass + `",
		"x_ipsec_pre_shared_key": "` + ipsecPSK + `",
		"vlan": 50,
		"nested": {"radius_secret": "` + radiusSec + `", "ok": "keep"},
		"list": [{"password": "` + pw + `"}]
	}`)

	out := redactSensitivePayload(body)

	for _, leak := range []string{wgKey, pass, ipsecPSK, radiusSec, pw} {
		if strings.Contains(out, leak) {
			t.Errorf("redacted payload still leaks %q: %s", leak, out)
		}
	}

	// Non-sensitive fields must survive.
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("redacted payload is not valid JSON: %v\n%s", err, out)
	}
	if m["name"] != "wgadmin" {
		t.Errorf("name was lost: %v", m["name"])
	}
	if m["x_wireguard_private_key"] != "REDACTED" {
		t.Errorf("private key not redacted: %v", m["x_wireguard_private_key"])
	}
	if nested, ok := m["nested"].(map[string]any); !ok || nested["ok"] != "keep" {
		t.Errorf("nested non-sensitive value lost: %v", m["nested"])
	}

	// Non-JSON body is omitted, not echoed.
	if got := redactSensitivePayload([]byte("not json")); strings.Contains(got, "not json") {
		t.Errorf("non-JSON body echoed: %q", got)
	}
}

// TestRequestErrorRedactsMACOverride drives a real non-2xx response through
// doRequest and proves the returned error never carries the WAN clone MAC. The
// error message embeds the request body, so a sensitive key missing from
// sensitivePayloadKeys leaks straight into provider diagnostics.
func TestRequestErrorRedactsMACOverride(t *testing.T) {
	const sentinelMAC = "de:ad:be:ef:00:99"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handleNewStyleSetup(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"meta":{"rc":"error","msg":"api.err.InvalidPayload"},"data":[]}`))
	}))
	defer srv.Close()

	c, err := New(context.Background(), &Config{BaseURL: srv.URL, APIKey: "test-key"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	network := &Network{
		Name:               Ptr("wan"),
		Purpose:            "wan",
		MACOverride:        sentinelMAC,
		MACOverrideEnabled: true,
	}
	err = c.doRequest(
		context.Background(),
		http.MethodPost,
		"s/default/rest/networkconf",
		network,
		nil,
	)
	if err == nil {
		t.Fatal("expected an error from the 400 response, got nil")
	}

	msg := err.Error()
	if strings.Contains(msg, sentinelMAC) {
		t.Errorf("error leaks mac_override: %s", msg)
	}
	if !strings.Contains(msg, `"mac_override":"REDACTED"`) {
		t.Errorf("mac_override was not redacted in the payload; got: %s", msg)
	}
	if !strings.Contains(msg, `"mac_override_enabled":true`) {
		t.Errorf("mac_override_enabled should survive redaction; got: %s", msg)
	}
	if !strings.Contains(msg, "api.err.InvalidPayload") {
		t.Errorf("controller message not surfaced; got: %s", msg)
	}
}
