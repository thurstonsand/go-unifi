package unifi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateDeviceNeverWritesOutletState(t *testing.T) {
	const (
		site = "default"
		mac  = "00:11:22:33:44:55"
		id   = "000000000000000000000001"
	)

	var updateBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handleNewStyleSetup(w, r) {
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == loginPathNew {
			w.Header().Set("X-Csrf-Token", "tok")
			w.WriteHeader(http.StatusOK)
			return
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/proxy/network/api/s/default/stat/device/"+mac:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"meta":{"rc":"ok"},"data":[{"_id":"` + id + `","mac":"` + mac + `","name":"old","outlet_enabled":true,"outlet_overrides":[{"index":1,"relay_state":true,"cycle_enabled":false}]}]}`))
		case r.Method == http.MethodPut && r.URL.Path == "/proxy/network/api/s/default/rest/device/"+id:
			if err := json.NewDecoder(r.Body).Decode(&updateBody); err != nil {
				t.Errorf("decoding update body: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"meta":{"rc":"ok"},"data":[{"_id":"` + id + `","mac":"` + mac + `","name":"new"}]}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	client, err := New(context.Background(), &Config{
		BaseURL:  srv.URL,
		Username: "admin",
		Password: "admin",
	})
	if err != nil {
		t.Fatalf("client init: %v", err)
	}

	idx := int64(1)
	acRecoverySeconds := int64(60)
	internetLossSeconds := int64(5)
	_, err = client.UpdateDevice(context.Background(), site, &Device{
		ID:                                    id,
		MAC:                                   mac,
		Name:                                  "new",
		OutletEnabled:                         false,
		OutletPowerCycleEnabled:               true,
		OutletPowerCycleOnAcRecoveryEnabled:   true,
		OutletPowerCycleOnAcRecoverySeconds:   &acRecoverySeconds,
		OutletPowerCycleOnInternetLossSeconds: &internetLossSeconds,
		OutletOverrides: []DeviceOutletOverrides{{
			Index:        &idx,
			RelayState:   false,
			CycleEnabled: true,
		}},
	})
	if err != nil {
		t.Fatalf("UpdateDevice: %v", err)
	}

	for key := range updateBody {
		if strings.HasPrefix(key, "outlet_") {
			t.Errorf("update body contains prohibited key %q: %#v", key, updateBody)
		}
	}
	if updateBody["name"] != "new" {
		t.Errorf("update body name = %#v, want new", updateBody["name"])
	}
}
