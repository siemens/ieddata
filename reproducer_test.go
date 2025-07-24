package reproducer

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	testDatabaseFile     = "test.db"
	testTimestampSeconds = 1752778638
)

type Item struct {
	Name     string    `db:"name"`
	Modified time.Time `db:"modified"`
}

func TestDB(t *testing.T) {
	_ = os.Remove(testDatabaseFile)
	db, err := sqlx.Open(DriverName, testDatabaseFile+"?_inttotime=true")
	if err != nil {
		t.Fatalf("cannot open database using driver %s, reason: %s", DriverName, err.Error())
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("cannot close database, reason: %s", err.Error())
		}
	}()

	createstmnt := `CREATE TABLE "test" (
		"name" 		varchar(32) NOT NULL,
		"modified"	timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY("name")
	)`
	if _, err := db.Exec(createstmnt); err != nil {
		t.Fatalf("cannot create table, reason: %s", err.Error())
	}

	insertstmnt := `INSERT INTO "test" (name, modified) VALUES ("foobar", ` + strconv.FormatInt(testTimestampSeconds, 10) + `)`
	if _, err := db.Exec(insertstmnt); err != nil {
		t.Fatalf("cannot insert row, reason: %s", err.Error())
	}

	items := []Item{}
	rows, err := db.Queryx(`SELECT * FROM "test"`)
	if err != nil {
		t.Fatalf("cannot select, reason: %s", err.Error())
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Fatalf("cannot close rows, reason: %s", err.Error())
		}
	}()
	for rows.Next() {
		var item Item
		if err := rows.StructScan(&item); err != nil {
			t.Fatalf("cannot scan struct, reason: %s", err.Error())
		}
		items = append(items, item)
	}
	if len(items) != 1 {
		t.Fatalf("expected exactly one element, got %d elements", len(items))
	}

	testTime := time.Unix(testTimestampSeconds, 0).UTC()
	if !items[0].Modified.Equal(testTime) {
		t.Fatalf("expected %s, but got %s", testTime, items[0].Modified)
	}
}
