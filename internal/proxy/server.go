package proxy

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/prajwalx/pgsql-lintproxy/internal/linter"
)

func StartProxy(localPort, dbAddr string) {
	listener, err := net.Listen("tcp", ":"+localPort)
	if err != nil {
		log.Fatalf("Failed to start tcp server %v", err)
	}
	log.Printf("Pgsql-LintProxy active: localhost:%s -> %s", localPort, dbAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleSession(conn, dbAddr)
	}
}

func handleSession(clientConn net.Conn, dbAddr string) {
	fmt.Println("Handle Session..")
	defer clientConn.Close()

	dbConn, err := net.Dial("tcp", dbAddr)

	if err != nil {
		log.Printf("Could not connect to backend DB: %v", err)
		return
	}
	defer dbConn.Close()
	fmt.Println("tcp dial to db success")

	// 1. Initialize the roles
	// We act as the "Backend" to the user's CLI
	backend := pgproto3.NewBackend(clientConn, clientConn)

	// We act as the "Frontend" to the real Database
	frontend := pgproto3.NewFrontend(dbConn, dbConn)

	// 2. Handle the Startup Handshake (Crucial!)
	// When a client connects, they send a StartupMessage.
	// We must forward this to the DB to establish the session.
	startupMsg, err := backend.ReceiveStartupMessage()
	if err != nil {
		return
	}
	fmt.Println("Sending startup msg", startupMsg)
	frontend.Send(startupMsg)
	if err := frontend.Flush(); err != nil {
		log.Printf("Flush error: %v", err)
		return
	}

	// 3. The Bi-Directional Pump
	// Goroutine: Stream EVERYTHING from DB -> Client (Results, Errors, Data)
	// This MUST be a goroutine so it doesn't block the loop below.
	// It sends DataRows, Auth requests, and ReadyForQuery signals back to TablePlus.
	go func() {
		_, err := io.Copy(clientConn, dbConn)
		if err != nil {
			return
		}
	}()

	// Main Loop: Stream Client -> DB (Intercepting Queries)
	for {
		// Read message from the Client
		msg, err := backend.Receive()
		fmt.Println("read msg from backend", msg)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("Receive error: %v", err)
			}
			break
		}

		// LOGIC INTERCEPTION: Simple Query
		if q, ok := msg.(*pgproto3.Query); ok {
			if err := linter.ValidateSQL(q.String); err != nil {
				fmt.Println(err)
				sendErrorToClient(clientConn, err)
				continue // Drop the message, don't forward to DB
			}
		}

		// Forward the encoded message to the real DB
		frontend.Send(msg)

		if err := frontend.Flush(); err != nil {
			log.Printf("Flush error: %v", err)
			return
		}

	}

}

func sendErrorToClient(conn net.Conn, lintErr error) {
	errResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		Code:     "42501", // Insufficient Privilege code
		Message:  lintErr.Error(),
	}
	res, err := errResp.Encode(nil)
	if err != nil {
		log.Fatalf("err in encoding %v", err)
	}
	conn.Write(res)

	// 2. Create the ReadyForQuery Message (This is the 'Unblocker')
	// 'I' means the backend is currently Idle (not in a transaction block)
	readyMsg := &pgproto3.ReadyForQuery{TxStatus: 'I'}
	res, err = readyMsg.Encode(nil)
	if err != nil {
		log.Fatalf("err in encoding %v", err)
	}
	conn.Write(res)
}
