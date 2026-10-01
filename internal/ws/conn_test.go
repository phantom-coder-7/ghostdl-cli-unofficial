package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/version"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/wstest"
)

// helloAckHandler upgrades and replies to hello with hello_ack, then waits for close.
func helloAckHandler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		require.NoError(t, conn.WriteJSON(HelloAckMsg{
			Type:            TypeHelloAck,
			ProtocolVersion: ProtocolVersion,
			AppVersion:      "3.0.0",
		}))
		conn.ReadMessage()
	}
}

var testUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// TestDial_WSS_SkipVerifyConnectsToUnknownCA verifies skip-verify can dial a self-signed wss server.
func TestDial_WSS_SkipVerifyConnectsToUnknownCA(t *testing.T) {
	srv := httptest.NewTLSServer(helloAckHandler(t))
	defer srv.Close()

	c, err := Dial(context.Background(), wstest.TLSURL(srv), "test-token", WithInsecureSkipVerify(true))
	require.NoError(t, err)
	require.NotNil(t, c)
	_ = c.Close()
}

// TestDial_WSS_VerifyFailsAgainstUnknownCA verifies default wss dials reject unknown CA certs.
func TestDial_WSS_VerifyFailsAgainstUnknownCA(t *testing.T) {
	srv := httptest.NewTLSServer(helloAckHandler(t))
	defer srv.Close()

	_, err := Dial(context.Background(), wstest.TLSURL(srv), "test-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect")
}

// TestDial_WSS_VerifyFalseExplicitStillFails verifies WithInsecureSkipVerify(false) still verifies.
func TestDial_WSS_VerifyFalseExplicitStillFails(t *testing.T) {
	srv := httptest.NewTLSServer(helloAckHandler(t))
	defer srv.Close()

	_, err := Dial(context.Background(), wstest.TLSURL(srv), "test-token", WithInsecureSkipVerify(false))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect")
}

// TestDial_HelloAck verifies Dial succeeds when the server replies with hello_ack.
func TestDial_HelloAck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		assert.Equal(t, TypeHello, hello.Type)
		assert.Equal(t, "test-token", hello.Token)
		assert.Equal(t, ProtocolVersion, hello.ProtocolVersion)
		assert.NotEmpty(t, hello.RequestID)

		require.NoError(t, conn.WriteJSON(HelloAckMsg{
			Type:            TypeHelloAck,
			ProtocolVersion: ProtocolVersion,
			AppVersion:      "3.0.0",
		}))

		conn.ReadMessage()
	}))
	defer srv.Close()

	c, err := Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, "3.0.0", c.AppVersion())
	_ = c.Close()
}

// TestDial_Unauthorized verifies Dial returns an error when the server sends unauthorized.
func TestDial_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))

		require.NoError(t, conn.WriteJSON(ErrorMsg{
			Type: TypeError,
			Code: "unauthorized",
		}))
	}))
	defer srv.Close()

	_, err := Dial(context.Background(), wstest.URL(srv), "bad-token")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
	assert.Contains(t, err.Error(), "token is invalid")
}

// TestDial_ProtocolMismatch verifies Dial returns an error on protocol_mismatch.
func TestDial_ProtocolMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))

		require.NoError(t, conn.WriteJSON(ErrorMsg{
			Type: TypeError,
			Code: "protocol_mismatch",
		}))
	}))
	defer srv.Close()

	_, err := Dial(context.Background(), wstest.URL(srv), "any-token")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProtocolMismatch)
	assert.Contains(t, err.Error(), "protocol")
}

// TestDial_UnknownResponseType verifies Dial returns an error on unexpected response type.
func TestDial_UnknownResponseType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))

		require.NoError(t, conn.WriteJSON(map[string]string{"type": "something_unexpected"}))
	}))
	defer srv.Close()

	_, err := Dial(context.Background(), wstest.URL(srv), "any-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected response type")
}

// TestPairRequest_Success verifies PairRequest returns the token on user approval.
func TestPairRequest_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var req PairRequestMsg
		require.NoError(t, conn.ReadJSON(&req))
		assert.Equal(t, TypePairRequest, req.Type)
		assert.Equal(t, ProtocolVersion, req.ProtocolVersion)
		assert.Equal(t, version.Display(), req.ExtensionVersion)
		assert.NotEmpty(t, req.RequestID)

		require.NoError(t, conn.WriteJSON(PairResultMsg{
			Type:      TypePairResult,
			RequestID: req.RequestID,
			OK:        true,
			Message:   "配对成功",
			Token:     "new-pair-token",
		}))
	}))
	defer srv.Close()

	token, err := PairRequest(context.Background(), wstest.URL(srv))
	require.NoError(t, err)
	assert.Equal(t, "new-pair-token", token)
}

