package server

import (
	"encoding/json"
	"gostop/internal/game"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func dialTestWS(t *testing.T, srvURL string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(srvURL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial ws error: %v", err)
	}
	return conn
}

func readMsg(t *testing.T, conn *websocket.Conn) (ServerMsg, map[string]interface{}) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read ws error: %v", err)
	}
	var msg ServerMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("unmarshal ServerMsg error: %v", err)
	}
	var raw map[string]interface{}
	_ = json.Unmarshal(data, &raw)
	return msg, raw
}

func readUntilType(t *testing.T, conn *websocket.Conn, want string) (ServerMsg, map[string]interface{}) {
	t.Helper()
	for i := 0; i < 10; i++ {
		msg, raw := readMsg(t, conn)
		if msg.Type == want {
			return msg, raw
		}
	}
	t.Fatalf("did not receive message of type %q", want)
	return ServerMsg{}, nil
}

func TestEndToEndLobbyAndGameFlow(t *testing.T) {
	room := NewRoom("http://192.168.0.10:8081", []string{"http://192.168.0.10:8081"})
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWS(room, w, r)
	})
	mux.HandleFunc("/qr", ServeQR("http://192.168.0.10:8081"))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// 1. Verify /qr returns PNG
	resp, err := http.Get(srv.URL + "/qr")
	if err != nil || resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("unexpected /qr response: err=%v status=%v ct=%v", err, resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	_ = resp.Body.Close()

	// 2. Connect two clients; both immediately get initial lobby
	c1 := dialTestWS(t, srv.URL)
	defer c1.Close()
	c2 := dialTestWS(t, srv.URL)
	defer c2.Close()

	readUntilType(t, c1, "lobby")
	readUntilType(t, c2, "lobby")

	// 3. Client 1 joins -> must explicitly receive "seat": 0 in raw JSON!
	if err := c1.WriteJSON(ClientMsg{Type: "join", Name: "Alice"}); err != nil {
		t.Fatal(err)
	}
	w1, raw1 := readUntilType(t, c1, "welcome")
	seatVal, ok := raw1["seat"]
	if !ok || seatVal.(float64) != 0 || w1.Token == "" {
		t.Fatalf("expected explicit seat=0 and token for host, got raw=%v msg=%+v", raw1, w1)
	}

	// Unseated Client 2 must immediately receive live lobby update showing Alice
	l2, _ := readUntilType(t, c2, "lobby")
	if len(l2.Lobby.Players) != 1 || l2.Lobby.Players[0].Name != "Alice" {
		t.Fatalf("expected unseated client 2 to see Alice live, got %+v", l2.Lobby)
	}

	// 4. Client 2 joins Seat 1, then Leaves -> gets seat=-1 and empty token
	_ = c2.WriteJSON(ClientMsg{Type: "join", Name: "Bob"})
	w2, _ := readUntilType(t, c2, "welcome")
	if w2.Seat != 1 || w2.Token == "" {
		t.Fatalf("expected seat 1 for Bob, got %+v", w2)
	}
	readUntilType(t, c2, "lobby")
	readUntilType(t, c1, "lobby") // c1 sees Alice
	readUntilType(t, c1, "lobby") // c1 sees Bob

	_ = c2.WriteJSON(ClientMsg{Type: "leave"})
	wLeave, _ := readUntilType(t, c2, "welcome")
	if wLeave.Seat != -1 || wLeave.Token != "" {
		t.Fatalf("expected seat=-1 and empty token on leave, got %+v", wLeave)
	}
	lAfterLeave, _ := readUntilType(t, c1, "lobby")
	if len(lAfterLeave.Lobby.Players) != 1 {
		t.Fatalf("expected 1 player after Bob leaves, got %d", len(lAfterLeave.Lobby.Players))
	}

	// 5. Bob rejoins; 3rd client (Incognito) tries to join -> spectator + error toast
	_ = c2.WriteJSON(ClientMsg{Type: "join", Name: "Bob"})
	w2, _ = readUntilType(t, c2, "welcome")
	if w2.Seat != 1 {
		t.Fatalf("expected Bob back in seat 1, got %d", w2.Seat)
	}

	c3 := dialTestWS(t, srv.URL)
	defer c3.Close()
	readUntilType(t, c3, "lobby")
	_ = c3.WriteJSON(ClientMsg{Type: "join", Name: "Charlie"})
	w3, _ := readUntilType(t, c3, "welcome")
	if w3.Seat != -1 || !w3.Spectating {
		t.Fatalf("expected Charlie to be spectator, got %+v", w3)
	}
	errMsg, _ := readUntilType(t, c3, "error")
	if !strings.Contains(errMsg.Error, "full") {
		t.Fatalf("expected full-table error toast, got %q", errMsg.Error)
	}

	// 6. Stale token reconnect must NOT auto-seat
	c4 := dialTestWS(t, srv.URL)
	defer c4.Close()
	readUntilType(t, c4, "lobby")
	_ = c4.WriteJSON(ClientMsg{Type: "reconnect", Token: "expired-token", Name: "Ghost"})
	w4, _ := readUntilType(t, c4, "welcome")
	if w4.Seat != -1 || w4.Token != "" {
		t.Fatalf("expected stale token to be cleared, got %+v", w4)
	}

	// 7. Start game and play to completion over WebSockets
	_ = c1.WriteJSON(ClientMsg{Type: "start_game"})
	s1, _ := readUntilType(t, c1, "state")
	s2, _ := readUntilType(t, c2, "state")
	if s1.Game == nil || s2.Game == nil || s1.Game.MyIndex != 0 || s2.Game.MyIndex != 1 {
		t.Fatalf("unexpected initial game views: s1=%+v s2=%+v", s1.Game, s2.Game)
	}

	cur1, cur2 := s1.Game, s2.Game
	for steps := 0; steps < 200 && cur1.Phase != game.PhaseRoundOver; steps++ {
		var activeConn *websocket.Conn
		var activeView *game.PlayerView
		if cur1.CurrentTurn == 0 {
			activeConn = c1
			activeView = cur1
		} else {
			activeConn = c2
			activeView = cur2
		}

		var act game.Action
		switch activeView.Phase {
		case game.PhasePlayHand:
			if len(activeView.MyHand) > 0 {
				act = game.Action{Type: game.ActionPlayCard, CardID: activeView.MyHand[0].ID}
			} else {
				act = game.Action{Type: game.ActionBombPass}
			}
		case game.PhaseChooseHandMatch, game.PhaseChooseFlipMatch:
			act = game.Action{Type: game.ActionChooseMatch, CardID: activeView.ChoiceCards[0].ID}
		case game.PhaseGoOrStop:
			act = game.Action{Type: game.ActionCallStop}
		}

		if err := activeConn.WriteJSON(ClientMsg{Type: "action", Action: &act}); err != nil {
			t.Fatalf("send action error: %v", err)
		}
		ns1, _ := readUntilType(t, c1, "state")
		ns2, _ := readUntilType(t, c2, "state")
		cur1, cur2 = ns1.Game, ns2.Game
	}

	if cur1.Phase != game.PhaseRoundOver {
		t.Fatalf("expected game to finish, phase=%s", cur1.Phase)
	}

	// 8. Return to lobby clears game state for both clients
	_ = c1.WriteJSON(ClientMsg{Type: "return_lobby"})
	end1, _ := readUntilType(t, c1, "state")
	end2, _ := readUntilType(t, c2, "state")
	if end1.Game != nil || end2.Game != nil {
		t.Fatalf("expected nil game after return_lobby, got %+v / %+v", end1.Game, end2.Game)
	}

	// 9. Host switches to 4P mode -> live lobby broadcasts 4P config
	_ = c1.WriteJSON(ClientMsg{Type: "set_rules", PlayerCount: 4})
	l4, _ := readUntilType(t, c2, "lobby")
	if l4.Lobby.Config.PlayerCount != 4 {
		t.Fatalf("expected 4P config broadcast, got %d", l4.Lobby.Config.PlayerCount)
	}
}

