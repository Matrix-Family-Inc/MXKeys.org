/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Fri 03 Apr 2026 UTC
 * Status: Created
 */

package raft

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"mxkeys/internal/zero/log"
	"mxkeys/internal/zero/nettls"
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

		n.wg.Add(1)
		go n.handleConnection(conn)
	}
}

// handleConnection handles an incoming connection.
func (n *Node) handleConnection(conn net.Conn) {
	defer n.wg.Done()
	defer conn.Close()

	msg, err := n.readRPC(conn)
	if err != nil {
		log.Debug("Failed to read RPC", "remote", conn.RemoteAddr(), "error", err)
		return
	}
	if err := n.verifyRPC(msg); err != nil {
		log.Debug("RPC verification failed", "remote", conn.RemoteAddr(), "error", err)
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

// sendRPC sends an RPC message to a peer.
//
// Equivalent to sendRPCCtx with a background context. Kept as the
// default call shape for election/replication paths that do not yet
// carry a caller context; every code path that has a context of its
// own (Propose, forwarded proposals, catch-up snapshots) should call
// sendRPCCtx directly so cancellation actually tears the in-flight
// RPC down instead of waiting out the TCP deadline.
func (n *Node) sendRPC(peer string, msgType MessageType, payload interface{}) (*RPCMessage, error) {
	return n.sendRPCCtx(context.Background(), peer, msgType, payload)
}

// sendRPCCtx is sendRPC with terminal ctx propagation.
//
// Cancellation contract:
//
//   - If ctx is already done when sendRPCCtx is entered, the call
//     returns ctx.Err() without touching the network.
//   - If ctx is cancelled mid-flight, the underlying connection is
//     closed so the outstanding write/read wakes up immediately
//     instead of blocking until nettls's default deadline fires.
//     The final error reported to the caller is ctx.Err() rather
//     than the resulting use-of-closed-connection noise.
//
// This is the contract Propose, handleForwardProposal, and
// driveInstallSnapshot depend on when they hand in a
// Cluster.proposeCtx / Node.ctxWithStop bound context: stopCh close
// must terminate every outstanding raft RPC on this path without
// waiting for the full CommitTimeout.
func (n *Node) sendRPCCtx(ctx context.Context, peer string, msgType MessageType, payload interface{}) (*RPCMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// DialContext honours ctx at both the TCP-connect and TLS-
	// handshake stages, so a caller-initiated cancel (e.g. stopCh
	// close via ctxWithStop) short-circuits the dial itself rather
	// than waiting out its 2-second hard cap.
	conn, err := nettls.DialContext(ctx, "tcp", peer, 2*time.Second, n.config.TLS)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Watcher: close the socket when ctx fires so the blocking
	// write/read unblocks. The stop channel makes the watcher exit
	// cleanly on the happy path, before sendRPCCtx returns.
	watcherDone := make(chan struct{})
	stop := make(chan struct{})
	go func() {
		defer close(watcherDone)
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-stop:
		}
	}()
	defer func() {
		close(stop)
		<-watcherDone
	}()

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
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}

	response, err := n.readRPC(conn)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}
	if err := n.verifyRPC(response); err != nil {
		return nil, err
	}
	return response, nil
}
