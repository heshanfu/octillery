package octillery

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"go.knocknote.io/octillery/connection"
	osql "go.knocknote.io/octillery/database/sql"
	"go.knocknote.io/octillery/path"
)

func init() {
	BeforeCommitCallback(func(tx *connection.TxConnection, writeQueries []*connection.QueryLog) error {
		log.Println("BeforeCommit", writeQueries)
		return nil
	})
	AfterCommitCallback(func(*connection.TxConnection) {
		log.Println("AfterCommit")
	}, func(tx *connection.TxConnection, isCriticalError bool, failureQueries []*connection.QueryLog) {
		log.Println("AfterCommit", failureQueries)
	})
}

func checkErr(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("%+v", err)
	}
}

func fetchUserName(multiRows []*sql.Rows) string {
	var name string
	for _, rows := range multiRows {
		for rows.Next() {
			rows.Scan(&name)
		}
	}
	return name
}

var db *osql.DB

func TestLoadConfig(t *testing.T) {
	confPath := filepath.Join(path.ThisDirPath(), "test_databases.yml")
	err := LoadConfig(confPath)
	checkErr(t, err)
	db, err = osql.Open("sqlite3", "dummy_dsn")
	checkErr(t, err)
}

func TestDropTableWithSequencerAndWithoutShardKey(t *testing.T) {
	_, _, err := Exec(db, "drop table if exists users")
	checkErr(t, err)
}

func TestCreateTableWithSequencerAndWithoutShardKey(t *testing.T) {
	createTable := "create table if not exists users (id integer not null primary key, name varchar(255))"
	_, _, err := Exec(db, createTable)
	checkErr(t, err)
}

func TestInsertWithSequencerAndWithoutShardKey(t *testing.T) {
	_, result, err := Exec(db, "insert into users(id, name) values (null, 'bob')")
	checkErr(t, err)
	id, err := result.LastInsertId()
	checkErr(t, err)
	multiRows, _, err := Exec(db, fmt.Sprintf("select name from users where id = %d", id))
	checkErr(t, err)
	name := fetchUserName(multiRows)
	if name != "bob" {
		t.Fatal(errors.Errorf("cannot select from id = %d", id))
	}
	_, result, err = Exec(db, "insert into users(id, name) values (null, 'ken')")
	checkErr(t, err)
	id, err = result.LastInsertId()
	checkErr(t, err)
	multiRows, _, err = Exec(db, fmt.Sprintf("select name from users where id = %d", id))
	checkErr(t, err)
	name = fetchUserName(multiRows)
	if name != "ken" {
		t.Fatal(errors.Errorf("cannot select from id = %d", id))
	}
}

func TestDropTableWithoutSequencer(t *testing.T) {
	_, _, err := Exec(db, "drop table if exists user_items")
	checkErr(t, err)
}

func TestCreateTableWithoutSequencer(t *testing.T) {
	createTable := "create table if not exists user_items (id integer not null primary key autoincrement, user_id integer not null)"
	_, _, err := Exec(db, createTable)
	checkErr(t, err)
}

func TestInsertWithoutSequencer(t *testing.T) {
	userID := 10
	insertQuery := fmt.Sprintf("insert into user_items(id, user_id) values (null, %d)", userID)
	_, result, err := Exec(db, insertQuery)
	checkErr(t, err)
	id, err := result.LastInsertId()
	checkErr(t, err)
	if id != 1 {
		t.Fatal(err)
	}
	_, result, err = Exec(db, insertQuery)
	checkErr(t, err)
	id, err = result.LastInsertId()
	checkErr(t, err)
	if id != 2 {
		t.Fatal(err)
	}
	multiRows, _, err := Exec(db, fmt.Sprintf("select user_id from user_items where user_id = %d", userID))
	checkErr(t, err)
	var rowCount int
	for _, rows := range multiRows {
		for rows.Next() {
			var fetchedID int
			rows.Scan(&fetchedID)
			rowCount++
			if fetchedID != userID {
				t.Fatal(errors.New("cannot fetch user_id from user_items"))
			}
		}
	}
	if rowCount != 2 {
		t.Fatal(errors.New("cannot select from user_items"))
	}
}

func TestDropTableWithSequencerAndShardKey(t *testing.T) {
	_, _, err := Exec(db, "drop table if exists user_decks")
	checkErr(t, err)
}

func TestCreateTableWithSequencerAndShardKey(t *testing.T) {
	createTable := "create table if not exists user_decks (id integer not null primary key autoincrement, user_id integer not null)"
	_, _, err := Exec(db, createTable)
	checkErr(t, err)
}

func TestInsertWithSequencerAndShardKey(t *testing.T) {
	userID := 10
	insertQuery := fmt.Sprintf("insert into user_decks(id, user_id) values (null, %d)", userID)
	_, result, err := Exec(db, insertQuery)
	checkErr(t, err)
	id, err := result.LastInsertId()
	checkErr(t, err)
	// id is generated by sequencer. first row's id is 2
	if id <= 1 {
		t.Fatal(errors.Errorf("id(%d) <= 1", id))
	}
	_, result, err = Exec(db, insertQuery)
	checkErr(t, err)
	id, err = result.LastInsertId()
	checkErr(t, err)
	if id <= 2 {
		t.Fatal(errors.Errorf("id(%d) <= 2", id))
	}
	multiRows, _, err := Exec(db, fmt.Sprintf("select user_id from user_decks where user_id = %d", userID))
	checkErr(t, err)
	var rowCount int
	for _, rows := range multiRows {
		for rows.Next() {
			var fetchedID int
			rows.Scan(&fetchedID)
			rowCount++
			if fetchedID != userID {
				t.Fatal(errors.New("cannot fetch user_id from user_decks"))
			}
		}
	}
	if rowCount != 2 {
		t.Fatal(errors.New("cannot select from user_decks"))
	}
}

