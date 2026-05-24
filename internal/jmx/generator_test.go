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

func TestRenderMultipartBodyArguments(t *testing.T) {
	loops := 1
	title := "avatar"
	loaded := &spec.Spec{
		Name:   "multipart body",
		Target: "https://api.example.com",
		Load:   spec.Load{Users: 1, RampUp: spec.Duration{Seconds: 1, Set: true}, Loops: &loops},
		Requests: []spec.Request{{
			Name:    "upload",
			Method:  "POST",
			Path:    "/upload",
			Headers: map[string]string{"Content-Type": "multipart/form-data", "X-Trace": "yes"},
			BodyMultipart: []spec.MultipartPart{
				{Name: "title", Value: &title},
				{Name: "avatar", File: "fixtures/avatar.png", ContentType: "image/png"},
			},
		}},
	}
	rendered := Render(loaded)
	for _, expected := range []string{
		`<boolProp name="HTTPSampler.postBodyRaw">false</boolProp>`,
		`<boolProp name="HTTPSampler.DO_MULTIPART_POST">true</boolProp>`,
		`<elementProp name="title" elementType="HTTPArgument">`,
		`<stringProp name="Argument.value">avatar</stringProp>`,
		`<elementProp name="HTTPsampler.Files" elementType="HTTPFileArgs">`,
		`<stringProp name="File.path">fixtures/avatar.png</stringProp>`,
		`<stringProp name="File.paramname">avatar</stringProp>`,
		`<stringProp name="File.mimetype">image/png</stringProp>`,
		`<stringProp name="Header.name">X-Trace</stringProp>`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("multipart JMX missing %q\n%s", expected, rendered)
		}
	}
	if strings.Contains(rendered, `<stringProp name="Header.name">Content-Type</stringProp>`) {
		t.Fatalf("multipart JMX should let JMeter set Content-Type boundary\n%s", rendered)
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

// --- WebSocket Sampler Tests (using eu.luminis.jmeter.wssampler plugin) ---

func TestRenderWebSocketSampler(t *testing.T) {
	loops := 1
	loaded := &spec.Spec{
		Name:   "ws",
		Target: "wss://echo.example.com",
		Load:   spec.Load{Users: 1, RampUp: spec.Duration{Seconds: 1, Set: true}, Loops: &loops},
		Requests: []spec.Request{{
			Name:     "echo ping",
			Protocol: "websocket",
			Path:     "/socket",
			WebSocket: spec.WebSocket{
				Subprotocol: "echo-protocol",
				Headers: map[string]string{
					"X-Custom-Header": "test-value",
				},
				Timeout: spec.Duration{Seconds: 3, Set: true},
				Messages: []spec.WSMessage{{
					Send: "ping",
					Type: "text",
					Expect: &spec.WSExpect{
						Contains: "pong",
						Timeout:  spec.Duration{Seconds: 3, Set: true},
					},
				}},
			},
		}},
	}
	rendered := Render(loaded)
	for _, expected := range []string{
		`testclass="eu.luminis.jmeter.wssampler.RequestResponseWebSocketSampler"`,
		`testname="echo ping"`,
		`<stringProp name="server">echo.example.com</stringProp>`,
		`<stringProp name="port">443</stringProp>`,
		`<stringProp name="path">/socket</stringProp>`,
		`<boolProp name="TLS">true</boolProp>`,
		`<stringProp name="connectTimeout">3000</stringProp>`,
		`<stringProp name="readTimeout">3000</stringProp>`,
		`<stringProp name="requestData">ping</stringProp>`,
		`<boolProp name="createNewConnection">true</boolProp>`,
		`<HeaderManager guiclass="HeaderPanel" testclass="HeaderManager" testname="WebSocket Handshake Headers" enabled="true">`,
		`<stringProp name="Header.name">Sec-WebSocket-Protocol</stringProp>`,
		`<stringProp name="Header.value">echo-protocol</stringProp>`,
		`<stringProp name="Header.name">X-Custom-Header</stringProp>`,
		`<stringProp name="Header.value">test-value</stringProp>`,
		// Response assertion for expect_contains
		`<stringProp name="0">pong</stringProp>`,
		`Assertion.response_data`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("websocket JMX missing %q\n%s", expected, rendered)
		}
	}
}

