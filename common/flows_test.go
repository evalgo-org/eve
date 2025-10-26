package common

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFlowProcessState_Constants validates all flow process state constants.
func TestFlowProcessState_Constants(t *testing.T) {
	assert.Equal(t, FlowProcessState("started"), StateStarted)
	assert.Equal(t, FlowProcessState("running"), StateRunning)
	assert.Equal(t, FlowProcessState("successful"), StateSuccessful)
	assert.Equal(t, FlowProcessState("failed"), StateFailed)
}

// TestFlowProcessState_String validates string conversion of states.
func TestFlowProcessState_String(t *testing.T) {
	assert.Equal(t, "started", string(StateStarted))
	assert.Equal(t, "running", string(StateRunning))
	assert.Equal(t, "successful", string(StateSuccessful))
	assert.Equal(t, "failed", string(StateFailed))
}

// TestFlowProcessMessage_JSON tests JSON marshaling and unmarshaling.
func TestFlowProcessMessage_JSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("MarshalWithAllFields", func(t *testing.T) {
		msg := FlowProcessMessage{
			ProcessID:   "test-process-123",
			State:       StateRunning,
			Timestamp:   now,
			Metadata:    map[string]interface{}{"step": 1, "total": 5},
			ErrorMsg:    "test error",
			Description: "Test description",
		}

		data, err := json.Marshal(msg)
		require.NoError(t, err)
		assert.Contains(t, string(data), "test-process-123")
		assert.Contains(t, string(data), "running")

		var unmarshaled FlowProcessMessage
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, msg.ProcessID, unmarshaled.ProcessID)
		assert.Equal(t, msg.State, unmarshaled.State)
		assert.Equal(t, msg.ErrorMsg, unmarshaled.ErrorMsg)
		assert.Equal(t, msg.Description, unmarshaled.Description)
	})

	t.Run("MarshalWithMinimalFields", func(t *testing.T) {
		msg := FlowProcessMessage{
			ProcessID: "minimal-process",
			State:     StateStarted,
			Timestamp: now,
		}

		data, err := json.Marshal(msg)
		require.NoError(t, err)

		var unmarshaled FlowProcessMessage
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, msg.ProcessID, unmarshaled.ProcessID)
		assert.Equal(t, msg.State, unmarshaled.State)
		assert.Empty(t, unmarshaled.ErrorMsg)
		assert.Empty(t, unmarshaled.Description)
	})

	t.Run("AllStates", func(t *testing.T) {
		states := []FlowProcessState{StateStarted, StateRunning, StateSuccessful, StateFailed}

		for _, state := range states {
			msg := FlowProcessMessage{
				ProcessID: "test-" + string(state),
				State:     state,
				Timestamp: now,
			}

			data, err := json.Marshal(msg)
			require.NoError(t, err)

			var unmarshaled FlowProcessMessage
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, state, unmarshaled.State)
		}
	})
}

// TestFlowProcessDocument_JSON tests JSON marshaling of process documents.
func TestFlowProcessDocument_JSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("FullDocument", func(t *testing.T) {
		doc := FlowProcessDocument{
			ID:        "flow_process_test-123",
			Rev:       "1-abc123",
			ProcessID: "test-123",
			State:     StateSuccessful,
			CreatedAt: now.Add(-1 * time.Hour),
			UpdatedAt: now,
			History: []FlowStateChange{
				{State: StateStarted, Timestamp: now.Add(-1 * time.Hour)},
				{State: StateRunning, Timestamp: now.Add(-30 * time.Minute)},
				{State: StateSuccessful, Timestamp: now},
			},
			Metadata:    map[string]interface{}{"duration": 3600},
			Description: "Completed successfully",
		}

		data, err := json.Marshal(doc)
		require.NoError(t, err)
		assert.Contains(t, string(data), "flow_process_test-123")
		assert.Contains(t, string(data), "successful")

		var unmarshaled FlowProcessDocument
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, doc.ID, unmarshaled.ID)
		assert.Equal(t, doc.Rev, unmarshaled.Rev)
		assert.Equal(t, doc.ProcessID, unmarshaled.ProcessID)
		assert.Equal(t, doc.State, unmarshaled.State)
		assert.Len(t, unmarshaled.History, 3)
	})

	t.Run("FailedProcess", func(t *testing.T) {
		doc := FlowProcessDocument{
			ID:        "flow_process_failed-456",
			ProcessID: "failed-456",
			State:     StateFailed,
			CreatedAt: now,
			UpdatedAt: now,
			History: []FlowStateChange{
				{State: StateStarted, Timestamp: now},
				{State: StateFailed, Timestamp: now, ErrorMsg: "Network timeout"},
			},
			ErrorMsg: "Network timeout",
		}

		data, err := json.Marshal(doc)
		require.NoError(t, err)

		var unmarshaled FlowProcessDocument
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, StateFailed, unmarshaled.State)
		assert.Equal(t, "Network timeout", unmarshaled.ErrorMsg)
	})
}

