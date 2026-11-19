package backend

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

var radiusDB *sql.DB

func InitRadiusDB() {
	if radiusDB != nil {
		return
	}
	dsn := os.Getenv("RADIUS_DB_DSN") // e.g. "user:pass@tcp(127.0.0.1:3306)/radius"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic("radius db: " + err.Error())
	}
	radiusDB = db
}

// GrantAccess writes a time-limited user to FreeRADIUS.
// username = MAC address (lowercase), password = MAC, Session-Timeout = seconds.
func GrantAccess(mac string, durationMinutes int) error {
	InitRadiusDB()

	username := mac // MikroTik Hotspot MAC authentication defaults to MAC as username
	password := mac
	timeoutSeconds := durationMinutes * 60

	tx, err := radiusDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clean up old entries for the same MAC (avoid duplicate accumulation)
	for _, table := range []string{"radcheck", "radreply", "radusergroup"} {
		tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE username = ?", table), username)
	}

	// radcheck: password (Cleartext-Password, for PAP/CHAP)
	if _, err := tx.Exec(
		"INSERT INTO radcheck (username, attribute, op, value) VALUES (?, 'Cleartext-Password', ':=', ?)",
		username, password,
	); err != nil {
		return err
	}

	// radreply: session duration (MikroTik will disconnect when time is up)
	if _, err := tx.Exec(
		"INSERT INTO radreply (username, attribute, op, value) VALUES (?, 'Session-Timeout', ':=', ?)",
		username, fmt.Sprintf("%d", timeoutSeconds),
	); err != nil {
		return err
	}

	return tx.Commit()
}