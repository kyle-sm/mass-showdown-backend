package messages_test

import (
	"testing"

	"surrealchemist.com/mass-showdown-backend/messages"
)

func TestParseMessage(t *testing.T) {
	testMsg := []byte(">▣世界から解放され▣\n|challstr|4|b950e6ed3443a6e3456211dbfffc3f0c2e84327a08241f8b45c6b\n")
	wsm, err := messages.ParseServerMessage(testMsg)
	if err != nil {
		t.Error(err)
	}
	if wsm.RoomID != "▣世界から解放され▣" {
		t.Fatalf("Expected RoomID to be myroom but instead got '%s'", wsm.RoomID)
	}
	if len(wsm.Messages) != 1 {
		t.Fatalf("Expected 1 message but got %d", len(wsm.Messages))
	}
	if wsm.Messages[0].Type != "challstr" {
		t.Errorf("Expected message type to be challstr but it was %s", wsm.Messages[0].Type)
	}
	if len(wsm.Messages[0].Data) != 2 {
		t.Fatalf("Expected data length of 2 but got %d", len(wsm.Messages[0].Data))
	}
	if wsm.Messages[0].Data[0] != "4" {
		t.Errorf("Expected index 0 of data to be '4' but got '%s'", wsm.Messages[0].Data[0])
	}
	
}

func TestParseMessageWithRawMessage(t *testing.T) {
	rawMsg := []byte("Yo waddup")
	wsr, err := messages.ParseServerMessage(rawMsg)
	if err != nil {
		t.Error(err)
	}
	if wsr.Messages[0].Type != "raw" {
		t.Errorf("Expected message type to be raw but it was %s", wsr.Messages[0].Type)
	}
}

func TestParseBattleStartMessage(t *testing.T) {
	testMsg := []byte(`|player|p1|Anonycat|60|1200
|player|p2|Anonybird|113|1300
|teamsize|p1|4
|teamsize|p2|5
|gametype|doubles
|gen|7
|tier|[Gen 7] Doubles Ubers
|rule|Species Clause: Limit one of each Pokémon
|rule|OHKO Clause: OHKO moves are banned
|rule|Moody Clause: Moody is banned
|rule|Evasion Abilities Clause: Evasion abilities are banned
|rule|Evasion Moves Clause: Evasion moves are banned
|rule|Endless Battle Clause: Forcing endless battles is banned
|rule|HP Percentage Mod: HP is shown in percentages
|clearpoke
|poke|p1|Pikachu, L59, F|item
|poke|p1|Kecleon, M|item
|poke|p1|Jynx, F|item
|poke|p1|Mewtwo|item
|poke|p2|Hoopa-Unbound|
|poke|p2|Smeargle, L1, F|item
|poke|p2|Forretress, L31, F|
|poke|p2|Groudon, L60|item
|poke|p2|Feebas, L1, M|
|teampreview
|
|start`)
	wsm, err := messages.ParseServerMessage(testMsg)
	if err != nil {
		t.Error(err)
	}
	if len(wsm.Messages) != 27 {
		t.Fatalf("Expected 27 messages but got %d", len(wsm.Messages))
	}
	if wsm.Messages[0].Type != "player" {
		t.Errorf("Expected message type to be player but it was %s", wsm.Messages[0].Type)
	}
	if len(wsm.Messages[0].Data) != 4 {
		t.Fatalf("Expected data length of 4 but got %d", len(wsm.Messages[0].Data))
	}
	if wsm.Messages[1].Data[1] != "Anonybird" {
		t.Errorf("Expected index 1 of data to be 'Anonybird' but got '%s'", wsm.Messages[1].Data[1])
	}
}