func TestBotSeatAndUndoFlow(t *testing.T) {
	room := NewRoom("http://192.168.0.10:8081", []string{"http://192.168.0.10:8081"})
	room.botDelay = 20 * time.Millisecond // fast bot in tests
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWS(room, w, r)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c1 := dialTestWS(t, srv.URL)
	defer c1.Close()
	readUntilType(t, c1, "lobby")

	// Human joins as Host (Seat 0), adds 1 Bot (Seat 1), starts 2P round
	_ = c1.WriteJSON(ClientMsg{Type: "join", Name: "Eric"})
	readUntilType(t, c1, "welcome")
	readUntilType(t, c1, "lobby")

	_ = c1.WriteJSON(ClientMsg{Type: "add_bot"})
	lBot, _ := readUntilType(t, c1, "lobby")
	if len(lBot.Lobby.Players) != 2 || !lBot.Lobby.Players[1].IsBot {
		t.Fatalf("expected Bot in seat 1, got %+v", lBot.Lobby.Players)
	}

	_ = c1.WriteJSON(ClientMsg{Type: "start_game"})
	st, _ := readUntilType(t, c1, "state")
	if st.Game == nil || len(st.Game.MyHand) == 0 {
		t.Fatalf("expected active game with hand, got %+v", st.Game)
	}

	// Human plays 1 card -> canUndo becomes true, then human undoes!
	firstCardID := st.Game.MyHand[0].ID
	_ = c1.WriteJSON(ClientMsg{Type: "action", Action: &game.Action{Type: game.ActionPlayCard, CardID: firstCardID}})
	afterPlay, _ := readUntilType(t, c1, "state")
	if afterPlay.Game.Phase != game.PhaseRoundOver && !afterPlay.CanUndo {
		t.Fatalf("expected CanUndo=true after human move")
	}

	if afterPlay.CanUndo {
		_ = c1.WriteJSON(ClientMsg{Type: "undo"})
		for i := 0; i < 5; i++ {
			uMsg, _ := readUntilType(t, c1, "state")
			if uMsg.Game.CurrentTurn == 0 && !uMsg.CanUndo {
				return // Undo restored seat 0's turn and consumed undo!
			}
		}
		t.Fatalf("undo did not restore seat 0 turn")
	}
}

