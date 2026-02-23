package postgres

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

type QuerySubscription struct {
	SubName string
}

func (db *DB) GetSubscriptions(database string) ([]QuerySubscription, error) {
	conn, err := pgx.Connect(context.Background(), db.getConnString(database))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	rows, err := conn.Query(context.Background(), `
		SELECT s.subname
		FROM pg_subscription s
		JOIN pg_database d ON s.subdbid = d.oid
		WHERE d.datname = $1`, database)
	if err != nil {
		return nil, fmt.Errorf("unable to get subscriptions: %w", err)
	}
	defer rows.Close()

	subscriptions, err := pgx.CollectRows(rows, pgx.RowToStructByName[QuerySubscription])
	if err != nil {
		return nil, fmt.Errorf("unable to extract subscriptions names from SQL query: %w", err)
	}

	return subscriptions, nil
}

func (db *DB) CreateSubscription(subscription string, from DB, publication string) error {
	conn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	// This is a hack due to testcontainers not properly supporting docker network, preventing us
	// to actually create subscription and have it work. The WITH (connect = false) is a workaround to avoid
	// the subscription to try to connect to the publisher when created.
	connectionString := from.getConnString(from.Database)
	if os.Getenv("TEST_ENV") == "true" {
		connectionString = fmt.Sprintf("postgres://%s:%s@publisher:5432/%s", from.Role, from.Password, from.Database)
	}
	query := fmt.Sprintf("CREATE SUBSCRIPTION %s CONNECTION '%s' PUBLICATION %s",
		subscription,
		connectionString,
		publication)

	_, err = conn.Exec(context.Background(), query)
	if err != nil {
		return fmt.Errorf("unable to create subscription: %w", err)
	}

	return nil
}

func (db *DB) DropSubscription(subscription string) error {
	conn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	_, err = conn.Exec(context.Background(), fmt.Sprintf("DROP SUBSCRIPTION %s", subscription))
	if err != nil { // Retry block with force drop if client cannot connect to publisher
		_, err2 := conn.Exec(context.Background(), fmt.Sprintf("ALTER SUBSCRIPTION %s DISABLE", subscription))
		if err2 != nil {
			return fmt.Errorf("unable to disable subscription: %w", err)
		}
		_, err2 = conn.Exec(context.Background(), fmt.Sprintf("ALTER SUBSCRIPTION %s SET (slot_name=NONE)", subscription))
		if err2 != nil {
			return fmt.Errorf("unable to remove slot_name from subscription: %w", err)
		}
		_, err2 = conn.Exec(context.Background(), fmt.Sprintf("DROP SUBSCRIPTION %s", subscription))
		if err2 == nil {
			return nil
		}
		return fmt.Errorf("unable to drop subscription: %w", err)
	}

	return nil
}

func (db *DB) GetSubscriptionLag(subscriptionName string) (int64, error) {
	conn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		return 0, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	var lag int64
	err = conn.QueryRow(context.Background(), `
		SELECT pg_wal_lsn_diff(pg_current_wal_lsn(), confirmed_flush_lsn)
		FROM pg_replication_slots
		WHERE slot_type = 'logical' AND slot_name = $1::text`, subscriptionName).Scan(&lag)
	if err != nil {
		return 0, fmt.Errorf("unable to get subscription lag: %w", err)
	}

	return lag, nil
}
