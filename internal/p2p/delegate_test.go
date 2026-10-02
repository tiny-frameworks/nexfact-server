// Copyright 2026 Georg Hagn (tiny-frameworks)
// SPDX-License-Identifier: Apache-2.0

package p2p_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"codeberg.org/tiny-frameworks/nexfact-server/internal/p2p"
	"codeberg.org/tiny-frameworks/nexfact-server/internal/testsetup"

	"codeberg.org/tiny-frameworks/nexfact/api/writer"
	"codeberg.org/tiny-frameworks/nexfact/orchestrator"
	"codeberg.org/tiny-frameworks/nexutils/logger"
	"codeberg.org/tiny-frameworks/nexutils/p2p/rpc"
)

func TestOrchestrator_Service_Documentation(t *testing.T) {

	// =========================================================================
	// DOKU 1: Explicit "Manual" Client
	// =========================================================================
	t.Run("Demo_Manual_Peer_Connection", func(t *testing.T) {
		envRoot, resourcePath := testsetup.SetupTestEnv(t, testsetup.NATIVE)
		freeAddr := testsetup.GetFreeAddr(t)
		lgr := logger.Logger.With("test_name", t.Name())

		orch, err := orchestrator.New(envRoot)
		if err != nil {
			t.Fatalf("failed to create orchestrator: %v", err)
		}

		userAuth := p2p.NewNexfactUserStore()
		orchNode := p2p.NewDelegate(orch, freeAddr, 1*time.Second, userAuth, lgr)

		go func() { _ = orchNode.Start() }()
		defer orchNode.Stop()

		time.Sleep(50 * time.Millisecond)

		clientNode := rpc.NewNode(rpc.Options{Addr: freeAddr})
		clientPeer, err := clientNode.ConnectToPeer("ws://" + freeAddr + "/ws")
		if err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		defer clientPeer.Close()

		time.Sleep(100 * time.Millisecond)

		// Auth Call...
		callCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		var authRes rpc.AuthResult
		_ = clientPeer.Call(callCtx, "system.auth", rpc.AuthParams{
			Username: "admin",
			Password: "nexfact-secure-2026",
		}, &authRes)
		cancel()

		t.Run("Echo_Service", func(t *testing.T) { runOrchestrator_Echo_ServiceTest(t, clientPeer) })
		t.Run("FacturX_Service", func(t *testing.T) { runOrchestrator_FacturX_ServiceTest(t, clientPeer, resourcePath) })
		t.Run("CombineService", func(t *testing.T) { runOrchestrator_Combine_ServiceTest(t, clientPeer, resourcePath) })
	})

	// =========================================================================
	// DOKU 2: Autonomous "Managed" Client
	// =========================================================================
	t.Run("Demo_Managed_Client_Connection", func(t *testing.T) {
		// create FRESCH Environment for second loop!
		envRoot, resourcePath := testsetup.SetupTestEnv(t, testsetup.NATIVE)
		freeAddr := testsetup.GetFreeAddr(t)
		lgr := logger.Logger.With("test_name", t.Name())

		orch, err := orchestrator.New(envRoot)
		if err != nil {
			t.Fatalf("failed to create orchestrator: %v", err)
		}

		userAuth := p2p.NewNexfactUserStore()
		orchNode := p2p.NewDelegate(orch, freeAddr, 1*time.Second, userAuth, lgr)

		go func() { _ = orchNode.Start() }()
		defer orchNode.Stop()

		time.Sleep(50 * time.Millisecond)

		clientNode := rpc.NewNode(rpc.Options{Addr: freeAddr})
		managedClient := clientNode.ConnectWithAutoReconnect("ws://"+freeAddr+"/ws", rpc.ReconnectConfig{})
		managedClient.SetCredentials("admin", "nexfact-secure-2026")
		defer managedClient.Close()

		time.Sleep(100 * time.Millisecond)

		peer := managedClient.GetPeer()

		t.Run("Echo_Service", func(t *testing.T) { runOrchestrator_Echo_ServiceTest(t, peer) })
		t.Run("FacturX_Service", func(t *testing.T) { runOrchestrator_FacturX_ServiceTest(t, peer, resourcePath) })
		t.Run("CombineService", func(t *testing.T) { runOrchestrator_Combine_ServiceTest(t, peer, resourcePath) })
	})
}