// TestPairRequest_Denied verifies PairRequest returns an error when the user denies.
func TestPairRequest_Denied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var req PairRequestMsg
		require.NoError(t, conn.ReadJSON(&req))

		require.NoError(t, conn.WriteJSON(PairResultMsg{
			Type:      TypePairResult,
			RequestID: req.RequestID,
			OK:        false,
			Message:   "用户拒绝",
		}))
	}))
	defer srv.Close()

	_, err := PairRequest(context.Background(), wstest.URL(srv))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "denied")
}

// TestPairRequest_EmptyToken verifies PairRequest returns an error when ok=true but token is empty.
func TestPairRequest_EmptyToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var req PairRequestMsg
		require.NoError(t, conn.ReadJSON(&req))

		require.NoError(t, conn.WriteJSON(PairResultMsg{
			Type:      TypePairResult,
			RequestID: req.RequestID,
			OK:        true,
			Message:   "ok",
			Token:     "",
		}))
	}))
	defer srv.Close()

	_, err := PairRequest(context.Background(), wstest.URL(srv))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no token")
}

// TestDial_Unreachable verifies Dial returns an error when the server is not listening.
func TestDial_Unreachable(t *testing.T) {

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := wstest.URL(srv)
	srv.Close()

	_, err := Dial(context.Background(), url, "any-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect")
}

// TestPing_Success verifies Ping returns HelloAckMsg with appVersion on success.
func TestPing_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		assert.Equal(t, TypeHello, hello.Type)
		assert.Equal(t, "ping-token", hello.Token)

		require.NoError(t, conn.WriteJSON(HelloAckMsg{
			Type:            TypeHelloAck,
			ProtocolVersion: ProtocolVersion,
			AppVersion:      "3.5.0",
		}))
	}))
	defer srv.Close()

	ack, err := Ping(context.Background(), wstest.URL(srv), "ping-token")
	require.NoError(t, err)
	require.NotNil(t, ack)
	assert.Equal(t, TypeHelloAck, ack.Type)
	assert.Equal(t, "3.5.0", ack.AppVersion)
	assert.Equal(t, ProtocolVersion, ack.ProtocolVersion)
}

// TestPing_Unauthorized verifies Ping wraps ErrUnauthorized on bad token.
func TestPing_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))

		require.NoError(t, conn.WriteJSON(ErrorMsg{
			Type: TypeError,
			Code: "unauthorized",
		}))
	}))
	defer srv.Close()

	_, err := Ping(context.Background(), wstest.URL(srv), "bad-token")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
	assert.Contains(t, err.Error(), "token is invalid")
}

// TestPing_ProtocolMismatch verifies Ping wraps ErrProtocolMismatch.
func TestPing_ProtocolMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))

		require.NoError(t, conn.WriteJSON(ErrorMsg{
			Type: TypeError,
			Code: "protocol_mismatch",
		}))
	}))
	defer srv.Close()

	_, err := Ping(context.Background(), wstest.URL(srv), "any-token")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProtocolMismatch)
	assert.Contains(t, err.Error(), "protocol")
}

// TestPing_Unreachable verifies Ping returns an error when the server is not listening.
func TestPing_Unreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := wstest.URL(srv)
	srv.Close()

	_, err := Ping(context.Background(), url, "any-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect")
}

// TestDial_StoresTaskActions verifies Dial persists capabilities.taskActions from hello_ack.
func TestDial_StoresTaskActions(t *testing.T) {
	caps, err := json.Marshal(Capabilities{
		TaskSnapshots: true,
		TaskActions: []TaskAction{
			ActionTogglePause, ActionCancel, ActionRemove,
			ActionRedownload, ActionOpenFile, ActionOpenFolder,
		},
	})
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		require.NoError(t, conn.WriteJSON(HelloAckMsg{
			Type:            TypeHelloAck,
			ProtocolVersion: ProtocolVersion,
			AppVersion:      "4.0.0",
			Capabilities:    caps,
		}))
		conn.ReadMessage()
	}))
	defer srv.Close()

	c, err := Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer c.Close()

	actions := c.TaskActions()
	require.Len(t, actions, 6)
	assert.Equal(t, ActionRemove, actions[2])
}

