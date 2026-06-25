package presence

import (
	"testing"
	"time"
)

func TestJoinListLeave(t *testing.T) {
	r := New()
	a := r.Join("Alice", "10.0.0.1")
	b := r.Join("Bob", "10.0.0.2")
	if r.Count() != 2 {
		t.Fatalf("Count = %d, veut 2", r.Count())
	}
	list := r.List()
	if len(list) != 2 || list[0].Handle != "Alice" || list[1].Handle != "Bob" {
		t.Fatalf("List = %+v, veut [Alice Bob] dans l'ordre d'arrivee", list)
	}
	r.SetHandle(a, "Alice2")
	if got := r.List()[0].Handle; got != "Alice2" {
		t.Fatalf("SetHandle: handle = %q, veut Alice2", got)
	}
	r.Leave(b)
	if r.Count() != 1 {
		t.Fatalf("apres Leave: Count = %d, veut 1", r.Count())
	}
}

func TestListSortedByArrival(t *testing.T) {
	r := New()
	base := time.Unix(1000, 0)
	r.now = func() time.Time { base = base.Add(time.Second); return base }
	r.Join("premier", "")
	r.Join("second", "")
	r.Join("troisieme", "")
	list := r.List()
	want := []string{"premier", "second", "troisieme"}
	for i, w := range want {
		if list[i].Handle != w {
			t.Fatalf("List[%d] = %q, veut %q", i, list[i].Handle, w)
		}
	}
}

func TestChatSubscribePublishBacklog(t *testing.T) {
	r := New()
	id := r.Join("Alice", "")
	// Messages publies avant l'abonnement -> rappel (backlog).
	r.Publish(Message{From: "Bob", Text: "salut"})
	ch, backlog := r.Subscribe(id)
	if len(backlog) != 1 || backlog[0].Text != "salut" {
		t.Fatalf("backlog = %+v, veut [salut]", backlog)
	}
	// Message publie apres l'abonnement -> recu en direct.
	r.Publish(Message{From: "Bob", Text: "ca va ?"})
	select {
	case m := <-ch:
		if m.Text != "ca va ?" {
			t.Fatalf("recu %q, veut 'ca va ?'", m.Text)
		}
	case <-time.After(time.Second):
		t.Fatal("aucun message recu en direct")
	}
}

func TestPublishNonBlockingWhenBufferFull(t *testing.T) {
	r := New()
	id := r.Join("Slow", "")
	r.Subscribe(id) // on ne lit jamais le canal
	// Plus de messages que le tampon : ne doit jamais bloquer.
	done := make(chan struct{})
	go func() {
		for i := 0; i < subBuffer+50; i++ {
			r.Publish(Message{From: "flood", Text: "x"})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish a bloque sur un abonne lent")
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	r := New()
	id := r.Join("Alice", "")
	ch, _ := r.Subscribe(id)
	r.Unsubscribe(id)
	r.Publish(Message{From: "Bob", Text: "apres desabonnement"})
	select {
	case m, ok := <-ch:
		if ok {
			t.Fatalf("message recu apres Unsubscribe : %+v", m)
		}
	case <-time.After(100 * time.Millisecond):
		// rien recu : comportement attendu
	}
}

func TestBacklogBounded(t *testing.T) {
	r := New()
	for i := 0; i < backlogSize+10; i++ {
		r.Publish(Message{From: "x", Text: "m"})
	}
	_, backlog := r.Subscribe(r.Join("late", ""))
	if len(backlog) != backlogSize {
		t.Fatalf("backlog = %d, veut %d (borne)", len(backlog), backlogSize)
	}
}
