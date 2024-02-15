package packetize

import (
	"context"
	"errors"
	"net"
	"time"
)

type Listener struct {
	listener net.Listener
	connChan chan *Conn

	closeCtx context.Context
	close    context.CancelFunc
}

func Listen(network string, addr string) (*Listener, error) {
	listener, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}

	closeCtx, closeFunc := context.WithCancel(context.Background())
	lst := &Listener{
		listener: listener,
		connChan: make(chan *Conn),
		closeCtx: closeCtx,
		close:    closeFunc,
	}

	go func() {
		for lst.closeCtx.Err() == nil {
			conn, err := lst.listener.Accept()
			if err != nil {
				continue
			}

			cloneCtx, closeFunc := context.WithCancel(context.Background())
			pkConn := &Conn{
				conn:         conn,
				readDeadline: make(chan time.Time),
				packetChan:   make(chan []byte, 32),

				closeCtx: cloneCtx,
				close:    closeFunc,
			}

			go pkConn.process()

			lst.connChan <- pkConn
		}
	}()
	return lst, nil
}

func (lst *Listener) Accept() (net.Conn, error) {
	select {
	case conn := <-lst.connChan:
		return conn, nil
	case <-lst.closeCtx.Done():
		return nil, errors.New("accept: listener closed")
	}
}

func (lst *Listener) Close() error {
	lst.close()
	return lst.listener.Close()
}

func (lst *Listener) Addr() net.Addr {
	return lst.listener.Addr()
}
