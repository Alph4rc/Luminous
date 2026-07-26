package skills

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"luminous/internal/httpclient"
)

func TestQuerySchool_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/schools/XAUAT" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"message":"success","data":{"code":"XAUAT","name":"西安建筑科技大学","website":"https://xauatapi.xauat.site","features":["login","timetable","grade_query"]}}`))
	}))
	defer srv.Close()

	client := httpclient.NewClient(srv.URL, "", 10*time.Second)
	skill := NewQuerySchool(client)

	args := map[string]any{"school_code": "XAUAT"}
	result, err := skill.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	// Check the Luminous API response wrapper.
	if code, ok := parsed["code"].(float64); !ok || code != 200 {
		t.Errorf("expected code 200, got %v", parsed["code"])
	}
}

func TestQuerySchool_MissingSchoolCode(t *testing.T) {
	skill := NewQuerySchool(nil)

	_, err := skill.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing school_code, got nil")
	}
	if !strings.Contains(err.Error(), "school_code") {
		t.Errorf("error should mention school_code, got: %v", err)
	}
}

func TestQuerySchool_EmptySchoolCode(t *testing.T) {
	skill := NewQuerySchool(nil)

	_, err := skill.Execute(context.Background(), map[string]any{"school_code": ""})
	if err == nil {
		t.Fatal("expected error for empty school_code, got nil")
	}
}

func TestQuerySchool_NonStringSchoolCode(t *testing.T) {
	skill := NewQuerySchool(nil)

	_, err := skill.Execute(context.Background(), map[string]any{"school_code": 123})
	if err == nil {
		t.Fatal("expected error for non-string school_code, got nil")
	}
	if !strings.Contains(err.Error(), "字符串") {
		t.Errorf("error should mention type, got: %v", err)
	}
}

func TestQuerySchool_HTTP404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := httpclient.NewClient(srv.URL, "", 10*time.Second)
	skill := NewQuerySchool(client)

	_, err := skill.Execute(context.Background(), map[string]any{"school_code": "UNKNOWN"})
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should mention HTTP 404, got: %v", err)
	}
}

func TestQuerySchool_HTTP500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := httpclient.NewClient(srv.URL, "", 10*time.Second)
	skill := NewQuerySchool(client)

	_, err := skill.Execute(context.Background(), map[string]any{"school_code": "XAUAT"})
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention HTTP 500, got: %v", err)
	}
}

func TestQuerySchool_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	client := httpclient.NewClient(srv.URL, "", 10*time.Second)
	skill := NewQuerySchool(client)

	_, err := skill.Execute(context.Background(), map[string]any{"school_code": "XAUAT"})
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "JSON") {
		t.Errorf("error should mention JSON, got: %v", err)
	}
}

func TestQuerySchool_Definition(t *testing.T) {
	skill := NewQuerySchool(nil)
	def := skill.Definition()

	if def.Name != "query_school" {
		t.Errorf("expected name query_school, got %s", def.Name)
	}
	if def.InputSchema.Type != "object" {
		t.Errorf("expected schema type object, got %s", def.InputSchema.Type)
	}
	if _, ok := def.InputSchema.Properties["school_code"]; !ok {
		t.Error("expected school_code property in schema")
	}
	if len(def.InputSchema.Required) != 1 || def.InputSchema.Required[0] != "school_code" {
		t.Errorf("expected required to contain only school_code, got %v", def.InputSchema.Required)
	}
}
