package jmx

import (
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"strings"
	"testing"

	"github.com/devaryakjha/loadwright/internal/spec"
)

func TestRenderProducesValidJMX(t *testing.T) {
	loops := 2
	loaded := &spec.Spec{
		Name:   "api smoke",
		Target: "https://example.com",
		Load: spec.Load{
			Users:  5,
			RampUp: spec.Duration{Seconds: 10, Set: true},
			Loops:  &loops,
		},
		Requests: []spec.Request{
			{
				Name:    "GET health",
				Method:  "GET",
				Path:    "/health",
				Headers: map[string]string{"accept": "application/json"},
				Expect:  spec.Expect{Status: 200},
			},
		},
	}

	rendered := Render(loaded)
	var root struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal([]byte(rendered), &root); err != nil {
		t.Fatalf("rendered JMX is not XML: %v\n%s", err, rendered)
	}
	for _, expected := range []string{
		`testname="api smoke"`,
		`<stringProp name="HTTPSampler.domain">example.com</stringProp>`,
		`<stringProp name="HTTPSampler.path">/health</stringProp>`,
		`<stringProp name="Header.name">accept</stringProp>`,
		`<stringProp name="Assertion.test_field">Assertion.response_code</stringProp>`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("rendered JMX missing %q\n%s", expected, rendered)
		}
	}
}

func TestExampleGoldenHash(t *testing.T) {
	loops := 3
	loaded := &spec.Spec{
		Name:   "httpbin-basic",
		Target: "https://httpbin.org",
		Load: spec.Load{
			Users:  5,
			RampUp: spec.Duration{Seconds: 10, Set: true},
			Loops:  &loops,
		},
		Requests: []spec.Request{
			{
				Name:   "get status",
				Method: "GET",
				Path:   "/status/200",
				Expect: spec.Expect{Status: 200},
			},
			{
				Name:    "post json",
				Method:  "POST",
				Path:    "/post",
				Headers: map[string]string{"content-type": "application/json"},
				Body: map[string]any{
					"kind":   "example",
					"source": "Loadwright",
				},
				Expect: spec.Expect{Status: 200},
			},
		},
	}

	hash := sha256.Sum256([]byte(Render(loaded)))
	got := fmt.Sprintf("%x", hash)
	const want = "991b2afee60b1b676317415750e2625db7a6bc52469db0d3e63f4099a5f0d94e"
	if got != want {
		t.Fatalf("JMX golden hash changed: got %s, want %s", got, want)
	}
}

func TestRenderQueryParamsAndJSONBody(t *testing.T) {
	loops := 1
	loaded := &spec.Spec{
		Name:   "request shapes",
		Target: "https://api.example.com:8443",
		Load:   spec.Load{Users: 1, RampUp: spec.Duration{Seconds: 1, Set: true}, Loops: &loops},
		Requests: []spec.Request{
			{
				Name:   "search",
				Method: "GET",
				Path:   "/search",
				Query:  map[string]string{"page": "1", "q": "load test"},
			},
			{
				Name:    "create",
				Method:  "POST",
				Path:    "/items",
				Headers: map[string]string{"content-type": "application/json"},
				Body:    map[string]any{"name": "demo", "active": true},
			},
		},
	}
	rendered := Render(loaded)
	for _, expected := range []string{
		`<stringProp name="HTTPSampler.port">8443</stringProp>`,
		`<stringProp name="Argument.name">page</stringProp>`,
		`<stringProp name="Argument.value">load test</stringProp>`,
		`<boolProp name="HTTPSampler.postBodyRaw">true</boolProp>`,
		`{&#34;active&#34;:true,&#34;name&#34;:&#34;demo&#34;}`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("rendered JMX missing %q\n%s", expected, rendered)
		}
	}
}