// TestSendGD_CorrelatesByRequestID verifies SendGD matches responses on requestId.
func TestSendGD_CorrelatesByRequestID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		require.NoError(t, conn.WriteJSON(HelloAckMsg{
			Type:            TypeHelloAck,
			ProtocolVersion: ProtocolVersion,
			AppVersion:      "4.0.0",
		}))

		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var req TaskActionMsg
			if err := json.Unmarshal(msgBytes, &req); err != nil {
				continue
			}
			if req.Type != TypeTaskAction {
				continue
			}
			require.NoError(t, conn.WriteJSON(TaskActionResultMsg{
				Type:      TypeTaskActionResult,
				RequestID: req.RequestID,
				OK:        true,
				TaskID:    req.TaskID,
			}))
		}
	}))
	defer srv.Close()

	c, err := Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer c.Close()

	reqID := "req-sendgd-success"
	msg := TaskActionMsg{
		Type:      TypeTaskAction,
		RequestID: reqID,
		TaskID:    "task-abc",
		Action:    ActionTogglePause,
	}
	raw, err := c.SendGD(msg, reqID, TypeTaskActionResult, 5*time.Second)
	require.NoError(t, err)

	var result TaskActionResultMsg
	require.NoError(t, json.Unmarshal(raw, &result))
	assert.True(t, result.OK)
	assert.Equal(t, reqID, result.RequestID)
	assert.Equal(t, "task-abc", result.TaskID)
}

// TestSendGD_ErrorReply verifies a top-level error with requestId is a failure.
func TestSendGD_ErrorReply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		require.NoError(t, conn.WriteJSON(HelloAckMsg{
			Type:            TypeHelloAck,
			ProtocolVersion: ProtocolVersion,
			AppVersion:      "4.0.0",
		}))

		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var req TaskActionMsg
			if err := json.Unmarshal(msgBytes, &req); err != nil {
				continue
			}
			if req.Type != TypeTaskAction {
				continue
			}
			require.NoError(t, conn.WriteJSON(ErrorMsg{
				Type:      TypeError,
				RequestID: req.RequestID,
				Code:      "bad_request",
				Message:   "任务不存在",
			}))
		}
	}))
	defer srv.Close()

	c, err := Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer c.Close()

	reqID := "req-sendgd-error"
	msg := TaskActionMsg{
		Type:      TypeTaskAction,
		RequestID: reqID,
		TaskID:    "missing-task",
		Action:    ActionRemove,
	}
	_, err = c.SendGD(msg, reqID, TypeTaskActionResult, 5*time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad_request")
	assert.Contains(t, err.Error(), "任务不存在")
}

// TestSendGD_Timeout verifies SendGD returns a timeout error when no reply arrives.
func TestSendGD_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		require.NoError(t, conn.WriteJSON(HelloAckMsg{
			Type:            TypeHelloAck,
			ProtocolVersion: ProtocolVersion,
			AppVersion:      "4.0.0",
		}))

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	c, err := Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer c.Close()

	reqID := "req-sendgd-timeout"
	msg := TaskActionMsg{
		Type:      TypeTaskAction,
		RequestID: reqID,
		TaskID:    "task-abc",
		Action:    ActionTogglePause,
	}
	_, err = c.SendGD(msg, reqID, TypeTaskActionResult, 200*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

// TestSubscribeTasksStream_SkipsInvalidFrames verifies garbage frames are skipped without hanging.
func TestSubscribeTasksStream_SkipsInvalidFrames(t *testing.T) {
	valid := TaskSnapshotMsg{
		Type:  TypeTaskSnapshot,
		Tasks: []GDTask{{TaskID: "t1", Name: "a.zip", Status: "running", Progress: 10}},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		require.NoError(t, conn.WriteJSON(HelloAckMsg{
			Type:            TypeHelloAck,
			ProtocolVersion: ProtocolVersion,
			AppVersion:      "4.0.0",
		}))

		var sub SubscribeTasksMsg
		require.NoError(t, conn.ReadJSON(&sub))
		assert.Equal(t, TypeSubscribeTasks, sub.Type)

		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte("not json")))
		require.NoError(t, conn.WriteJSON(valid))
		conn.ReadMessage()
	}))
	defer srv.Close()

	c, err := Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := c.SubscribeTasksStream(ctx)
	require.NoError(t, err)

	var received []*TaskSnapshotMsg
	for snap := range ch {
		received = append(received, snap)
		cancel()
		break
	}

	require.Len(t, received, 1)
	assert.Equal(t, "t1", received[0].Tasks[0].TaskID)
}

