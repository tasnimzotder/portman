package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/tasnimzotder/portman/internal/model"
	portwait "github.com/tasnimzotder/portman/internal/wait"
)

func TestWaitCancellationAndTimeoutOutcomes(t *testing.T) {
	m := NewWaitModel(nil, 34567, time.Second, time.Millisecond, false)
	defer m.Cancel()
	final, _ := m.Update(tea.KeyPressMsg{Code: 'q'})
	if result := final.(WaitModel).Outcome(); result.Success || !errors.Is(result.Err, context.Canceled) {
		t.Fatalf("cancellation outcome: %+v", result)
	}
	late, _ := final.(WaitModel).Update(waitResultMsg{portwait.Result{Success: true}})
	if !errors.Is(late.(WaitModel).Outcome().Err, context.Canceled) {
		t.Fatal("late success overwrote cancellation")
	}
	m = NewWaitModel(nil, 34567, time.Second, time.Millisecond, false)
	defer m.Cancel()
	final, quit := m.Update(waitResultMsg{portwait.Result{Err: portwait.ErrTimeout}})
	if quit == nil {
		t.Fatal("completed wait did not quit")
	}
	if _, ok := quit().(tea.QuitMsg); !ok {
		t.Fatal("completed wait returned wrong command")
	}
	if result := final.(WaitModel).Outcome(); result.Success || !errors.Is(result.Err, portwait.ErrTimeout) {
		t.Fatalf("timeout outcome: %+v", result)
	}
}

func TestWatchRecoversAndUsesPreviousCounts(t *testing.T) {
	m := NewWatchModel(nil, time.Second, "port", false, false)
	a := model.Listener{Port: 34567, PID: 111, Protocol: "tcp", Address: "127.0.0.1", ConnectionCount: 1}
	result, _ := m.Update(listenersMsg{listeners: []model.Listener{a}})
	m = result.(WatchModel)
	result, _ = m.Update(listenersMsg{err: errors.New("temporary failure")})
	m = result.(WatchModel)
	a.ConnectionCount = 3
	result, _ = m.Update(listenersMsg{listeners: []model.Listener{a}})
	m = result.(WatchModel)
	if m.err != nil {
		t.Fatalf("recovery still shows %v", m.err)
	}
	if !strings.Contains(m.formatWatchRow(a), "+2") {
		t.Fatalf("missing previous-scan delta: %s", m.formatWatchRow(a))
	}
	b := a
	b.PID = 222
	result, _ = m.Update(listenersMsg{listeners: []model.Listener{b}})
	m = result.(WatchModel)
	if !m.removed[a.Key()] || !m.added[b.Key()] {
		t.Fatal("owner replacement was not detected")
	}
}

func TestSinglePortSnapshotSurvivesViews(t *testing.T) {
	m := NewWatchPortModel(nil, 34567, time.Second)
	a := &model.Listener{Port: 34567, PID: 111, ConnectionCount: 1, Stats: &model.ProcessStats{MemoryRSS: 1024}}
	next, _ := m.Update(portMsg{listener: a})
	m = next.(WatchModel)
	b := *a
	b.ConnectionCount = 3
	next, _ = m.Update(portMsg{listener: &b})
	m = next.(WatchModel)
	m.View()
	m.View()
	if m.prevSnap == nil || m.prevSnap.ConnectionCount != 1 {
		t.Fatal("rendering overwrote previous snapshot")
	}
	next, _ = m.Update(portMsg{listener: nil})
	m = next.(WatchModel)
	next, _ = m.Update(portMsg{listener: &b})
	m = next.(WatchModel)
	if m.prevSnap != nil {
		t.Fatal("new socket compared to removed socket")
	}
}

func TestKillFailureOutcome(t *testing.T) {
	m := KillModel{}
	want := errors.New("permission denied")
	result, _ := m.Update(killResultMsg{err: want, message: want.Error()})
	if !errors.Is(result.(KillModel).Outcome(), want) {
		t.Fatal("kill failure was lost")
	}
}