func TestRenderFormBodyArguments(t *testing.T) {
	loops := 1
	loaded := &spec.Spec{
		Name:   "form body",
		Target: "https://api.example.com",
		Load:   spec.Load{Users: 1, RampUp: spec.Duration{Seconds: 1, Set: true}, Loops: &loops},
		Requests: []spec.Request{{
			Name:   "login",
			Method: "POST",
			Path:   "/login",
			BodyForm: map[string]string{
				"password": "secret",
				"username": "demo",
			},
		}},
	}
	rendered := Render(loaded)
	for _, expected := range []string{
		`<boolProp name="HTTPSampler.postBodyRaw">false</boolProp>`,
		`<elementProp name="password" elementType="HTTPArgument">`,
		`<stringProp name="Argument.name">username</stringProp>`,
		`<stringProp name="Argument.value">demo</stringProp>`,
		`<stringProp name="Header.name">Content-Type</stringProp>`,
		`<stringProp name="Header.value">application/x-www-form-urlencoded</stringProp>`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("form JMX missing %q\n%s", expected, rendered)
		}
	}
	if strings.Contains(rendered, `HTTPSampler.postBodyRaw">true`) || strings.Contains(rendered, `{&#34;username&#34;`) {
		t.Fatalf("form body rendered as raw JSON\n%s", rendered)
	}
}

func TestRenderDurationBasedLoad(t *testing.T) {
	loaded := &spec.Spec{
		Name:   "duration load",
		Target: "https://example.com",
		Load: spec.Load{
			Users:    10,
			RampUp:   spec.Duration{Seconds: 30, Set: true},
			Duration: spec.Duration{Seconds: 120, Set: true},
		},
		Requests: []spec.Request{{Name: "health", Method: "GET", Path: "/health"}},
	}
	rendered := Render(loaded)
	for _, expected := range []string{
		`<boolProp name="LoopController.continue_forever">true</boolProp>`,
		`<stringProp name="LoopController.loops">-1</stringProp>`,
		`<boolProp name="ThreadGroup.scheduler">true</boolProp>`,
		`<stringProp name="ThreadGroup.duration">120</stringProp>`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("duration JMX missing %q\n%s", expected, rendered)
		}
	}
}

func TestRenderTimeouts(t *testing.T) {
	loops := 1
	loaded := &spec.Spec{
		Name:   "timeouts",
		Target: "https://example.com",
		Load:   spec.Load{Users: 1, RampUp: spec.Duration{Seconds: 1, Set: true}, Loops: &loops},
		Requests: []spec.Request{{
			Name:    "slow",
			Method:  "GET",
			Path:    "/slow",
			Timeout: spec.Duration{Seconds: 2, Set: true},
		}},
	}
	rendered := Render(loaded)
	for _, expected := range []string{
		`<stringProp name="HTTPSampler.connect_timeout">2000</stringProp>`,
		`<stringProp name="HTTPSampler.response_timeout">2000</stringProp>`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("timeout JMX missing %q\n%s", expected, rendered)
		}
	}
}

func TestRenderCSVDataSet(t *testing.T) {
	loops := 1
	recycle := false
	stopThread := true
	loaded := &spec.Spec{
		Name:   "csv",
		Target: "https://example.com",
		Data: map[string]spec.DataSet{
			"users": {
				File:       "users.csv",
				Variables:  []string{"username", "password"},
				Recycle:    &recycle,
				StopThread: &stopThread,
				Sharing:    "thread",
			},
		},
		Load: spec.Load{Users: 1, RampUp: spec.Duration{Seconds: 1, Set: true}, Loops: &loops},
		Requests: []spec.Request{{
			Name:   "login",
			Method: "POST",
			Path:   "/login",
			Body: map[string]any{
				"username": "${username}",
				"password": "${password}",
			},
		}},
	}
	rendered := Render(loaded)
	for _, expected := range []string{
		`<CSVDataSet guiclass="TestBeanGUI" testclass="CSVDataSet" testname="users" enabled="true">`,
		`<stringProp name="filename">users.csv</stringProp>`,
		`<stringProp name="variableNames">username,password</stringProp>`,
		`<boolProp name="ignoreFirstLine">true</boolProp>`,
		`<boolProp name="recycle">false</boolProp>`,
		`<boolProp name="stopThread">true</boolProp>`,
		`<stringProp name="shareMode">Current thread</stringProp>`,
		`${username}`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("CSV JMX missing %q\n%s", expected, rendered)
		}
	}
}