// TestSubscribeTasksStream_MultipleSnapshots verifies the stream yields repeated pushes.
func TestSubscribeTasksStream_MultipleSnapshots(t *testing.T) {
	snapshots := []TaskSnapshotMsg{
		{Type: TypeTaskSnapshot, Tasks: []GDTask{{TaskID: "t1", Name: "a.zip", Status: "running", Progress: 10}}},
		{Type: TypeTaskSnapshot, Tasks: []GDTask{{TaskID: "t1", Name: "a.zip", Status: "running", Progress: 50}}},
		{Type: TypeTaskSnapshot, Tasks: []GDTask{{TaskID: "t1", Name: "a.zip", Status: "completed", Progress: 100}}},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		require.NoError(t, conn.WriteJSON(HelloAckMsg{
			Type:            TypeHelloAck,
			ProtocolVersion: ProtocolVersion,
			AppVersion:      "4.0.0",
		}))

		var sub SubscribeTasksMsg
		require.NoError(t, conn.ReadJSON(&sub))
		assert.Equal(t, TypeSubscribeTasks, sub.Type)

		for _, snap := range snapshots {
			require.NoError(t, conn.WriteJSON(snap))
		}

		conn.ReadMessage()
	}))
	defer srv.Close()

	c, err := Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := c.SubscribeTasksStream(ctx)
	require.NoError(t, err)

	var received []*TaskSnapshotMsg
	for snap := range ch {
		received = append(received, snap)
		if len(received) == len(snapshots) {
			cancel()
		}
	}

	require.Len(t, received, 3)
	assert.Equal(t, 10.0, received[0].Tasks[0].Progress)
	assert.Equal(t, 50.0, received[1].Tasks[0].Progress)
	assert.Equal(t, 100.0, received[2].Tasks[0].Progress)
	assert.Equal(t, "completed", received[2].Tasks[0].Status)
}

// TestSubscribeTasksOnce_NoRequestID verifies snapshots without requestId are received.
// Real Ghost Downloader never echoes requestId on task_snapshot.
func TestSubscribeTasksOnce_NoRequestID(t *testing.T) {
	const snapshotJSON = `{"type":"task_snapshot","tasks":[{"taskId":"t1","name":"a.zip","status":"running","progress":10}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		require.NoError(t, conn.WriteJSON(HelloAckMsg{
			Type:            TypeHelloAck,
			ProtocolVersion: ProtocolVersion,
			AppVersion:      "4.3.7",
		}))

		var sub SubscribeTasksMsg
		require.NoError(t, conn.ReadJSON(&sub))
		assert.Equal(t, TypeSubscribeTasks, sub.Type)
		assert.NotEmpty(t, sub.RequestID)

		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(snapshotJSON)))
		conn.ReadMessage()
	}))
	defer srv.Close()

	c, err := Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer c.Close()

	snap, err := c.SubscribeTasksOnce(2 * time.Second)
	require.NoError(t, err)
	require.Len(t, snap.Tasks, 1)
	assert.Equal(t, "t1", snap.Tasks[0].TaskID)
}

// TestSendGD_IgnoresTaskSnapshotWithoutRequestID verifies task_snapshot pushes cannot
// satisfy a SendGD waiter. This is the production failure mode when snapshot reads
// incorrectly use SendGD correlation.
func TestSendGD_IgnoresTaskSnapshotWithoutRequestID(t *testing.T) {
	const snapshotJSON = `{"type":"task_snapshot","tasks":[]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		require.NoError(t, conn.WriteJSON(HelloAckMsg{
			Type:            TypeHelloAck,
			ProtocolVersion: ProtocolVersion,
			AppVersion:      "4.3.7",
		}))

		var action TaskActionMsg
		require.NoError(t, conn.ReadJSON(&action))
		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(snapshotJSON)))
		require.NoError(t, conn.WriteJSON(TaskActionResultMsg{
			Type:      TypeTaskActionResult,
			RequestID: action.RequestID,
			OK:        true,
			TaskID:    action.TaskID,
		}))
		conn.ReadMessage()
	}))
	defer srv.Close()

	c, err := Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer c.Close()

	reqID := "req-action-1"
	msg := TaskActionMsg{
		Type:      TypeTaskAction,
		RequestID: reqID,
		TaskID:    "t1",
		Action:    ActionRemove,
	}

	raw, err := c.SendGD(msg, reqID, TypeTaskActionResult, 2*time.Second)
	require.NoError(t, err)

	var result TaskActionResultMsg
	require.NoError(t, json.Unmarshal(raw, &result))
	assert.True(t, result.OK)
}

func TestConnCloseUnblocksRead(t *testing.T) {
	srv := httptest.NewServer(helloAckHandler(t))
	defer srv.Close()

	c, err := Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := c.SubscribeTasksStream(ctx)
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		_, ok := <-stream
		if !ok {
			close(done)
		}
	}()

	require.NoError(t, c.Close())

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not unblock the read loop")
	}
}
