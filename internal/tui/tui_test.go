package tui

import (
	"strings"
	"testing"

	"github.com/tasnimzotder/portman/internal/model"
)

// ---------- buildPortRows tests ----------

func TestBuildPortRows_EmptySlice(t *testing.T) {
	rows := buildPortRows(nil)
	if len(rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(rows))
	}

	rows = buildPortRows([]model.Listener{})
	if len(rows) != 0 {
		t.Errorf("expected 0 rows for empty slice, got %d", len(rows))
	}
}

func TestBuildPortRows_FullProcessInfo(t *testing.T) {
	listeners := []model.Listener{
		{
			Port:     8080,
			Protocol: "tcp",
			PID:      1234,
			Process: &model.Process{
				User:          "admin",
				Command:       "myapp serve",
				UptimeSeconds: 3661, // 1h1m1s
			},
			ConnectionCount: 5,
		},
	}

	rows := buildPortRows(listeners)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	// Columns: PORT, PROTO, PID, USER, COMMAND, CONNS, UPTIME
	if row[0] != "8080" {
		t.Errorf("PORT: got %q, want %q", row[0], "8080")
	}
	if row[1] != "tcp" {
		t.Errorf("PROTO: got %q, want %q", row[1], "tcp")
	}
	if row[2] != "1234" {
		t.Errorf("PID: got %q, want %q", row[2], "1234")
	}
	if row[3] != "admin" {
		t.Errorf("USER: got %q, want %q", row[3], "admin")
	}
	if row[4] != "myapp serve" {
		t.Errorf("COMMAND: got %q, want %q", row[4], "myapp serve")
	}
	if row[5] != "5" {
		t.Errorf("CONNS: got %q, want %q", row[5], "5")
	}
	// 3661 seconds = 1h1m
	if row[6] != "1h1m" {
		t.Errorf("UPTIME: got %q, want %q", row[6], "1h1m")
	}
}

func TestBuildPortRows_NilProcess(t *testing.T) {
	listeners := []model.Listener{
		{
			Port:            443,
			Protocol:        "tcp",
			PID:             5678,
			Process:         nil,
			ConnectionCount: 0,
		},
	}

	rows := buildPortRows(listeners)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	// PID > 0 so PID column shows the number
	if row[2] != "5678" {
		t.Errorf("PID: got %q, want %q", row[2], "5678")
	}
	// Process is nil, so user/command/uptime should be "-"
	if row[3] != "-" {
		t.Errorf("USER: got %q, want %q", row[3], "-")
	}
	if row[4] != "-" {
		t.Errorf("COMMAND: got %q, want %q", row[4], "-")
	}
	if row[6] != "-" {
		t.Errorf("UPTIME: got %q, want %q", row[6], "-")
	}
}

func TestBuildPortRows_PIDZero(t *testing.T) {
	listeners := []model.Listener{
		{
			Port:            53,
			Protocol:        "udp",
			PID:             0,
			Process:         nil,
			ConnectionCount: 0,
		},
	}

	rows := buildPortRows(listeners)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	// PID == 0 => shows "-"
	if row[2] != "-" {
		t.Errorf("PID: got %q, want %q for PID=0", row[2], "-")
	}
	if row[3] != "-" {
		t.Errorf("USER: got %q, want %q", row[3], "-")
	}
	if row[4] != "-" {
		t.Errorf("COMMAND: got %q, want %q", row[4], "-")
	}
	if row[6] != "-" {
		t.Errorf("UPTIME: got %q, want %q", row[6], "-")
	}
}

func TestBuildPortRows_CommandFallsBackToName(t *testing.T) {
	listeners := []model.Listener{
		{
			Port:     9090,
			Protocol: "tcp",
			PID:      100,
			Process: &model.Process{
				Name:    "prometheus",
				Command: "", // empty Command, should fall back to Name
			},
			ConnectionCount: 2,
		},
	}

	rows := buildPortRows(listeners)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	if rows[0][4] != "prometheus" {
		t.Errorf("COMMAND: got %q, want %q (fallback to Name)", rows[0][4], "prometheus")
	}
}

// ---------- filterByProtocol tests ----------

func TestFilterByProtocol_NoFilter(t *testing.T) {
	listeners := []model.Listener{
		{Port: 80, Protocol: "tcp"},
		{Port: 53, Protocol: "udp"},
		{Port: 443, Protocol: "tcp"},
	}

	result := filterByProtocol(listeners, false, false)
	if len(result) != 3 {
		t.Errorf("no filter: expected 3, got %d", len(result))
	}
}

