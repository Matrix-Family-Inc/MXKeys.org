/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package raft

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	raftMessageReadTimeout  = 5 * time.Second
	raftMessageWriteTimeout = 5 * time.Second
	maxRaftMessageSize      = 1 << 20
	maxRPCSkew              = 5 * time.Minute
)

func (n *Node) readRPC(conn net.Conn) (*RPCMessage, error) {
	if err := conn.SetReadDeadline(time.Now().Add(raftMessageReadTimeout)); err != nil {
		return nil, err
	}
	payload, err := readBoundedRPC(conn, maxRaftMessageSize)
	if err != nil {
		return nil, err
	}
	var msg RPCMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (n *Node) writeRPC(conn net.Conn, msg *RPCMessage) error {
	if msg == nil {
		return nil
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if len(payload) > maxRaftMessageSize {
		return fmt.Errorf("raft message exceeds max size")
	}
	if err := conn.SetWriteDeadline(time.Now().Add(raftMessageWriteTimeout)); err != nil {
		return err
	}
	_, err = conn.Write(append(payload, '\n'))
	return err
}

func readBoundedRPC(r io.Reader, maxBytes int) ([]byte, error) {
	reader := bufio.NewReaderSize(r, maxBytes+1)
	payload, err := reader.ReadSlice('\n')
	switch {
	case err == nil:
	case err == bufio.ErrBufferFull:
		return nil, fmt.Errorf("raft message exceeds max size of %d bytes", maxBytes)
	case err == io.EOF && len(payload) > 0:
	default:
		return nil, err
	}

	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 {
		return nil, io.EOF
	}
	if len(payload) > maxBytes {
		return nil, fmt.Errorf("raft message exceeds max size of %d bytes", maxBytes)
	}
	return payload, nil
}
