# Reproducer

[cznic/sqlite issue #214](https://gitlab.com/cznic/sqlite/-/issues/214)

The reproducer uses [`github.com/jmoiron/sqlx`](https://github.com/jmoiron/sqlx)
as a thin wrapper on top of `database/sql` to scan rows into struct fields.

```bash
go test .
# --- FAIL: TestDB (0.04s)
#     reproducer_test.go:62: cannot scan struct, reason: sql: Scan error on column index 1, name "modified": unsupported Scan, storing driver.Value type int64 into type *time.Time
# FAIL
# FAIL    example.org/reproducer  0.044s
# FAIL
```

For reference, the test passes when using `mattn/sqlite3`:

```bash
go test -tags mattn -v .
# === RUN   TestDB
# --- PASS: TestDB (0.02s)
# PASS
# ok      example.org/reproducer  0.024s
```
