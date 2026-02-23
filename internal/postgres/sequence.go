package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Sequence struct {
	SchemaName   string
	SequenceName string
	LastValue    int64
	FullName     string
}

func (db *DB) ListSequences() ([]Sequence, error) {
	conn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	rows, err := conn.Query(context.Background(), `
		SELECT schemaname, sequencename, last_value
		FROM pg_sequences
		WHERE last_value IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("unable to get sequences from database: %w", err)
	}
	defer rows.Close()

	var sequences []Sequence
	for rows.Next() {
		var seq Sequence
		err := rows.Scan(&seq.SchemaName, &seq.SequenceName, &seq.LastValue)
		if err != nil {
			return nil, fmt.Errorf("unable to scan sequence data: %w", err)
		}
		seq.FullName = fmt.Sprintf("%s.%s", seq.SchemaName, seq.SequenceName)
		sequences = append(sequences, seq)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over sequences: %w", err)
	}

	return sequences, nil
}

func (db *DB) CopySequences(targetDB DB) error {
	sourceConn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		return fmt.Errorf("unable to connect to source database: %w", err)
	}
	defer sourceConn.Close(context.Background()) //nolint: errcheck

	targetConn, err := pgx.Connect(context.Background(), targetDB.getConnString(targetDB.Database))
	if err != nil {
		return fmt.Errorf("unable to connect to target database %s: %w", targetDB.Database, err)
	}
	defer targetConn.Close(context.Background()) //nolint: errcheck

	sourceSequences, err := db.ListSequences()
	if err != nil {
		return fmt.Errorf("unable to get sequences from source database: %w", err)
	}

	sequenceCount := len(sourceSequences)
	if sequenceCount == 0 {
		return nil // No sequences found in source database
	}

	tx, err := targetConn.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(context.Background())
		} else {
			_ = tx.Commit(context.Background())
		}
	}()

	for _, sourceSequence := range sourceSequences {
		sequenceFullName := fmt.Sprintf("%s.%s", sourceSequence.SchemaName, sourceSequence.SequenceName)
		fmt.Printf("| Copying sequence %s (last_value: %d) to %s database %s\n",
			sequenceFullName, sourceSequence.LastValue, targetDB.Host, targetDB.Database)

		_, err = tx.Exec(context.Background(),
			fmt.Sprintf("CREATE SEQUENCE IF NOT EXISTS %s ", sequenceFullName))
		if err != nil {
			return fmt.Errorf("unable to create sequence %s: %w", sequenceFullName, err)
		}

		_, err = tx.Exec(context.Background(),
			fmt.Sprintf("SELECT setval('%s', %d)", sequenceFullName, sourceSequence.LastValue))
		if err != nil {
			return fmt.Errorf("unable to copy sequence %s: %w", sequenceFullName, err)
		}
	}

	return nil
}