func TestFilterByProtocol_TCPOnly(t *testing.T) {
	listeners := []model.Listener{
		{Port: 80, Protocol: "tcp"},
		{Port: 53, Protocol: "udp"},
		{Port: 443, Protocol: "tcp"},
		{Port: 5353, Protocol: "udp"},
	}

	result := filterByProtocol(listeners, true, false)
	if len(result) != 2 {
		t.Fatalf("tcpOnly: expected 2, got %d", len(result))
	}
	for _, l := range result {
		if l.Protocol != "tcp" {
			t.Errorf("tcpOnly: unexpected protocol %q", l.Protocol)
		}
	}
}

func TestFilterByProtocol_UDPOnly(t *testing.T) {
	listeners := []model.Listener{
		{Port: 80, Protocol: "tcp"},
		{Port: 53, Protocol: "udp"},
		{Port: 443, Protocol: "tcp"},
		{Port: 5353, Protocol: "udp"},
	}

	result := filterByProtocol(listeners, false, true)
	if len(result) != 2 {
		t.Fatalf("udpOnly: expected 2, got %d", len(result))
	}
	for _, l := range result {
		if l.Protocol != "udp" {
			t.Errorf("udpOnly: unexpected protocol %q", l.Protocol)
		}
	}
}

func TestFilterByProtocol_EmptyInput(t *testing.T) {
	result := filterByProtocol(nil, true, false)
	if len(result) != 0 {
		t.Errorf("empty input with tcpOnly: expected 0, got %d", len(result))
	}
}

func TestFilterByProtocol_BothFlags(t *testing.T) {
	// When both flags are set, only items matching both conditions pass,
	// but since a listener can't be both tcp and udp, nothing passes.
	listeners := []model.Listener{
		{Port: 80, Protocol: "tcp"},
		{Port: 53, Protocol: "udp"},
	}

	result := filterByProtocol(listeners, true, true)
	if len(result) != 0 {
		t.Errorf("both flags set: expected 0 (nothing is both tcp and udp), got %d", len(result))
	}
}

// ---------- helpBar / joinHelp tests ----------

func TestHelpBar_ReturnsJoinedString(t *testing.T) {
	result := helpBar("q: quit", "r: refresh", "?: help")
	// helpBar wraps with StyleDim.Render, which may add ANSI codes.
	// We just verify the content is in there.
	if !strings.Contains(result, "q: quit") {
		t.Errorf("expected help bar to contain %q, got %q", "q: quit", result)
	}
	if !strings.Contains(result, "r: refresh") {
		t.Errorf("expected help bar to contain %q, got %q", "r: refresh", result)
	}
	if !strings.Contains(result, "?: help") {
		t.Errorf("expected help bar to contain %q, got %q", "?: help", result)
	}
}

func TestJoinHelp_SingleItem(t *testing.T) {
	result := joinHelp("q: quit")
	if result != "q: quit" {
		t.Errorf("single item: got %q, want %q", result, "q: quit")
	}
}

func TestJoinHelp_MultipleItems(t *testing.T) {
	result := joinHelp("a", "b", "c")
	expected := "a · b · c"
	if result != expected {
		t.Errorf("multiple items: got %q, want %q", result, expected)
	}
}

func TestJoinHelp_NoItems(t *testing.T) {
	result := joinHelp()
	if result != "" {
		t.Errorf("no items: got %q, want empty", result)
	}
}

// ---------- newPortTable tests ----------

func TestNewPortTable_DefaultHeight(t *testing.T) {
	tbl := newPortTable(0)
	// Table should be created without panicking.
	// We verify it has the expected columns by checking View output contains headers.
	view := tbl.View()
	if len(view) == 0 {
		t.Error("expected non-empty table view")
	}
	if !strings.Contains(view, "PORT") {
		t.Error("expected table to contain PORT header")
	}
	if !strings.Contains(view, "PROTO") {
		t.Error("expected table to contain PROTO header")
	}
}

func TestNewPortTable_ExplicitHeight(t *testing.T) {
	tbl := newPortTable(10)
	view := tbl.View()
	if len(view) == 0 {
		t.Error("expected non-empty table view")
	}
	if !strings.Contains(view, "PORT") {
		t.Error("expected table to contain PORT header")
	}
}

func TestNewPortTable_NegativeHeight(t *testing.T) {
	// Negative should also default to 20, since height <= 0
	tbl := newPortTable(-5)
	view := tbl.View()
	if len(view) == 0 {
		t.Error("expected non-empty table view for negative height")
	}
}