func TestReclaimDisconnectedSeat(t *testing.T) {
	room := NewRoom("http://192.168.0.10:8081", []string{"http://192.168.0.10:8081"})
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWS(room, w, r)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c1 := dialTestWS(t, srv.URL)
	defer c1.Close()
	c2 := dialTestWS(t, srv.URL)
	readUntilType(t, c1, "lobby")
	readUntilType(t, c2, "lobby")

	_ = c1.WriteJSON(ClientMsg{Type: "join", Name: "Eric"})
	readUntilType(t, c1, "welcome")
	_ = c2.WriteJSON(ClientMsg{Type: "join", Name: "Lexie"})
	readUntilType(t, c2, "welcome")
	_ = c1.WriteJSON(ClientMsg{Type: "start_game"})
	readUntilType(t, c1, "state")
	readUntilType(t, c2, "state")

	// Lexie disconnects (loses phone tab)
	_ = c2.Close()
	time.Sleep(40 * time.Millisecond)

	// Lexie opens a brand new tab (no token) -> receives spectator state, then clicks Rejoin as Lexie (seat 1)
	c2New := dialTestWS(t, srv.URL)
	defer c2New.Close()
	readUntilType(t, c2New, "lobby")
	readUntilType(t, c2New, "state")

	_ = c2New.WriteJSON(ClientMsg{Type: "reclaim", TargetSeat: 1})
	w, _ := readUntilType(t, c2New, "welcome")
	if w.Seat != 1 || w.Token == "" {
		t.Fatalf("expected reclaimed seat 1 with token, got %+v", w)
	}
	st, _ := readUntilType(t, c2New, "state")
	if st.Game == nil || st.Game.MyIndex != 1 || len(st.Game.MyHand) == 0 {
		t.Fatalf("expected Lexie's active hand restored, got %+v", st.Game)
	}
}

func TestNagariPotCarryover(t *testing.T) {
	room := NewRoom("http://localhost:8081", nil)
	room.seats = []*seatPlayer{
		{name: "A"},
		{name: "B"},
	}
	// Round 1 ends in Nagari (draw) -> each active player puts 1 pt into pot (pot = 2)
	room.game = &game.GameState{
		Players: []*game.PlayerState{{Name: "A"}, {Name: "B"}},
		Result: &game.RoundResult{
			IsDraw:      true,
			WinnerIdx:   -1,
			Settlements: []int{0, 0},
		},
	}
	room.applyRoundSettlementLocked()
	if room.pot != 2 || room.seats[0].totalPoints != -1 || room.seats[1].totalPoints != -1 {
		t.Fatalf("expected pot=2 and totals [-1,-1], got pot=%d totals=[%d,%d]", room.pot, room.seats[0].totalPoints, room.seats[1].totalPoints)
	}

	// Round 2: Player 0 wins 7 pts -> collects the 2-pt pot (+9 total for round) and pot resets to 0
	room.game.Result = &game.RoundResult{
		IsDraw:      false,
		WinnerIdx:   0,
		Settlements: []int{7, -7},
	}
	room.applyRoundSettlementLocked()
	if room.pot != 0 || room.seats[0].totalPoints != 8 || room.seats[1].totalPoints != -8 {
		t.Fatalf("expected pot=0 and totals [8,-8], got pot=%d totals=[%d,%d]", room.pot, room.seats[0].totalPoints, room.seats[1].totalPoints)
	}
}
