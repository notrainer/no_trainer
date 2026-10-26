package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := Config{Secret: "test-secret", Users: []*User{{Username: "user1"}}}
	for _, test := range []struct {
		name   string
		user   interface{}
		status int
	}{
		{"missing cookie", nil, http.StatusFound},
		{"assigned user", "user1", http.StatusOK},
		{"unknown user", "other", http.StatusUnauthorized},
		{"invalid claim type", 42, http.StatusFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, router := gin.CreateTestContext(recorder)
			loadTemplates(router)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			if test.user != nil {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user": test.user, "exp": time.Now().Add(time.Minute).Unix()})
				signed, err := token.SignedString([]byte(config.Secret))
				if err != nil {
					t.Fatal(err)
				}
				c.Request.AddCookie(&http.Cookie{Name: "jwt", Value: signed})
			}
			user := authenticatedUser(c, config)
			if recorder.Code != test.status {
				t.Fatalf("status %d, want %d", recorder.Code, test.status)
			}
			if (user != nil) != (test.status == http.StatusOK) {
				t.Fatal("authentication result does not match status")
			}
		})
	}
}

func TestAssignedAccessInstructions(t *testing.T) {
	t.Setenv("NO_TRAINER_USERS", `[{"username":"user7","password":"never-render-this-password","targets":[{"name":"client7","icon":"vnc-icon","url":"https://nmap.b3n.eu/client7/","ssh_command":"ssh -p 30007 abc@nmap.b3n.eu","ssh_host_key":"SHA256:example"}]}]`)
	t.Setenv("NO_TRAINER_TARGETS", "")
	config, err := loadConfig(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	c, router := gin.CreateTestContext(recorder)
	loadTemplates(router)
	c.HTML(http.StatusOK, "index.html", gin.H{"User": config.Users[0], "Targets": config.Users[0].Targets})
	body := recorder.Body.String()
	for _, expected := range []string{"ssh -p 30007 abc@nmap.b3n.eu", "SHA256:example", "/materials/nmap-training.pdf", "Passwort deiner Portal-Anmeldung"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("missing instruction: %s", expected)
		}
	}
	for _, forbidden := range []string{"30001", "never-render-this-password"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("leaked unrelated access information: %s", forbidden)
		}
	}
}

func TestPortalRoutesRequireSession(t *testing.T) {
	config := Config{Secret: "test-secret", Users: []*User{{Username: "user1", Password: "test-password", Targets: TargetList{{Name: "client1", Icon: "vnc-icon", Url: "/training-client/"}}}}}
	router := setupRouter(config)
	for _, path := range []string{"/login", "/portal-assets/main.css", "/portal-assets/main.js"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK || recorder.Body.Len() == 0 {
			t.Fatalf("%s: expected a nonempty HTTP 200 response, got %d", path, recorder.Code)
		}
	}
	for _, path := range []string{"/", "/materials/nmap-training.pdf", "/materials/nmap-day1.pdf", "/materials/nmap-day2.pdf", "/materials/nmap-day3.pdf"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusFound || recorder.Header().Get("Location") != "/login" {
			t.Fatalf("%s: anonymous request was not redirected to login", path)
		}
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("username=user1&password=test-password"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(recorder, request)
	cookies := recorder.Result().Cookies()
	if recorder.Code != http.StatusFound || len(cookies) != 1 || cookies[0].MaxAge != 28800 || !cookies[0].HttpOnly {
		t.Fatalf("login did not create an eight-hour HttpOnly session: %v", recorder.Header())
	}
	claims, err := ValidateJWT(cookies[0].Value, config.Secret)
	if err != nil || claims["user"] != "user1" {
		t.Fatalf("invalid login token: %v", err)
	}
	expires := time.Unix(int64(claims["exp"].(float64)), 0)
	if remaining := time.Until(expires); remaining < 8*time.Hour-time.Minute || remaining > 8*time.Hour {
		t.Fatalf("unexpected token lifetime: %s", remaining)
	}

	request = httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(cookies[0])
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `data-tabs-target="#vnc-client1"`) || !strings.Contains(recorder.Body.String(), `src="/training-client/"`) {
		t.Fatal("authenticated home page did not render assigned target navigation")
	}

	request = httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("username=user1&password=wrong-password"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized || len(recorder.Result().Cookies()) != 0 {
		t.Fatal("rejected credentials did not return HTTP 401 without a session")
	}

	request = httptest.NewRequest(http.MethodGet, "/materials/not-course-material.pdf", nil)
	request.AddCookie(cookies[0])
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatal("authenticated request bypassed material filename whitelist")
	}

	for _, test := range []struct {
		name    string
		secret  string
		expires time.Time
	}{
		{"expired", config.Secret, time.Now().Add(-time.Minute)},
		{"wrong signature", "other-secret", time.Now().Add(time.Hour)},
	} {
		t.Run(test.name, func(t *testing.T) {
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user": "user1", "exp": test.expires.Unix()})
			signed, err := token.SignedString([]byte(test.secret))
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodGet, "/materials/nmap-training.pdf", nil)
			request.AddCookie(&http.Cookie{Name: "jwt", Value: signed})
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusFound || recorder.Header().Get("Location") != "/login" {
				t.Fatal("invalid token granted access to materials")
			}
		})
	}
}

