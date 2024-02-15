package packetize

import (
	"context"
	"encoding/binary"
	"errors"
	bufpool "github.com/valyala/bytebufferpool"
	"io"
	"net"
	"time"
)

type Conn struct {
	conn         net.Conn
	readDeadline <-chan time.Time
	packetChan   chan []byte

	closeCtx context.Context
	close    context.CancelFunc
}

func (conn *Conn) Read(b []byte) (n int, err error) {
	select {
	case packet := <-conn.packetChan:
		if len(b) < len(packet) {
			err = errors.New("message sent on a socket was larger than the buffer used to receive the message into")
		}
		return copy(b, packet), err
	case <-conn.closeCtx.Done():
		return 0, errors.New("error reading from conn: connection closed")
	case <-conn.readDeadline:
		return 0, errors.New("error reading from conn: read timeout")
	}
}

func (conn *Conn) Write(b []byte) (n int, err error) {
	if conn.closeCtx.Err() != nil {
		return 0, errors.New("error writing to conn: connection closed")
	}

	pk := bufpool.Get()
	defer bufpool.Put(pk)

	err = binary.Write(pk, binary.BigEndian, uint16(len(b)))
	if err != nil {
		return
	}
	_, err = pk.Write(b)
	if err != nil {
		return
	}

	_, err = conn.conn.Write(pk.B)
	return
}

func (conn *Conn) Close() error {
	if conn.closeCtx.Err() != nil {
		return errors.New("conn is already closed")
	}

	conn.close()
	return conn.conn.Close()
}

func (conn *Conn) LocalAddr() net.Addr {
	return conn.conn.LocalAddr()
}

func (conn *Conn) RemoteAddr() net.Addr {
	return conn.conn.RemoteAddr()
}

func (conn *Conn) SetDeadline(t time.Time) error {
	return conn.SetReadDeadline(t)
}

func (conn *Conn) SetReadDeadline(t time.Time) error {
	if t.IsZero() {
		conn.readDeadline = make(chan time.Time)
		return nil
	}
	if t.Before(time.Now()) {
		return errors.New("read deadline cannot be before now")
	}
	conn.readDeadline = time.After(t.Sub(time.Now()))
	return nil
}

func (conn *Conn) SetWriteDeadline(time.Time) error {
	return nil
}

func (conn *Conn) process() {
	lenBuf := make([]byte, 2)

	for conn.closeCtx.Err() == nil {
		length, err := io.ReadFull(conn.conn, lenBuf)
		if err != nil || length != 2 {
			_ = conn.Close()
			break
		}

		pkLen := int(binary.BigEndian.Uint16(lenBuf))

		pk := make([]byte, pkLen)

		length, err = io.ReadFull(conn.conn, pk)
		if err != nil || length != pkLen {
			_ = conn.Close()
			break
		}

		conn.packetChan <- pk
	}
}
