package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"net/http"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "example_password"
	dbname   = "example_name"
)

func main() {
	db := connectToDb()

	pems := getPems(db, 1, 5000)
	ch := createChan(pems)

	mux := http.NewServeMux()
	mux.Handle("/get_cert", createHandler(&ch))

	http.ListenAndServe("http://example.url:8115", mux)
}

//создаём Хэндлер, который возвращает по сертификату за запрос
func createHandler(ch *chan string) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(NextInChan(*ch)))
	}
	return http.HandlerFunc(fn)
}

//создаём коннект к БД
func connectToDb() *sql.DB {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, _ := sql.Open("postgres", psqlInfo)

	defer db.Close()

	err := db.Ping()
	if err != nil {
		fmt.Println(err)
	}

	return db
}

//получаем данные сертификатов из БД
func getPems(db *sql.DB, caId int, quantity int) []string {
	rows, err := db.Query("SELECT crt_id,cert_serialno,cert_pem_data FROM certificates WHERE ca_id = $1 "+
		"AND revokation_date IS NULL AND cert_not_after>current_TIMESTAMP"+
		"ORDER BY ca_id ASC LIMIT $2", caId, quantity)
	if err != nil {
		fmt.Println(err)
	}
	defer rows.Close()

	strings := make([]string, quantity, quantity)

	for rows.Next() {
		var pem string
		err = rows.Scan(&pem)
		strings = append(strings, pem)
	}
	return strings
}

//создаём канал из данных, полученных из БД
func createChan(pems []string) chan string {
	ch := make(chan string, 10000)
	go func() {
		defer close(ch)
		i := 0
		for {
			ch <- pems[i]
			if i+1 < len(pems) {
				i++
				continue
			}
			i = 0
		}
	}()
	return ch
}

func NextInChan(ch chan string) string {
	return <-ch
}
