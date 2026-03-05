package main

import (
	"flag"
	"fmt"

	"github.com/prajwalx/pgsql-lintproxy/internal/proxy"
)

func main() {
	localPort := flag.String("p", "5433", "Local proxy port")                     // lint proxy port
	dbAddr := flag.String("db", "127.0.0.1:5432", "Destination Postgres address") // postgreSql address

	fmt.Println("Starting proxy")
	proxy.StartProxy(*localPort, *dbAddr)

}
