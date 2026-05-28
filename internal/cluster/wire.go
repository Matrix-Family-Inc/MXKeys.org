/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package cluster

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
	clusterMessageReadTimeout  = 5 * time.Second
	clusterMessageWriteTimeout = 5 * time.Second
	maxClusterMessageSize      = 1 << 20
)

func (c *Cluster) readMessage(conn net.Conn) (*ClusterMessage, error) {
	if err := conn.SetReadDeadline(time.Now().Add(clusterMessageReadTimeout)); err != nil {
		return nil, err
	}
	payload, err := readBoundedJSON(conn, maxClusterMessageSize)
	if err != nil {
		return nil, err
	}
	var msg ClusterMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (c *Cluster) writeMessage(conn net.Conn, msg *ClusterMessage) error {
	if msg == nil {
		return nil
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if len(payload) > maxClusterMessageSize {
		return fmt.Errorf("cluster message exceeds max size")
	}
	if err := conn.SetWriteDeadline(time.Now().Add(clusterMessageWriteTimeout)); err != nil {
		return err
	}
	_, err = conn.Write(append(payload, '\n'))
	return err
}

func readBoundedJSON(r io.Reader, maxBytes int) ([]byte, error) {
	reader := bufio.NewReaderSize(r, maxBytes+1)
	payload, err := reader.ReadSlice('\n')
	switch {
	case err == nil:
	case err == bufio.ErrBufferFull:
		return nil, fmt.Errorf("message exceeds max size of %d bytes", maxBytes)
	case err == io.EOF && len(payload) > 0:
	default:
		return nil, err
	}

	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 {
		return nil, io.EOF
	}
	if len(payload) > maxBytes {
		return nil, fmt.Errorf("message exceeds max size of %d bytes", maxBytes)
	}
	return payload, nil
}
