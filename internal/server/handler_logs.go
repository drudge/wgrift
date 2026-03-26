package server

import (
	"net/http"
	"strconv"
)

func (s *Server) handleInterfaceLogs(w http.ResponseWriter, r *http.Request) {
	ifaceID := r.PathValue("ifaceId")
	limit, offset := parsePagination(r)

	logs, total, err := s.store.ListConnectionLogs(ifaceID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSONList(w, http.StatusOK, logs, total)
}

func (s *Server) handlePeerLogs(w http.ResponseWriter, r *http.Request) {
	peerID := r.PathValue("id")
	limit, offset := parsePagination(r)

	logs, total, err := s.store.ListPeerConnectionLogs(peerID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSONList(w, http.StatusOK, logs, total)
}

func parsePagination(r *http.Request) (int, int) {
	limit := 50
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	return limit, offset
}
