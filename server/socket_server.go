package server

import (
	"bufio"
	"fmt"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-wire"
	tmsp "github.com/tendermint/tmsp/types"
	"github.com/zballs/comit/types"
	"io"
	"net"
	"strings"
	"sync"
)

const (
	QUERY_CHAIN_ID byte = 0
	QUERY_SIZE     byte = 1
	QUERY_BY_KEY   byte = 2
	QUERY_BY_INDEX byte = 3
	QUERY_ISSUES   byte = 4
)

// var maxNumberConnections = 2

type SocketServer struct {
	*Hub
	QuitService

	proto    string
	addr     string
	listener net.Listener

	connsMtx   sync.Mutex
	conns      map[int]net.Conn
	nextConnID int

	appMtx sync.Mutex
	app    tmsp.Application
}

func NewSocketServer(protoAddr string, app tmsp.Application) (Service, error) {
	parts := strings.SplitN(protoAddr, "://", 2)
	proto, addr := parts[0], parts[1]
	s := &SocketServer{
		Hub:      NewHub(),
		proto:    proto,
		addr:     addr,
		listener: nil,
		app:      app,
		conns:    make(map[int]net.Conn),
	}
	s.QuitService = *NewQuitService(nil, "TMSPServer", s)
	_, err := s.Start() // Just start it
	return s, err
}

func (s *SocketServer) OnStart() error {
	s.QuitService.OnStart()
	ln, err := net.Listen(s.proto, s.addr)
	if err != nil {
		return err
	}
	s.listener = ln
	go s.acceptConnectionsRoutine()
	go s.Run() // run hub
	return nil
}

func (s *SocketServer) OnStop() {
	s.QuitService.OnStop()
	s.listener.Close()

	s.connsMtx.Lock()
	for id, conn := range s.conns {
		delete(s.conns, id)
		conn.Close()
	}
	s.connsMtx.Unlock()
}

func (s *SocketServer) addConn(conn net.Conn) int {
	s.connsMtx.Lock()
	defer s.connsMtx.Unlock()

	connID := s.nextConnID
	s.nextConnID += 1
	s.conns[connID] = conn

	return connID
}

// deletes conn even if close errs
func (s *SocketServer) rmConn(connID int, conn net.Conn) error {
	s.connsMtx.Lock()
	defer s.connsMtx.Unlock()

	delete(s.conns, connID)
	return conn.Close()
}

func (s *SocketServer) acceptConnectionsRoutine() {
	// semaphore := make(chan struct{}, maxNumberConnections)

	for {
		// semaphore <- struct{}{}

		// Accept a connection
		log.Notice("Waiting for new connection...")
		conn, err := s.listener.Accept()
		if err != nil {
			if !s.IsRunning() {
				return // Ignore error from listener closing.
			}
			Exit("Failed to accept connection: " + err.Error())
		} else {
			log.Notice("Accepted a new connection")
		}

		connID := s.addConn(conn)

		closeConn := make(chan error, 2)             // Push to signal connection closed
		responses := make(chan *tmsp.Response, 1000) // A channel to buffer responses

		// Read requests from conn and deal with them
		go s.handleRequests(closeConn, conn, responses)
		// Pull responses from 'responses' and write them to conn.
		go s.handleResponses(closeConn, responses, conn)

		go func() {
			// Wait until signal to close connection
			errClose := <-closeConn
			if errClose == io.EOF {
				log.Warn("Connection was closed by client")
			} else if errClose != nil {
				log.Warn("Connection error", "error", errClose)
			} else {
				// never happens
				log.Warn("Connection was closed.")
			}

			// Close the connection
			err := s.rmConn(connID, conn)
			if err != nil {
				log.Warn("Error in closing connection", "error", err)
			}

			// <-semaphore
		}()
	}
}

// Read requests from conn and deal with them
func (s *SocketServer) handleRequests(closeConn chan error, conn net.Conn, responses chan<- *tmsp.Response) {
	var count int
	var bufReader = bufio.NewReader(conn)
	for {
		var req = &tmsp.Request{}
		err := tmsp.ReadMessage(bufReader, req)
		if err != nil {
			if err == io.EOF {
				closeConn <- err
			} else {
				closeConn <- fmt.Errorf("Error reading message: %v", err.Error())
			}
			return
		}
		s.appMtx.Lock()
		count++
		s.handleRequest(req, responses)
		s.appMtx.Unlock()
	}
}

