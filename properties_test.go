package exaroton

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func boolPtr(v bool) *bool {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func TestGetProperties(t *testing.T) {
	var gotPath string

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		encodeEnvelope(t, w, true, nil, map[string]any{
			"autostop":       true,
			"autostop-time":  10,
			"autostart-type": "whitelisted",
			"auto-update":    "ask",
		})
	})

	props, err := c.Server(testServerID).GetProperties(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if gotPath != "/servers/"+testServerID+"/options/properties/" {
		t.Fatalf("path = %q", gotPath)
	}
	if props.AutoStop == nil || !*props.AutoStop {
		t.Fatalf("autostop = %v", props.AutoStop)
	}
	if props.AutoStopTime == nil || *props.AutoStopTime != 10 {
		t.Fatalf("autostop-time = %v", props.AutoStopTime)
	}
	if props.AutoStartType == nil || *props.AutoStartType != AutoStartTypeWhitelisted {
		t.Fatalf("autostart-type = %v", props.AutoStartType)
	}
	if props.AutoUpdate == nil || *props.AutoUpdate != AutoUpdateAsk {
		t.Fatalf("auto-update = %v", props.AutoUpdate)
	}
	if props.ForceUpgrade != nil {
		t.Fatalf("force-upgrade should be nil, got %v", props.ForceUpgrade)
	}
}

func TestGetPropertiesEmpty(t *testing.T) {
	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		encodeEnvelope(t, w, true, nil, map[string]any{})
	})

	props, err := c.Server(testServerID).GetProperties(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if props == nil {
		t.Fatal("expected empty properties, got nil")
	}
	if props.AutoStop != nil || props.AutoStopTime != nil || props.AutoStart != nil ||
		props.AutoStartType != nil || props.LauncherAutoStart != nil ||
		props.LauncherAutoStartType != nil || props.EnableLobby != nil ||
		props.AutoUpdate != nil || props.ForceUpgrade != nil {
		t.Fatalf("expected all fields nil, got %+v", props)
	}
}

func TestUpdateProperties(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	c := accountTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		encodeEnvelope(t, w, true, nil, map[string]any{
			"autostop":      false,
			"autostop-time": 5,
			"auto-update":   "never",
		})
	})

	props, err := c.Server(testServerID).UpdateProperties(context.Background(), ServerProperties{
		AutoStop:     boolPtr(false),
		AutoStopTime: intPtr(5),
		AutoUpdate:   autoUpdatePtr(AutoUpdateNever),
	})
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s", gotMethod)
	}
	if gotPath != "/servers/"+testServerID+"/options/properties/" {
		t.Fatalf("path = %q", gotPath)
	}
	if len(gotBody) != 3 {
		t.Fatalf("unexpected body: %v", gotBody)
	}
	if v, ok := gotBody["autostop"]; !ok || v != false {
		t.Fatalf("autostop not sent explicitly: %v", gotBody)
	}
	if gotBody["autostop-time"] != float64(5) {
		t.Fatalf("autostop-time = %v", gotBody["autostop-time"])
	}
	if gotBody["auto-update"] != "never" {
		t.Fatalf("auto-update = %v", gotBody["auto-update"])
	}
	if props.AutoStop == nil || *props.AutoStop {
		t.Fatalf("response autostop = %v", props.AutoStop)
	}
}

func autoUpdatePtr(v AutoUpdateMode) *AutoUpdateMode {
	return &v
}

func TestUpdatePropertiesValidation(t *testing.T) {
	c, _ := NewClient("tok", WithBaseURL("http://127.0.0.1:1/"))
	server := c.Server(testServerID)
	ctx := context.Background()

	badAutoStart := AutoStartType("sometimes")
	badUpdate := AutoUpdateMode("weekly")
	two := 2
	tooLong := maxAutoStopMinutes + 1

	tests := []struct {
		name  string
		props ServerProperties
	}{
		{"empty", ServerProperties{}},
		{"autostop-time too small", ServerProperties{AutoStopTime: &two}},
		{"autostop-time too large", ServerProperties{AutoStopTime: &tooLong}},
		{"invalid autostart type", ServerProperties{AutoStartType: &badAutoStart}},
		{"invalid launcher autostart type", ServerProperties{LauncherAutoStartType: &badAutoStart}},
		{"invalid auto-update", ServerProperties{AutoUpdate: &badUpdate}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := server.UpdateProperties(ctx, tt.props); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
