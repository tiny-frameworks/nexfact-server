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

type ServerDelegate struct {
	rpc.DefaultNexDelegate
	log *slog.Logger
	o   *orchestrator.Orchestrator
}

func NewDelegate(orch *orchestrator.Orchestrator, addr string, heartbeat time.Duration, lgr *slog.Logger) *rpc.Node {

	oNode := rpc.NewNode(rpc.Options{
		Addr:              addr,
		HeartbeatInterval: heartbeat,
		Delegate: &ServerDelegate{
			log: lgr,
			o:   orch,
		},
	})

	return oNode
}

func (d *ServerDelegate) OnRequest(ctx context.Context, peer *rpc.Peer, method string, params []byte) (any, error) {
	switch method {
	case "nexfact.process":
		return d.privateProcess(peer, params)

	case "nexfact.echo":
		return d.privateEcho(peer, params)

	default:
		return d.DefaultNexDelegate.OnRequest(ctx, peer, method, params)
	}
}

func (d *ServerDelegate) privateEcho(peer *rpc.Peer, params []byte) (any, error) {
	var echo string

	if err := json.Unmarshal(params, &echo); err != nil {
		return nil, &rpc.JsonRPCerror{Code: rpc.InvalidParams, Message: rpc.StdError[rpc.InvalidParams]}
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

	// 1. Staging Directory anlegen & Fehler abfangen
	tempDir, err := os.MkdirTemp("", "nexfact_stage_*")
	if err != nil {
		nerr := errors.Wrap(errors.ExecutionError,
			"failed to create temporary staging directory",
			"delegate.privateProcess", err)
		errors.LogError(d.log, nerr)
		return nil, rpc.NewRPCErrorFromNexError(rpc.InternalError, nerr)
	}
	defer os.RemoveAll(tempDir)

	// 2. Preprocessing mit typsicherer errors.As Conversion
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

	// 3. Orchestrator-Ausführung
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

	// 4. Ausgabedatei einlesen (Serverseitiger Fehler -> InternalError)
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
