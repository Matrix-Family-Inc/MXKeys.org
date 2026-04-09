/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 03 Apr 2026 UTC
 * Status: Created
 */

package raft

import (
	"encoding/json"
	"net"
	"time"
)

// acceptLoop accepts incoming connections.
func (n *Node) acceptLoop() {
	defer n.wg.Done()

	for {
		select {
		case <-n.stopCh:
			return
		default:
		}

		conn, err := n.listener.Accept()
		if err != nil {
			select {
			case <-n.stopCh:
				return
			default:
				continue
			}
		}

		go n.handleConnection(conn)
	}
}

// handleConnection handles an incoming connection.
func (n *Node) handleConnection(conn net.Conn) {
	defer conn.Close()

	msg, err := n.readRPC(conn)
	if err != nil {
		return
	}
	if err := n.verifyRPC(msg); err != nil {
		return
	}

	response := n.handleRPC(msg)
	if response == nil {
		return
	}
	if err := n.signRPC(response); err != nil {
		return
	}
	_ = n.writeRPC(conn, response)
}

// connectPeers connects to peer nodes.
func (n *Node) connectPeers() {
	defer n.wg.Done()

	for _, peer := range n.config.Peers {
		conn, err := net.DialTimeout("tcp", peer, 5*time.Second)
		if err != nil {
			continue
		}

		n.mu.Lock()
		n.peers[peer] = conn
		n.mu.Unlock()
	}
}

// sendRPC sends an RPC message to a peer.
func (n *Node) sendRPC(peer string, msgType MessageType, payload interface{}) (*RPCMessage, error) {
	conn, err := net.DialTimeout("tcp", peer, 2*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	msg := RPCMessage{
		Type:      msgType,
		From:      n.config.NodeID,
		Timestamp: time.Now().UTC(),
		Payload:   payloadBytes,
	}
	if err := n.signRPC(&msg); err != nil {
		return nil, err
	}

	if err := n.writeRPC(conn, &msg); err != nil {
		return nil, err
	}

	response, err := n.readRPC(conn)
	if err != nil {
		return nil, err
	}
	if err := n.verifyRPC(response); err != nil {
		return nil, err
	}

	return response, nil
}