func runOrchestrator_Echo_ServiceTest(t *testing.T, clientPeer *rpc.Peer) {

	const (
		expected = "Echo: Hi orchestrator, how are you?"
		msg      = "Hi orchestrator, how are you?"
	)
	var response any

	callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	rpcErr := clientPeer.Call(callCtx, "system.echo", msg, &response)
	cancel()

	if rpcErr != nil {
		t.Fatalf("RPC request failed: %v", rpcErr)
	}

	if response != expected {
		t.Errorf("expected %q, got %q", expected, response)
	}
}

func runOrchestrator_FacturX_ServiceTest(t *testing.T, clientPeer *rpc.Peer, resourcePath string) {

	const expected = "<ram:ID>RE-0815-max-fx</ram:ID>"
	var response writer.JobRPCResponse

	file := filepath.Join(resourcePath, "test_generate_fx.json")
	jsonData, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("failed to read json: %v", err)
	}

	callCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	rpcErr := clientPeer.Call(callCtx, "nexfact.process", string(jsonData), &response)
	cancel()

	if rpcErr != nil {
		t.Fatalf("RPC request failed: %v", rpcErr)
	}

	result := string(response.OutputFile.Data)
	if !strings.Contains(result, expected) {
		t.Errorf("expected invoice number %q not found in XML response", expected)
	}
}

func runOrchestrator_Combine_ServiceTest(t *testing.T, clientPeer *rpc.Peer, resourcePath string) {
	const expected = "iVBOR"

	// 1.Loading resources (a crash occurs automatically if a file is missing))
	pdfData := loadTestFile(t, resourcePath, "test_combine_invoice.pdf")
	xmlData := loadTestFile(t, resourcePath, "test_combine_invoice.xml")
	attData := loadTestFile(t, resourcePath, "test_attachment.png")

	// 2. Set up a map-based payload
	payload := map[string]any{
		"mandant": map[string]any{
			"seller": "testSeller",
		},
		"options": map[string]any{
			"queue": "combine",
			"pdf": map[string]any{
				"name": "test_combine_invoice.pdf",
				"data": pdfData,
			},
			"factur-x": map[string]any{
				"name": "test_combine_invoice.xml",
				"data": xmlData,
			},
			"attachments": []any{
				map[string]any{
					"name": "test_attachment.png",
					"data": attData,
				},
			},
		},
		"invoice": map[string]any{
			"invoice_id": "RE-0815-combine-mit",
		},
	}

	// 3. Marshal to JSON
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal request payload: %v", err)
	}

	// 4. RPC Call
	var response writer.JobRPCResponse
	callCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if rpcErr := clientPeer.Call(callCtx, "nexfact.process", string(jsonBytes), &response); rpcErr != nil {
		t.Fatalf("RPC request failed: %v", rpcErr)
	}

	// 1. response.OutputFile.Data is already []byte (raw binary data) due to json.Unmarshal.
	pdfBytes := response.OutputFile.Data

	// 2. Plausibility-Check: Is the file exactly at least 448089 bytes in size?
	if len(pdfBytes) != 448089 {
		t.Errorf("size of PDF output is not correct. Expectes 448089, got: %d bytes", len(pdfBytes))
	}

	// 3. Magic Bytes check (%PDF-)
	pdfHeader := []byte("%PDF-")
	if !bytes.HasPrefix(pdfBytes, pdfHeader) {
		t.Errorf("decoded output is not a valid PDF! Expected header %q, got %q", pdfHeader, pdfBytes[:5])
	}

}

// loadTestFile reads a file from resourcePath.
// In case of an error, the test is immediately aborted using t.Fatalf.
func loadTestFile(t *testing.T, resourcePath, filename string) []byte {
	t.Helper() // Ensure that Go reports the line within the test itself when an error occurs, rather than the line inside the helper function.

	path := filepath.Join(resourcePath, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read test resource %q: %v", filename, err)
	}
	return data
}
