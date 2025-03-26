package mcp

import (
	"testing"
	"time"
)

func TestPromptsCapability(t *testing.T) {
	// Setup test environment
	serverProvider := NewTestServerImpl()
	serverInstance := &ServerImpl{
		MCPVersionNegotiator: serverProvider,
		CapPromptsProvider:   serverProvider,
		CapToolsProvider:     serverProvider,
	}
	clientProvider := &testClientImpl{roots_ListChanged: make(chan struct{})}
	clientInstance := &ClientImpl{
		CapRootsProvider:    clientProvider,
		CapSamplingProvider: clientProvider,
	}

	ts, err := SetupClientServer(serverInstance, clientInstance)
	if err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}
	defer ts.Cleanup()

	// Run test cases
	t.Run("Initialization", func(t *testing.T) {
		ts.Init(t)
	})

	// Run prompts-specific tests
	t.Run("ListPrompts", func(t *testing.T) {
		prompts, err := ts.Client.PromptsList(ts.Ctx, "")
		if err != nil {
			t.Fatalf("PromptsList failed: %v", err)
		}
		if len(prompts) == 0 || len(prompts[0].Prompts) == 0 {
			t.Fatal("Expected at least one test prompt")
		}
	})

	t.Run("GetPrompt", func(t *testing.T) {
		presp, err := ts.Client.PromptsGet(ts.Ctx, "test_prompt")
		if err != nil {
			t.Fatalf("PromptsGet failed: %v", err)
		}
		if len(presp.Messages) == 0 || len(presp.Messages[0].Content.Text) == 0 {
			t.Fatal("Expected at least one test message in the prompt")
		}
	})

	t.Run("GetNonexistentPrompt", func(t *testing.T) {
		_, err := ts.Client.PromptsGet(ts.Ctx, "bad_prompt")
		if err == nil {
			t.Fatal("Expected error for non-existent prompt")
		}
	})

	t.Run("PromptsNotifications", func(t *testing.T) {
		// Create a channel to track notification completion
		notifyDone := make(chan struct{})

		go func() {
			// Trigger a prompts list change
			serverProvider.prompts_ListChanged <- struct{}{}
			close(notifyDone)
		}()

		select {
		case <-notifyDone:
			// Notification sent successfully
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for prompts list change notification")
		}
	})

	// Cleanup
	ts.Cancel()
	<-time.After(100 * time.Millisecond) // Allow for graceful shutdown
}
