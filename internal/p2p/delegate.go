// Copyright 2026 Georg Hagn (tiny-frameworks)
// SPDX-License-Identifier: Apache-2.0

package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"codeberg.org/tiny-frameworks/nexfact/api/writer"
	"codeberg.org/tiny-frameworks/nexfact/orchestrator"
	"codeberg.org/tiny-frameworks/nexutils/errors"
	"codeberg.org/tiny-frameworks/nexutils/p2p/rpc"
)

// ServerDelegate controls the RPC methods of nexfact-server
type ServerDelegate struct {
	rpc.DefaultNexDelegate
	log *slog.Logger
	o   *orchestrator.Orchestrator
}

// NewDelegate creates the RPC node with the ServerDelegate and injects the UserAuthenticator.
func NewDelegate(orch *orchestrator.Orchestrator, addr string, heartbeat time.Duration, auth rpc.UserAuthenticator, lgr *slog.Logger) *rpc.Node {

	delegate := &ServerDelegate{
		log: lgr,
		o:   orch,
	}

	// Set the provided authenticator (or fall back to dummy)
	delegate.SetAuthenticator(auth)

	return rpc.NewNode(rpc.Options{
		Addr:              addr,
		HeartbeatInterval: heartbeat,
		Delegate:          delegate,
	})
}

func (d *ServerDelegate) OnRequest(ctx context.Context, peer *rpc.Peer, method string, params []byte) (any, error) {
	switch method {
	case "nexfact.process":
		// Optional safeguard: Check if the peer is authorized
		if !peer.IsAuthorized() {
			nexErr := errors.New(
				errors.UnauthorizedError,
				"forbidden, not authorizes",
				"nexfact-server.internal.p2p.delegate",
			)
			return nil, rpc.NewRPCErrorFromNexError(rpc.InvalidRequest, nexErr)
		}
		return d.privateProcess(peer, params)

	case "system.echo":
		return d.privateEcho(peer, params)

	default:
		// "auth" und unknowm methods fallback to DefaultNexDelegate!
		return d.DefaultNexDelegate.OnRequest(ctx, peer, method, params)
	}
}

func (d *ServerDelegate) privateEcho(peer *rpc.Peer, params []byte) (any, error) {
	var echo string

	if err := json.Unmarshal(params, &echo); err != nil {
		nexErr := errors.Wrap(
			errors.InvalidFormat,
			"invalid params",
			"nexfact-server.internal.p2p.delegate",
			err,
		)
		return nil, rpc.NewRPCErrorFromNexError(rpc.InvalidParams, nexErr)
	}
	d.log.Info("[Echo] Received echo request from partner", "params", params, "peer", peer.RemoteAddr())
	return "Echo: " + echo, nil
}

func (d *ServerDelegate) privateProcess(peer *rpc.Peer, params []byte) (any, error) {
	var jsonInvoice string

	if err := json.Unmarshal(params, &jsonInvoice); err != nil {
		nerr := errors.Wrap(errors.InvalidValue,
			"nexfact.process: invalid parameters",
			"delegate.privateProcess", err)
		errors.LogError(d.log, nerr)
		return nil, rpc.NewRPCErrorFromNexError(rpc.InvalidParams, nerr)
	}
	d.log.Info("nexfact.process", "peer", peer.RemoteAddr())

	// 1. Create staging directory & handle errors
	tempDir, err := os.MkdirTemp("", "nexfact_stage_*")
	if err != nil {
		nerr := errors.Wrap(errors.ExecutionError,
			"failed to create temporary staging directory",
			"delegate.privateProcess", err)
		errors.LogError(d.log, nerr)
		return nil, rpc.NewRPCErrorFromNexError(rpc.InternalError, nerr)
	}
	defer os.RemoveAll(tempDir)

	// 2. Preprocessing with type-safe errors.As Conversion
	oJSON, err := PreProcess(tempDir, []byte(jsonInvoice))
	if err != nil {
		var nerr *errors.Error
		if !errors.As(err, &nerr) {
			nerr = errors.Wrap(errors.ExecutionError,
				"preprocessing failed",
				"delegate.privateProcess", err)
		}
		errors.LogError(d.log, nerr)
		return nil, rpc.NewRPCErrorFromNexError(rpc.InternalError, nerr)
	}

	// 3. Orchestrator execution
	out, err := d.o.RunJsonJob(oJSON)
	if err != nil {
		var nerr *errors.Error
		if !errors.As(err, &nerr) {
			nerr = errors.Wrap(errors.ExecutionError,
				"orchestrator job execution failed",
				"delegate.privateProcess", err)
		}
		errors.LogError(d.log, nerr)
		return nil, rpc.NewRPCErrorFromNexError(rpc.InternalError, nerr)
	}

	// 4. Read output file (server-side error -> InternalError)
	outBytes, err := os.ReadFile(out.OutputPath)
	if err != nil {
		nerr := errors.Wrap(errors.ReadError,
			fmt.Sprintf("cannot read output file: %s", out.OutputPath),
			"delegate.privateProcess", err)
		errors.LogError(d.log, nerr)
		return nil, rpc.NewRPCErrorFromNexError(rpc.InternalError, nerr)
	}

	mimeType := http.DetectContentType(outBytes)
	return writer.JobRPCResponse{
		JobID: out.JobID,
		OutputFile: writer.OutputFileType{
			Name:     filepath.Base(out.OutputPath),
			MimeType: mimeType,
			Size:     int64(len(outBytes)),
			Data:     outBytes,
		},
	}, nil
}