func TestDropTableWithoutSharding(t *testing.T) {
	_, _, err := Exec(db, "drop table if exists user_stages")
	checkErr(t, err)
}

func TestCreateTableWithoutSharding(t *testing.T) {
	createTable := "create table if not exists user_stages (id integer not null primary key autoincrement, user_id integer not null)"
	_, _, err := Exec(db, createTable)
	checkErr(t, err)
}

func TestInsertWithoutSharding(t *testing.T) {
	userID := 10
	insertQuery := fmt.Sprintf("insert into user_stages(id, user_id) values (null, %d)", userID)
	_, result, err := Exec(db, insertQuery)
	checkErr(t, err)
	id, err := result.LastInsertId()
	checkErr(t, err)

	if id != 1 {
		t.Fatal(errors.Errorf("id(%d) != 1", id))
	}
	_, result, err = Exec(db, insertQuery)
	checkErr(t, err)
	id, err = result.LastInsertId()
	checkErr(t, err)
	if id != 2 {
		t.Fatal(errors.Errorf("id(%d) != 2", id))
	}
	multiRows, _, err := Exec(db, fmt.Sprintf("select user_id from user_stages where user_id = %d", userID))
	checkErr(t, err)
	var rowCount int
	for _, rows := range multiRows {
		for rows.Next() {
			var fetchedID int
			rows.Scan(&fetchedID)
			rowCount++
			if fetchedID != userID {
				t.Fatal(errors.New("cannot fetch user_id from user_stages"))
			}
		}
	}
	if rowCount != 2 {
		t.Fatal(errors.New("cannot select from user_stages"))
	}
}

func TestRollbackWithSequencerAndWithoutShardKey(t *testing.T) {
	db, err := osql.Open("mysql", "root:@tcp(127.0.0.1:3306)/?parseTime=true")
	defer db.Close()
	checkErr(t, err)
	tx, err := db.Begin()
	checkErr(t, err)
	result, err := tx.Exec("insert into users(id, name) values (null, 'alice')")
	checkErr(t, err)
	id, err := result.LastInsertId()
	checkErr(t, err)
	tx.Rollback()
	var name string
	err = db.QueryRow(fmt.Sprintf("select name from users where id = %d", id)).Scan(&name)
	if err == nil {
		t.Fatal(errors.New("cannot rollback"))
	}
}

func TestRollbackWithoutSharding(t *testing.T) {
	db, err := osql.Open("mysql", "root:@tcp(127.0.0.1:3306)/?parseTime=true")
	defer db.Close()
	checkErr(t, err)
	tx, err := db.Begin()
	checkErr(t, err)
	userID := 20
	result, err := tx.Exec(fmt.Sprintf("insert into user_stages(id, user_id) values (null, %d)", userID))
	checkErr(t, err)
	id, err := result.LastInsertId()
	checkErr(t, err)
	tx.Rollback()
	var fetchID interface{}
	err = db.QueryRow(fmt.Sprintf("select user_id from user_stages where id = %d", id)).Scan(&fetchID)
	if err == nil {
		t.Fatal(errors.New("cannot rollback"))
	}
}

func TestCommitWithoutSharding(t *testing.T) {
	db, err := osql.Open("mysql", "root:@tcp(127.0.0.1:3306)/?parseTime=true")
	defer db.Close()
	checkErr(t, err)
	tx, err := db.Begin()
	checkErr(t, err)
	userID := 20
	result, err := tx.Exec(fmt.Sprintf("insert into user_stages(id, user_id) values (null, %d)", userID))
	checkErr(t, err)
	id, err := result.LastInsertId()
	checkErr(t, err)
	checkErr(t, tx.Commit())
	var fetchID interface{}
	checkErr(t, db.QueryRow(fmt.Sprintf("select user_id from user_stages where id = %d", id)).Scan(&fetchID))
}

func TestPrepareWithoutSharding(t *testing.T) {
	db, err := osql.Open("mysql", "root:@tcp(127.0.0.1:3306)/?parseTime=true")
	defer db.Close()
	checkErr(t, err)
	stmt, err := db.Prepare("insert into user_stages(id, user_id) values (null, ?)")
	checkErr(t, err)
	userID := 30
	result, err := stmt.Exec(userID)
	checkErr(t, err)
	id, err := result.LastInsertId()
	checkErr(t, err)
	var fetchID int
	err = db.QueryRow(fmt.Sprintf("select user_id from user_stages where id = %d", id)).Scan(&fetchID)
	checkErr(t, err)
	if fetchID != userID {
		t.Fatal(errors.New("cannot get userID"))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stmt, err = db.PrepareContext(ctx, "insert into user_stages(id, user_id) values (null, ?)")
	checkErr(t, err)
	userID = 40
	result, err = stmt.Exec(userID)
	checkErr(t, err)
	id, err = result.LastInsertId()
	checkErr(t, err)
	err = db.QueryRow(fmt.Sprintf("select user_id from user_stages where id = %d", id)).Scan(&fetchID)
	checkErr(t, err)
	if fetchID != userID {
		t.Fatal(errors.New("cannot get userID"))
	}
}
