package server

import (
	"bufio"
	"fmt"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-wire"
	"github.com/tendermint/tmsp/types"
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

	CHECK_TX_CONNECT       = 5
	CHECK_TX_REGISTER byte = 6
	CHECK_TX_UPDATE   byte = 7
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
	app    types.Application
}

func NewSocketServer(protoAddr string, app types.Application) (Service, error) {
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

		closeConn := make(chan error, 2)              // Push to signal connection closed
		responses := make(chan *types.Response, 1000) // A channel to buffer responses

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
func (s *SocketServer) handleRequests(closeConn chan error, conn net.Conn, responses chan<- *types.Response) {
	var count int
	var bufReader = bufio.NewReader(conn)
	for {
		var req = &types.Request{}
		err := types.ReadMessage(bufReader, req)
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

func (s *SocketServer) handleRequest(req *types.Request, responses chan<- *types.Response) {
	switch r := req.Value.(type) {
	case *types.Request_Echo:
		responses <- types.ToResponseEcho(r.Echo.Message)
	case *types.Request_Flush:
		responses <- types.ToResponseFlush()
	case *types.Request_Info:
		data := s.app.Info()
		responses <- types.ToResponseInfo(data)
	case *types.Request_SetOption:
		so := r.SetOption
		logStr := s.app.SetOption(so.Key, so.Value)
		responses <- types.ToResponseSetOption(logStr)
	case *types.Request_AppendTx:
		res := s.app.AppendTx(r.AppendTx.Tx)
		responses <- types.ToResponseAppendTx(res.Code, res.Data, res.Log)
	case *types.Request_CheckTx:
		res := s.app.CheckTx(r.CheckTx.Tx)
		responses <- types.ToResponseCheckTx(res.Code, res.Data, res.Log)
	case *types.Request_Commit:
		res := s.app.Commit()
		responses <- types.ToResponseCommit(res.Code, res.Data, res.Log)
	case *types.Request_Query:
		res := s.app.Query(r.Query.Query)
		responses <- types.ToResponseQuery(res.Code, res.Data, res.Log)
	case *types.Request_InitChain:
		if app, ok := s.app.(types.BlockchainAware); ok {
			app.InitChain(r.InitChain.Validators)
			responses <- types.ToResponseInitChain()
		} else {
			responses <- types.ToResponseInitChain()
		}
	case *types.Request_EndBlock:
		if app, ok := s.app.(types.BlockchainAware); ok {
			validators := app.EndBlock(r.EndBlock.Height)
			responses <- types.ToResponseEndBlock(validators)
		} else {
			responses <- types.ToResponseEndBlock(nil)
		}
	default:
		responses <- types.ToResponseException("Unknown request")
	}
}

// Pull responses from 'responses' and write them to conn.
func (s *SocketServer) handleResponses(closeConn chan error, responses <-chan *types.Response, conn net.Conn) {
	var count int
	var err error
	var bufWriter = bufio.NewWriter(conn)
	for {
		res := <-responses

		switch res.Value.(type) {

		case *types.Response_AppendTx:

			resAppendTx := res.GetAppendTx()
			code := resAppendTx.Code
			data := resAppendTx.Data
			log_ := resAppendTx.Log

			if code == types.CodeType_OK && log_ == "form" {
				s.SendUpdate(data)
			}

		case *types.Response_CheckTx:

			resCheckTx := res.GetCheckTx()
			code := resCheckTx.Code
			data := resCheckTx.Data

			fmt.Printf("%v\n", resCheckTx)

			if code == types.CodeType_OK && len(data) > 0 {

				switch data[0] {

				case CHECK_TX_REGISTER:
					addr, _, err := wire.GetByteSlice(data[1:])
					if err != nil {
						log.Error(err.Error())
					}
					cli := NewClient(conn, addr)
					s.Register(cli)

				default:
				}
			}

		default:
		}

		err = types.WriteMessage(res, bufWriter)
		if err != nil {
			closeConn <- fmt.Errorf("Error writing message: %v", err.Error())
			return
		}
		if _, ok := res.Value.(*types.Response_Flush); ok {
			err = bufWriter.Flush()
			if err != nil {
				closeConn <- fmt.Errorf("Error flushing write buffer: %v", err.Error())
				return
			}
		}
		count++
	}
}
