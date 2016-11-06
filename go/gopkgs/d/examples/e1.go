package main

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
)

func main() {
	db, _ := sql.Open("test", "somesource")
	stmt, _ := db.Prepare("SELECT")
	if rows, err := stmt.Query(); err != nil {
		fmt.Println("-------- rows:", rows, "err:", err)
	} else {
		for rows.Next() {
			var id int64
			rows.Scan(&id)
			fmt.Println("scanned id:", id)
		}
	}
}

func init() {
	sql.Register("test", &testDriver{})
}

// ********************* Driver ***********************
type testDriver struct{}

func (d *testDriver) Open(name string) (conn driver.Conn, err error) {
	conn = &testConn{}
	return
}

// ********************* Connection ***********************
type testConn struct{}

func (c *testConn) Prepare(query string) (stmt driver.Stmt, err error) {
	stmt = &testStmt{}
	return
}

func (c *testConn) Close() (err error) {
	return
}

func (c *testConn) Begin() (tx driver.Tx, err error) {
	return
}

// ********************* Statement ***********************
type testStmt struct{}

func (st *testStmt) Exec(args []driver.Value) (res driver.Result, err error) {
	res = &testResult{}
	return
}

func (st *testStmt) Query(args []driver.Value) (rows driver.Rows, err error) {
	rows = &testRows{}
	return
}

func (st *testStmt) NumInput() (num int) {
	return
}

func (st *testStmt) Close() (err error) {
	return
}

// ********************* Transaction ***********************
type testTx struct{}

func (tx *testTx) Commit() (err error) {
	return
}

func (tx *testTx) Rollback() (err error) {
	return
}

// ********************* Result ***********************
type testResult struct{}

func (res *testResult) LastInsertId() (id int64, err error) {
	return
}

func (res *testResult) RowsAffected() (cnt int64, err error) {
	return
}

// ********************* Rows ***********************
type testRows struct {
	count int
}

func (rows *testRows) Columns() (columns []string) {
	columns = []string{"id"}
	return
}

func (rows *testRows) Close() (err error) {
	return
}

func (rows *testRows) Next(dest []driver.Value) (err error) {
	if rows.count >= 10 {
		err = errors.New("run out of index")
	}
	rows.count++

	dest[0] = rows.count

	return
}