func (s *SocketServer) handleRequest(req *tmsp.Request, responses chan<- *tmsp.Response) {
	switch r := req.Value.(type) {
	case *tmsp.Request_Echo:
		responses <- tmsp.ToResponseEcho(r.Echo.Message)
	case *tmsp.Request_Flush:
		responses <- tmsp.ToResponseFlush()
	case *tmsp.Request_Info:
		data := s.app.Info()
		responses <- tmsp.ToResponseInfo(data)
	case *tmsp.Request_SetOption:
		so := r.SetOption
		logStr := s.app.SetOption(so.Key, so.Value)
		responses <- tmsp.ToResponseSetOption(logStr)
	case *tmsp.Request_AppendTx:
		res := s.app.AppendTx(r.AppendTx.Tx)
		responses <- tmsp.ToResponseAppendTx(res.Code, res.Data, res.Log)
	case *tmsp.Request_CheckTx:
		res := s.app.CheckTx(r.CheckTx.Tx)
		responses <- tmsp.ToResponseCheckTx(res.Code, res.Data, res.Log)
	case *tmsp.Request_Commit:
		res := s.app.Commit()
		responses <- tmsp.ToResponseCommit(res.Code, res.Data, res.Log)
	case *tmsp.Request_Query:
		res := s.app.Query(r.Query.Query)
		responses <- tmsp.ToResponseQuery(res.Code, res.Data, res.Log)
	case *tmsp.Request_InitChain:
		if app, ok := s.app.(tmsp.BlockchainAware); ok {
			app.InitChain(r.InitChain.Validators)
			responses <- tmsp.ToResponseInitChain()
		} else {
			responses <- tmsp.ToResponseInitChain()
		}
	case *tmsp.Request_EndBlock:
		if app, ok := s.app.(tmsp.BlockchainAware); ok {
			validators := app.EndBlock(r.EndBlock.Height)
			responses <- tmsp.ToResponseEndBlock(validators)
		} else {
			responses <- tmsp.ToResponseEndBlock(nil)
		}
	default:
		responses <- tmsp.ToResponseException("Unknown request")
	}
}

// Pull responses from 'responses' and write them to conn.
func (s *SocketServer) handleResponses(closeConn chan error, responses <-chan *tmsp.Response, conn net.Conn) {
	var count int
	var err error
	var bufWriter = bufio.NewWriter(conn)

FOR_LOOP:
	for {
		res := <-responses

		switch res.Value.(type) {

		case *tmsp.Response_AppendTx:

			resAppendTx := res.GetAppendTx()

			code := resAppendTx.Code
			data := resAppendTx.Data

			if code == tmsp.CodeType_OK && len(data) > 0 {

				switch data[0] {

				case types.SubmitTx:

					update, _, err := wire.GetByteSlice(data[1:])

					if err != nil {
						log.Error(err.Error())
						continue FOR_LOOP
					}

					s.SendUpdate(update)

				case types.ResolveTx:

					update, _, err := wire.GetByteSlice(data[1:])

					if err != nil {
						log.Error(err.Error())
						continue FOR_LOOP
					}

					s.SendUpdate(update)

				default:
				}
			}

		case *tmsp.Response_CheckTx:

			resCheckTx := res.GetCheckTx()

			code := resCheckTx.Code
			data := resCheckTx.Data

			if code == tmsp.CodeType_OK && len(data) > 0 {

				if data[0] == types.UpdateTx {

					addr, _, err := wire.GetByteSlice(data[1:])

					if err != nil {

						log.Error(err.Error())
						continue FOR_LOOP
					}

					cli := NewClient(conn, addr)
					s.Register(cli)
				}
			}

		default:
		}

		err = tmsp.WriteMessage(res, bufWriter)
		if err != nil {
			closeConn <- fmt.Errorf("Error writing message: %v", err.Error())
			return
		}
		if _, ok := res.Value.(*tmsp.Response_Flush); ok {
			err = bufWriter.Flush()
			if err != nil {
				closeConn <- fmt.Errorf("Error flushing write buffer: %v", err.Error())
				return
			}
		}
		count++
	}
}
