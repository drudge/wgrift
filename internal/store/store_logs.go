package store

import (
	"fmt"
	"time"

	"github.com/drudge/wgrift/internal/models"
)

func (s *SQLiteStore) CreateConnectionLog(log *models.ConnectionLog) error {
	if log.RecordedAt.IsZero() {
		log.RecordedAt = time.Now().UTC()
	}

	result, err := s.db.Exec(`
		INSERT INTO connection_logs (peer_id, interface_id, event, endpoint, transfer_rx, transfer_tx, recorded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		log.PeerID, log.InterfaceID, log.Event, log.Endpoint,
		log.TransferRx, log.TransferTx, log.RecordedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting connection log: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		log.ID = id
	}
	return nil
}

func (s *SQLiteStore) ListConnectionLogs(interfaceID string, limit, offset int) ([]models.ConnectionLog, int, error) {
	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM connection_logs WHERE interface_id = ?", interfaceID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting connection logs: %w", err)
	}

	rows, err := s.db.Query(`
		SELECT cl.id, cl.peer_id, COALESCE(p.name, ''), cl.interface_id, cl.event, cl.endpoint, cl.transfer_rx, cl.transfer_tx, cl.recorded_at
		FROM connection_logs cl
		LEFT JOIN peers p ON p.id = cl.peer_id
		WHERE cl.interface_id = ?
		ORDER BY cl.recorded_at DESC LIMIT ? OFFSET ?`, interfaceID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying connection logs: %w", err)
	}
	defer rows.Close()

	logs, err := scanConnectionLogs(rows)
	if err != nil {
		return nil, 0, err
	}
	computeLogDurations(logs)
	return logs, total, nil
}

func (s *SQLiteStore) ListPeerConnectionLogs(peerID string, limit, offset int) ([]models.ConnectionLog, int, error) {
	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM connection_logs WHERE peer_id = ?", peerID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting peer connection logs: %w", err)
	}

	rows, err := s.db.Query(`
		SELECT cl.id, cl.peer_id, COALESCE(p.name, ''), cl.interface_id, cl.event, cl.endpoint, cl.transfer_rx, cl.transfer_tx, cl.recorded_at
		FROM connection_logs cl
		LEFT JOIN peers p ON p.id = cl.peer_id
		WHERE cl.peer_id = ?
		ORDER BY cl.recorded_at DESC LIMIT ? OFFSET ?`, peerID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying peer connection logs: %w", err)
	}
	defer rows.Close()

	logs, err := scanConnectionLogs(rows)
	if err != nil {
		return nil, 0, err
	}
	computeLogDurations(logs)
	return logs, total, nil
}

// LastConnectedEvents returns the recorded_at of the most recent "connected"
// event for each peer, keyed by peer_id. Used to restore ConnectedSince on restart.
func (s *SQLiteStore) LastConnectedEvents() (map[string]time.Time, error) {
	rows, err := s.db.Query(`
		SELECT cl.peer_id, cl.recorded_at
		FROM connection_logs cl
		INNER JOIN (
			SELECT peer_id, MAX(id) AS max_id
			FROM connection_logs
			WHERE event = 'connected'
			GROUP BY peer_id
		) latest ON cl.id = latest.max_id
		WHERE NOT EXISTS (
			SELECT 1 FROM connection_logs cl2
			WHERE cl2.peer_id = cl.peer_id
			  AND cl2.event = 'disconnected'
			  AND cl2.id > cl.id
		)`)
	if err != nil {
		return nil, fmt.Errorf("querying last connected events: %w", err)
	}
	defer rows.Close()

	m := make(map[string]time.Time)
	for rows.Next() {
		var peerID string
		var t time.Time
		if err := rows.Scan(&peerID, &t); err != nil {
			return nil, fmt.Errorf("scanning last connected event: %w", err)
		}
		m[peerID] = t
	}
	return m, rows.Err()
}

func (s *SQLiteStore) DeleteOldConnectionLogs(before time.Time) error {
	_, err := s.db.Exec("DELETE FROM connection_logs WHERE recorded_at < ?", before)
	if err != nil {
		return fmt.Errorf("deleting old connection logs: %w", err)
	}
	return nil
}

func scanConnectionLogs(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]models.ConnectionLog, error) {
	var result []models.ConnectionLog
	for rows.Next() {
		var log models.ConnectionLog
		if err := rows.Scan(&log.ID, &log.PeerID, &log.PeerName, &log.InterfaceID, &log.Event, &log.Endpoint,
			&log.TransferRx, &log.TransferTx, &log.RecordedAt); err != nil {
			return nil, fmt.Errorf("scanning connection log: %w", err)
		}
		result = append(result, log)
	}
	return result, rows.Err()
}

// computeLogDurations fills in Duration (seconds) for disconnect events
// by finding the preceding connect event for the same peer in the list.
// Logs must be in descending order by recorded_at.
func computeLogDurations(logs []models.ConnectionLog) {
	// For each disconnect, scan forward (further back in time) for the matching connect
	for i := range logs {
		if logs[i].Event != "disconnected" {
			continue
		}
		for j := i + 1; j < len(logs); j++ {
			if logs[j].PeerID == logs[i].PeerID && logs[j].Event == "connected" {
				logs[i].Duration = int64(logs[i].RecordedAt.Sub(logs[j].RecordedAt).Seconds())
				break
			}
		}
	}
}
