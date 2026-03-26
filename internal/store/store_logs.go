package store

import (
	"fmt"
	"time"

	"github.com/drudge/wgrift/internal/models"
)

func (s *SQLiteStore) CreateConnectionLog(log *models.ConnectionLog) error {
	now := time.Now().UTC()
	log.RecordedAt = now

	result, err := s.db.Exec(`
		INSERT INTO connection_logs (peer_id, interface_id, event, transfer_rx, transfer_tx, recorded_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		log.PeerID, log.InterfaceID, log.Event,
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
		SELECT cl.id, cl.peer_id, COALESCE(p.name, ''), cl.interface_id, cl.event, cl.transfer_rx, cl.transfer_tx, cl.recorded_at
		FROM connection_logs cl
		LEFT JOIN peers p ON p.id = cl.peer_id
		WHERE cl.interface_id = ?
		ORDER BY cl.recorded_at DESC LIMIT ? OFFSET ?`, interfaceID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying connection logs: %w", err)
	}
	defer rows.Close()

	logs, err := scanConnectionLogs(rows)
	return logs, total, err
}

func (s *SQLiteStore) ListPeerConnectionLogs(peerID string, limit, offset int) ([]models.ConnectionLog, int, error) {
	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM connection_logs WHERE peer_id = ?", peerID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting peer connection logs: %w", err)
	}

	rows, err := s.db.Query(`
		SELECT cl.id, cl.peer_id, COALESCE(p.name, ''), cl.interface_id, cl.event, cl.transfer_rx, cl.transfer_tx, cl.recorded_at
		FROM connection_logs cl
		LEFT JOIN peers p ON p.id = cl.peer_id
		WHERE cl.peer_id = ?
		ORDER BY cl.recorded_at DESC LIMIT ? OFFSET ?`, peerID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying peer connection logs: %w", err)
	}
	defer rows.Close()

	logs, err := scanConnectionLogs(rows)
	return logs, total, err
}

func (s *SQLiteStore) DeleteOldConnectionLogs(before time.Time) error {
	_, err := s.db.Exec("DELETE FROM connection_logs WHERE recorded_at < ?", before)
	if err != nil {
		return fmt.Errorf("deleting old connection logs: %w", err)
	}
	return nil
}

func scanConnectionLogs(rows interface{ Next() bool; Scan(...any) error; Err() error }) ([]models.ConnectionLog, error) {
	var result []models.ConnectionLog
	for rows.Next() {
		var log models.ConnectionLog
		if err := rows.Scan(&log.ID, &log.PeerID, &log.PeerName, &log.InterfaceID, &log.Event,
			&log.TransferRx, &log.TransferTx, &log.RecordedAt); err != nil {
			return nil, fmt.Errorf("scanning connection log: %w", err)
		}
		result = append(result, log)
	}
	return result, rows.Err()
}