// TestFlowStateChange_JSON tests state change serialization.
func TestFlowStateChange_JSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("StateChangeWithError", func(t *testing.T) {
		change := FlowStateChange{
			State:     StateFailed,
			Timestamp: now,
			ErrorMsg:  "Validation failed",
		}

		data, err := json.Marshal(change)
		require.NoError(t, err)

		var unmarshaled FlowStateChange
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, change.State, unmarshaled.State)
		assert.Equal(t, change.ErrorMsg, unmarshaled.ErrorMsg)
	})

	t.Run("StateChangeWithoutError", func(t *testing.T) {
		change := FlowStateChange{
			State:     StateSuccessful,
			Timestamp: now,
		}

		data, err := json.Marshal(change)
		require.NoError(t, err)

		var unmarshaled FlowStateChange
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, change.State, unmarshaled.State)
		assert.Empty(t, unmarshaled.ErrorMsg)
	})
}

// TestFlowCouchDBResponse_JSON tests CouchDB response parsing.
func TestFlowCouchDBResponse_JSON(t *testing.T) {
	t.Run("SuccessResponse", func(t *testing.T) {
		resp := FlowCouchDBResponse{
			OK:  true,
			ID:  "flow_process_test",
			Rev: "2-xyz789",
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		var unmarshaled FlowCouchDBResponse
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.True(t, unmarshaled.OK)
		assert.Equal(t, resp.ID, unmarshaled.ID)
		assert.Equal(t, resp.Rev, unmarshaled.Rev)
	})

	t.Run("ParseFromJSON", func(t *testing.T) {
		jsonData := `{"ok":true,"id":"doc123","rev":"1-abc"}`

		var resp FlowCouchDBResponse
		err := json.Unmarshal([]byte(jsonData), &resp)
		require.NoError(t, err)

		assert.True(t, resp.OK)
		assert.Equal(t, "doc123", resp.ID)
		assert.Equal(t, "1-abc", resp.Rev)
	})
}

// TestFlowCouchDBError_JSON tests CouchDB error parsing.
func TestFlowCouchDBError_JSON(t *testing.T) {
	t.Run("ConflictError", func(t *testing.T) {
		err := FlowCouchDBError{
			Error:  "conflict",
			Reason: "Document update conflict",
		}

		data, jsonErr := json.Marshal(err)
		require.NoError(t, jsonErr)

		var unmarshaled FlowCouchDBError
		jsonErr = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, jsonErr)

		assert.Equal(t, "conflict", unmarshaled.Error)
		assert.Equal(t, "Document update conflict", unmarshaled.Reason)
	})

	t.Run("NotFoundError", func(t *testing.T) {
		jsonData := `{"error":"not_found","reason":"missing"}`

		var err FlowCouchDBError
		jsonErr := json.Unmarshal([]byte(jsonData), &err)
		require.NoError(t, jsonErr)

		assert.Equal(t, "not_found", err.Error)
		assert.Equal(t, "missing", err.Reason)
	})
}

// TestFlowConfig_Structure tests the FlowConfig structure.
func TestFlowConfig_Structure(t *testing.T) {
	t.Run("CompleteConfig", func(t *testing.T) {
		config := FlowConfig{
			RabbitMQURL:  "amqp://guest:guest@localhost:5672/",
			QueueName:    "test_queue",
			CouchDBURL:   "http://admin:password@localhost:5984",
			DatabaseName: "test_db",
			ApiKey:       "test-api-key",
		}

		assert.NotEmpty(t, config.RabbitMQURL)
		assert.NotEmpty(t, config.QueueName)
		assert.NotEmpty(t, config.CouchDBURL)
		assert.NotEmpty(t, config.DatabaseName)
		assert.NotEmpty(t, config.ApiKey)
	})

	t.Run("EmptyConfig", func(t *testing.T) {
		config := FlowConfig{}

		assert.Empty(t, config.RabbitMQURL)
		assert.Empty(t, config.QueueName)
		assert.Empty(t, config.CouchDBURL)
		assert.Empty(t, config.DatabaseName)
		assert.Empty(t, config.ApiKey)
	})
}

