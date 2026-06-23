package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer() *Server {
	return New(Config{Addr: "127.0.0.1:0"}, nil, nil)
}

func TestHealthz(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	srv.MetricsHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "ok" {
		t.Errorf("corps = %q, want \"ok\"", got)
	}
}

func TestMetricsReflectsCounters(t *testing.T) {
	srv := newTestServer()
	// Simule l'activité comptée par serveConn.
	srv.connTotal.Add(7)
	srv.connActive.Add(2)
	srv.connRejected.Add(3)

	st := srv.Stats()
	if st.ConnTotal != 7 || st.ConnActive != 2 || st.ConnRejected != 3 {
		t.Fatalf("Stats incohérent : %+v", st)
	}
	if st.Uptime < 0 {
		t.Errorf("uptime négatif : %v", st.Uptime)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	srv.MetricsHandler().ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, want := range []string{
		"bbsoric_connections_total 7",
		"bbsoric_connections_active 2",
		"bbsoric_connections_rejected_total 3",
		"# TYPE bbsoric_uptime_seconds gauge",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("métriques sans %q :\n%s", want, body)
		}
	}
}
