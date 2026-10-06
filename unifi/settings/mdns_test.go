package settings

import (
	"encoding/json"
	"testing"
)

// TestMdnsRoundTrip checks the site-level mdns setting (un)marshals correctly,
// using a payload shaped like a real UniFi Network 10.5.67 response in Custom
// mode. enabled_for/enabled_for_network_ids are absent from the 10.5.x field
// spec and are re-injected by the generator, so pin them here.
func TestMdnsRoundTrip(t *testing.T) {
	raw := `{
		"_id": "6600000000000000000000a1",
		"site_id": "6600000000000000000000b2",
		"key": "mdns",
		"mode": "custom",
		"enabled_for": "some",
		"enabled_for_network_ids": [
			"6600000000000000000000c1",
			"6600000000000000000000c2",
			"6600000000000000000000c3"
		],
		"predefined_services": [
			{"code": "apple_airPlay"},
			{"code": "homeKit"}
		],
		"custom_services": []
	}`

	var s Mdns
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s.GetKey() != "mdns" {
		t.Errorf("GetKey() = %q, want mdns", s.GetKey())
	}
	if s.Mode != "custom" {
		t.Errorf("Mode = %q, want custom", s.Mode)
	}
	if s.EnabledFor != "some" {
		t.Errorf("EnabledFor = %q, want some", s.EnabledFor)
	}
	if len(s.EnabledForNetworkIDs) != 3 {
		t.Errorf("EnabledForNetworkIDs = %v, want 3 entries", s.EnabledForNetworkIDs)
	}
	if len(s.PredefinedServices) != 2 ||
		s.PredefinedServices[0].Code != "apple_airPlay" ||
		s.PredefinedServices[1].Code != "homeKit" {
		t.Errorf("PredefinedServices = %v", s.PredefinedServices)
	}
	if len(s.CustomServices) != 0 {
		t.Errorf("CustomServices = %v, want empty", s.CustomServices)
	}

	if k, err := GetSettingKey(&s); err != nil || k != "mdns" {
		t.Errorf("GetSettingKey = (%q, %v), want (mdns, nil)", k, err)
	}

	b, err := json.Marshal(&s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back map[string]any
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if back["key"] != "mdns" || back["mode"] != "custom" || back["enabled_for"] != "some" {
		t.Errorf(
			"round-trip lost fields: key=%v mode=%v enabled_for=%v",
			back["key"], back["mode"], back["enabled_for"],
		)
	}
	ids, ok := back["enabled_for_network_ids"].([]any)
	if !ok || len(ids) != 3 {
		t.Errorf("enabled_for_network_ids = %v", back["enabled_for_network_ids"])
	}
}

// TestMdnsEmptyServiceListsSerialize pins that explicitly empty service lists
// reach the controller as []. With the default omitempty codegen they would be
// dropped from the body and the controller would keep the previous services,
// making "remove every service" a silent no-op.
func TestMdnsEmptyServiceListsSerialize(t *testing.T) {
	s := Mdns{
		Mode:                 "custom",
		EnabledFor:           "all",
		EnabledForNetworkIDs: []string{},
		PredefinedServices:   []SettingMdnsPredefinedServices{},
		CustomServices:       []SettingMdnsCustomServices{},
	}
	s.SetKey("mdns")

	b, err := json.Marshal(&s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back map[string]any
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	for _, key := range []string{
		"predefined_services",
		"custom_services",
		"enabled_for_network_ids",
	} {
		val, present := back[key]
		if !present {
			t.Errorf("%s missing from body, want []", key)
			continue
		}
		list, ok := val.([]any)
		if !ok || len(list) != 0 {
			t.Errorf("%s = %v, want []", key, val)
		}
	}
}