// TestFlowProcessMessage_Metadata tests metadata handling.
func TestFlowProcessMessage_Metadata(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("ComplexMetadata", func(t *testing.T) {
		msg := FlowProcessMessage{
			ProcessID: "meta-test",
			State:     StateRunning,
			Timestamp: now,
			Metadata: map[string]interface{}{
				"string":  "value",
				"number":  42,
				"float":   3.14,
				"bool":    true,
				"nested":  map[string]interface{}{"key": "nested_value"},
				"array":   []interface{}{1, 2, 3},
			},
		}

		data, err := json.Marshal(msg)
		require.NoError(t, err)

		var unmarshaled FlowProcessMessage
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.NotNil(t, unmarshaled.Metadata)
		assert.Equal(t, "value", unmarshaled.Metadata["string"])
		assert.Equal(t, float64(42), unmarshaled.Metadata["number"])
	})

	t.Run("NilMetadata", func(t *testing.T) {
		msg := FlowProcessMessage{
			ProcessID: "no-meta",
			State:     StateStarted,
			Timestamp: now,
			Metadata:  nil,
		}

		data, err := json.Marshal(msg)
		require.NoError(t, err)

		var unmarshaled FlowProcessMessage
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		// JSON omitempty should exclude nil metadata
		assert.Nil(t, unmarshaled.Metadata)
	})
}

// TestFlowProcessDocument_History tests history array handling.
func TestFlowProcessDocument_History(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("MultipleStateChanges", func(t *testing.T) {
		doc := FlowProcessDocument{
			ID:        "history-test",
			ProcessID: "history-test",
			State:     StateSuccessful,
			CreatedAt: now.Add(-10 * time.Minute),
			UpdatedAt: now,
			History: []FlowStateChange{
				{State: StateStarted, Timestamp: now.Add(-10 * time.Minute)},
				{State: StateRunning, Timestamp: now.Add(-8 * time.Minute)},
				{State: StateRunning, Timestamp: now.Add(-5 * time.Minute)},
				{State: StateRunning, Timestamp: now.Add(-2 * time.Minute)},
				{State: StateSuccessful, Timestamp: now},
			},
		}

		data, err := json.Marshal(doc)
		require.NoError(t, err)

		var unmarshaled FlowProcessDocument
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Len(t, unmarshaled.History, 5)
		assert.Equal(t, StateStarted, unmarshaled.History[0].State)
		assert.Equal(t, StateSuccessful, unmarshaled.History[4].State)
	})

	t.Run("EmptyHistory", func(t *testing.T) {
		doc := FlowProcessDocument{
			ID:        "no-history",
			ProcessID: "no-history",
			State:     StateStarted,
			CreatedAt: now,
			UpdatedAt: now,
			History:   []FlowStateChange{},
		}

		data, err := json.Marshal(doc)
		require.NoError(t, err)

		var unmarshaled FlowProcessDocument
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.NotNil(t, unmarshaled.History)
		assert.Len(t, unmarshaled.History, 0)
	})
}

// TestFlowProcessDocument_Timestamps tests timestamp handling.
func TestFlowProcessDocument_Timestamps(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	created := now.Add(-1 * time.Hour)

	doc := FlowProcessDocument{
		ID:        "timestamp-test",
		ProcessID: "timestamp-test",
		State:     StateRunning,
		CreatedAt: created,
		UpdatedAt: now,
		History:   []FlowStateChange{},
	}

	data, err := json.Marshal(doc)
	require.NoError(t, err)

	var unmarshaled FlowProcessDocument
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Allow small differences due to JSON time precision
	assert.WithinDuration(t, created, unmarshaled.CreatedAt, time.Second)
	assert.WithinDuration(t, now, unmarshaled.UpdatedAt, time.Second)
	assert.True(t, unmarshaled.UpdatedAt.After(unmarshaled.CreatedAt) ||
		unmarshaled.UpdatedAt.Equal(unmarshaled.CreatedAt))
}

// TestFlowStateChange_AllStates tests all possible state transitions.
func TestFlowStateChange_AllStates(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	states := []FlowProcessState{StateStarted, StateRunning, StateSuccessful, StateFailed}

	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			change := FlowStateChange{
				State:     state,
				Timestamp: now,
			}

			if state == StateFailed {
				change.ErrorMsg = "Test error for " + string(state)
			}

			data, err := json.Marshal(change)
			require.NoError(t, err)

			var unmarshaled FlowStateChange
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, state, unmarshaled.State)

			if state == StateFailed {
				assert.NotEmpty(t, unmarshaled.ErrorMsg)
			}
		})
	}
}