func TestRenderWebSocketMultiMessage(t *testing.T) {
	loops := 1
	loaded := &spec.Spec{
		Name:   "ws-multi",
		Target: "wss://echo.example.com",
		Load:   spec.Load{Users: 1, RampUp: spec.Duration{Seconds: 1, Set: true}, Loops: &loops},
		Requests: []spec.Request{{
			Name:     "multi",
			Protocol: "websocket",
			WebSocket: spec.WebSocket{
				Timeout: spec.Duration{Seconds: 5, Set: true},
				CloseTimeout: spec.Duration{
					Seconds: 2,
					Set:     true,
				},
				Messages: []spec.WSMessage{
					{Send: "hello", Type: "text", Expect: &spec.WSExpect{Contains: "hello", Timeout: spec.Duration{Seconds: 5, Set: true}}},
					{Send: "world", Type: "text"},
				},
			},
		}},
	}
	rendered := Render(loaded)
	for _, expected := range []string{
		// Transaction controller wraps multi-message
		`testclass="TransactionController" testname="multi"`,
		// Open connection
		`testclass="eu.luminis.jmeter.wssampler.OpenWebSocketSampler"`,
		// First message: request-response with reuse
		`testclass="eu.luminis.jmeter.wssampler.RequestResponseWebSocketSampler"`,
		`<stringProp name="requestData">hello</stringProp>`,
		`<boolProp name="createNewConnection">false</boolProp>`,
		// Second message: fire-and-forget write
		`testclass="eu.luminis.jmeter.wssampler.SingleWriteWebSocketSampler"`,
		`<stringProp name="requestData">world</stringProp>`,
		// Close connection
		`testclass="eu.luminis.jmeter.wssampler.CloseWebSocketSampler"`,
		`<stringProp name="readTimeout">2000</stringProp>`,
		// Contains assertion for first message
		`<stringProp name="0">hello</stringProp>`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("multi-message JMX missing %q\n%s", expected, rendered)
		}
	}
}

func TestRenderWebSocketBinaryMessage(t *testing.T) {
	loops := 1
	loaded := &spec.Spec{
		Name:   "ws-bin",
		Target: "wss://echo.example.com",
		Load:   spec.Load{Users: 1, RampUp: spec.Duration{Seconds: 1, Set: true}, Loops: &loops},
		Requests: []spec.Request{{
			Name:     "binary",
			Protocol: "websocket",
			WebSocket: spec.WebSocket{
				Messages: []spec.WSMessage{{Send: "aGVsbG8=", Type: "binary", Expect: &spec.WSExpect{Contains: "hello"}}},
			},
		}},
	}
	rendered := Render(loaded)
	if !strings.Contains(rendered, `<stringProp name="payloadType">Binary</stringProp>`) {
		t.Fatalf("binary JMX missing Binary payloadType\n%s", rendered)
	}
}

func TestRenderWebSocketWithDelay(t *testing.T) {
	loops := 1
	loaded := &spec.Spec{
		Name:   "ws-delay",
		Target: "wss://echo.example.com",
		Load:   spec.Load{Users: 1, RampUp: spec.Duration{Seconds: 1, Set: true}, Loops: &loops},
		Requests: []spec.Request{{
			Name:     "delay",
			Protocol: "websocket",
			WebSocket: spec.WebSocket{
				Messages: []spec.WSMessage{
					{Send: "first", Type: "text", Expect: &spec.WSExpect{Contains: "first"}},
					{Send: "delayed", Type: "text", Delay: spec.Duration{Seconds: 2, Set: true}},
				},
			},
		}},
	}
	rendered := Render(loaded)
	if !strings.Contains(rendered, `<stringProp name="ConstantTimer.delay">2000</stringProp>`) {
		t.Fatalf("delay JMX missing ConstantTimer\n%s", rendered)
	}
}

func TestRenderWebSocketLegacyCompat(t *testing.T) {
	loops := 1
	loaded := &spec.Spec{
		Name:   "ws-legacy",
		Target: "wss://echo.example.com",
		Load:   spec.Load{Users: 1, RampUp: spec.Duration{Seconds: 1, Set: true}, Loops: &loops},
		Requests: []spec.Request{{
			Name:     "legacy",
			Protocol: "websocket",
			Path:     "/echo",
			WebSocket: spec.WebSocket{
				Message:        "ping",
				ExpectContains: "pong",
				Timeout:        spec.Duration{Seconds: 5, Set: true},
			},
		}},
	}
	// NormalizeAndValidate normalizes legacy fields into Messages[].
	if err := loaded.NormalizeAndValidate(); err != nil {
		t.Fatalf("NormalizeAndValidate() error = %v", err)
	}
	rendered := Render(loaded)
	for _, expected := range []string{
		`testclass="eu.luminis.jmeter.wssampler.RequestResponseWebSocketSampler"`,
		`<stringProp name="requestData">ping</stringProp>`,
		`<stringProp name="connectTimeout">5000</stringProp>`,
		`<stringProp name="0">pong</stringProp>`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("legacy compat JMX missing %q\n%s", expected, rendered)
		}
	}
}

func TestRenderWebSocketURLParsing(t *testing.T) {
	tests := []struct {
		url    string
		server string
		port   string
		path   string
		useTLS bool
	}{
		{"wss://echo.example.com/socket", "echo.example.com", "443", "/socket", true},
		{"ws://localhost:8080/ws", "localhost", "8080", "/ws", false},
		{"wss://example.com", "example.com", "443", "/", true},
		{"ws://example.com:9090/path/to/ws", "example.com", "9090", "/path/to/ws", false},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			server, port, path, useTLS := parseWebSocketURL(tt.url)
			if server != tt.server || port != tt.port || path != tt.path || useTLS != tt.useTLS {
				t.Fatalf("parseWebSocketURL(%q) = (%q, %q, %q, %t), want (%q, %q, %q, %t)",
					tt.url, server, port, path, useTLS, tt.server, tt.port, tt.path, tt.useTLS)
			}
		})
	}
}
