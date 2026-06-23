package server

import (
	"fmt"
	"net/http"
)

// MetricsHandler renvoie un http.Handler exposant deux routes de supervision :
//
//   - GET /healthz : sonde de vivacité (200 « ok »), pour un probe externe
//     (timer systemd, uptime-kuma, Caddy health check…).
//   - GET /metrics : métriques au format texte exposé par Prometheus
//     (`# HELP`/`# TYPE` + valeurs), exploitables par n'importe quel scraper.
//
// Sécurité : ce handler n'applique aucune authentification et expose l'état du
// serveur. Il est destiné à une adresse **locale** (ex. 127.0.0.1) et ne doit
// jamais être joignable depuis Internet — le BBS, lui, écoute sur 0.0.0.0:6502.
func (s *Server) MetricsHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "ok")
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		st := s.Stats()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "# HELP bbsoric_uptime_seconds Temps écoulé depuis le démarrage du serveur.\n")
		fmt.Fprintf(w, "# TYPE bbsoric_uptime_seconds gauge\n")
		fmt.Fprintf(w, "bbsoric_uptime_seconds %d\n", int64(st.Uptime.Seconds()))
		fmt.Fprintf(w, "# HELP bbsoric_connections_total Connexions TCP acceptées (cumul).\n")
		fmt.Fprintf(w, "# TYPE bbsoric_connections_total counter\n")
		fmt.Fprintf(w, "bbsoric_connections_total %d\n", st.ConnTotal)
		fmt.Fprintf(w, "# HELP bbsoric_connections_active Sessions en cours.\n")
		fmt.Fprintf(w, "# TYPE bbsoric_connections_active gauge\n")
		fmt.Fprintf(w, "bbsoric_connections_active %d\n", st.ConnActive)
		fmt.Fprintf(w, "# HELP bbsoric_connections_rejected_total Connexions refusées par un garde-fou (cumul).\n")
		fmt.Fprintf(w, "# TYPE bbsoric_connections_rejected_total counter\n")
		fmt.Fprintf(w, "bbsoric_connections_rejected_total %d\n", st.ConnRejected)
	})

	return mux
}