func TestGatewayDesktopAuthorization(t *testing.T) {
	config := Config{Secret: "test-secret", Users: []*User{{Username: "user1", Targets: TargetList{{Url: "https://nmap.b3n.eu/client1/"}, {Url: "https://nmap.b3n.eu/"}}}}}
	router := setupRouter(config)
	valid, err := GenerateJWT("user1", config.Secret)
	if err != nil {
		t.Fatal(err)
	}
	unknown, err := GenerateJWT("unknown", config.Secret)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, token, uri string
		status           int
	}{
		{"missing session", "", "/client1/", http.StatusUnauthorized},
		{"invalid session", "invalid", "/client1/", http.StatusUnauthorized},
		{"unknown user", unknown, "/client1/", http.StatusUnauthorized},
		{"assigned desktop", valid, "/client1/", http.StatusNoContent},
		{"assigned root", valid, "/client1", http.StatusNoContent},
		{"assigned assets", valid, "/client1/portal-assets/main.js?v=1", http.StatusNoContent},
		{"assigned socket", valid, "/client1/ws", http.StatusNoContent},
		{"other desktop", valid, "/client2/", http.StatusForbidden},
		{"prefix collision", valid, "/client10/", http.StatusForbidden},
		{"traversal", valid, "/client1/../client2/", http.StatusForbidden},
		{"encoded traversal", valid, "/client1/%2e%2e/client2/", http.StatusForbidden},
		{"encoded slash traversal", valid, "/client1%2f..%2fclient2/", http.StatusForbidden},
		{"no original URI", valid, "", http.StatusForbidden},
		{"absolute URI", valid, "https://other.example/client1/", http.StatusForbidden},
		{"root target grants nothing", valid, "/client7/", http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/internal/auth", nil)
			request.Header.Set("X-Original-URI", test.uri)
			if test.token != "" {
				request.AddCookie(&http.Cookie{Name: "jwt", Value: test.token})
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != test.status {
				t.Fatalf("status %d, want %d", recorder.Code, test.status)
			}
			if recorder.Header().Get("WWW-Authenticate") != "" {
				t.Fatal("Gateway check must not prompt for Basic Auth")
			}
		})
	}
}

func TestWebluaVisibilityAndAuthentication(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		config := Config{Secret: "test-secret", Weblua: enabled, Users: []*User{{Username: "user1"}}}
		router := setupRouter(config)
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusFound || strings.Contains(recorder.Body.String(), "/playground") {
			t.Fatal("anonymous request exposed the Weblua")
		}
		token, err := GenerateJWT("user1", config.Secret)
		if err != nil {
			t.Fatal(err)
		}
		request = httptest.NewRequest(http.MethodGet, "/", nil)
		request.AddCookie(&http.Cookie{Name: "jwt", Value: token})
		recorder = httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		body := recorder.Body.String()
		if strings.Contains(body, "nmap-visualizer") || strings.Contains(body, "nmap-file") {
			t.Fatal("portal still renders the XML viewer")
		}
		if recorder.Code != http.StatusOK || strings.Contains(body, `data-tabs-target="#weblua"`) != enabled || strings.Contains(body, `src="/playground"`) != enabled {
			t.Fatalf("Weblua menu/panel did not follow enabled=%v", enabled)
		}
	}
}

func TestGatewayWebluaAuthorization(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		config := Config{Secret: "test-secret", Weblua: enabled, Users: []*User{{Username: "user1"}}}
		router := setupRouter(config)
		token, err := GenerateJWT("user1", config.Secret)
		if err != nil {
			t.Fatal(err)
		}
		for _, test := range []struct {
			uri     string
			allowed bool
		}{
			{"/playground", true}, {"/playground/", true}, {"/playground?example=1", true},
			{"/embed", true}, {"/assets/index-test.js", true}, {"/assets/runtime.wasm", true},
			{"/favicon.svg", true}, {"/favicon.png", true}, {"/apple-touch-icon.png", true},
			{"/site.webmanifest", true}, {"/icon-192.png", true}, {"/icon-512.png", true},
			{"/playground-evil", false}, {"/assets/../client2/", false},
			{"/assets/%2e%2e/client2/", false}, {"/client2/", false},
			{"https://other.example/playground", false},
		} {
			for _, authenticated := range []bool{false, true} {
				request := httptest.NewRequest(http.MethodGet, "/internal/auth", nil)
				request.Header.Set("X-Original-URI", test.uri)
				if authenticated {
					request.AddCookie(&http.Cookie{Name: "jwt", Value: token})
				}
				want := http.StatusUnauthorized
				if authenticated {
					want = http.StatusForbidden
					if enabled && test.allowed {
						want = http.StatusNoContent
					}
				}
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, request)
				if recorder.Code != want {
					t.Fatalf("Weblua enabled=%v authenticated=%v %s: status %d, want %d", enabled, authenticated, test.uri, recorder.Code, want)
				}
			}
		}
	}
}
