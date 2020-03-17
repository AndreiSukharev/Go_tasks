// тут лежит тестовый код
// менять вам может потребоваться только коннект к базе
package main

import (
	main2 "coursera/db"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
)

var (
	// DSN это соединение с базой
	// вы можете изменить этот на тот который вам нужен
	// docker run -p 3306:3306 -v $(PWD):/docker-entrypoint-initdb.d -e MYSQL_ROOT_PASSWORD=1234 -e MYSQL_DATABASE=golang -d mysql
	DSN = "root:1234@tcp(localhost:3306)/golang?charset=utf8"
	// DSN = "coursera:5QPbAUufx7@tcp(localhost:3306)/coursera?charset=utf8"
)

func main() {
	db, err := sql.Open("mysql", DSN)
	err = db.Ping() // вот тут будет первое подключение к базе
	if err != nil {
		panic(err)
	}
	defer db.Close()
	handler, err := main2.NewDbExplorer(db)
	if err != nil {
		panic(err)
	}

	fmt.Println("starting server at :8082")
	http.ListenAndServe(":8082", handler)
}

