package index

import (
	"math/rand"
	"reflect"
	"testing"
)

func appOp(dev string, seq, lam uint64, key string, data string, parents ...string) Op {
	return Op{Device: dev, DeviceName: dev, Seq: seq, Lamport: lam, Kind: OpApp, Path: "/",
		App: &AppChange{App: "chat", Key: key, Parents: parents, Data: []byte(data)}}
}

func heads(r AppRecord) []string {
	var out []string
	for _, h := range r.Live() {
		out = append(out, h.ID+"="+string(h.Data))
	}
	return out
}

func TestAppRecordHeads(t *testing.T) {
	ops := []Op{
		appOp("a", 1, 1, "vault", "v1"),
		appOp("a", 2, 2, "vault", "v2", "a.1"),
		// b and c both saw v2 and wrote offline.
		appOp("b", 1, 3, "vault", "b", "a.2"),
		appOp("c", 1, 3, "vault", "c", "a.2"),
		// Another record and another app are not touched.
		appOp("a", 3, 3, "other", "x"),
		{Device: "a", Seq: 4, Lamport: 4, Kind: OpApp, Path: "/", App: &AppChange{App: "mail", Key: "vault", Data: []byte("m")}},
	}
	x := New()
	x.Add(ops...)
	r := x.AppRecord("chat", "vault")
	if got, want := heads(r), []string{"b.1=b", "c.1=c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("heads %v, want %v", got, want)
	}
	if r.Revision() != "b.1,c.1" {
		t.Fatalf("revision %q", r.Revision())
	}
	// The merge replaces both.
	x.Add(appOp("b", 2, 4, "vault", "merged", ParseRevision(r.Revision())...))
	if got := heads(x.AppRecord("chat", "vault")); !reflect.DeepEqual(got, []string{"b.2=merged"}) {
		t.Fatalf("after merge %v", got)
	}
	if got := heads(x.AppRecord("mail", "vault")); !reflect.DeepEqual(got, []string{"a.4=m"}) {
		t.Fatalf("other app %v", got)
	}
	if got := x.Apps(); !reflect.DeepEqual(got, []string{"chat", "mail"}) {
		t.Fatalf("apps %v", got)
	}
	if n := len(x.AppRecords("chat")); n != 2 {
		t.Fatalf("%d chat records", n)
	}
	if r := x.AppRecord("chat", "nope"); r.Revision() != "" || len(r.Heads) != 0 {
		t.Fatalf("unknown record %+v", r)
	}
}

func TestAppRecordDelete(t *testing.T) {
	x := New()
	x.Add(appOp("a", 1, 1, "k", "v1"))
	del := Op{Device: "a", Seq: 2, Lamport: 2, Kind: OpApp, Path: "/",
		App: &AppChange{App: "chat", Key: "k", Parents: []string{"a.1"}, Deleted: true}}
	x.Add(del)
	r := x.AppRecord("chat", "k")
	if len(r.Live()) != 0 || r.Revision() != "a.2" {
		t.Fatalf("after delete %+v", r)
	}
	// A concurrent edit survives the delete; the next write replaces both.
	x.Add(appOp("b", 1, 2, "k", "edit", "a.1"))
	r = x.AppRecord("chat", "k")
	if got := heads(r); !reflect.DeepEqual(got, []string{"b.1=edit"}) || r.Revision() != "a.2,b.1" {
		t.Fatalf("delete vs edit: %v %q", got, r.Revision())
	}
	x.Add(appOp("b", 2, 3, "k", "again", ParseRevision(r.Revision())...))
	if r := x.AppRecord("chat", "k"); r.Revision() != "b.2" {
		t.Fatalf("after rewrite %q", r.Revision())
	}
}

// The heads are a function of the set of ops: every arrival order (a
// parent arriving after its child, too) gives the same record.
func TestAppRecordOrderIndependent(t *testing.T) {
	ops := []Op{
		appOp("a", 1, 1, "k", "1"),
		appOp("b", 1, 2, "k", "2", "a.1"),
		appOp("c", 1, 2, "k", "3", "a.1"),
		appOp("a", 2, 3, "k", "4", "b.1"),
		appOp("c", 2, 4, "k", "5", "a.2", "c.1"),
		appOp("b", 2, 3, "k", "6", "b.1"),
	}
	want := New()
	want.Add(ops...)
	exp := want.AppRecord("chat", "k")
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		perm := rng.Perm(len(ops))
		x := New()
		for _, j := range perm {
			x.Add(ops[j])
		}
		if got := x.AppRecord("chat", "k"); !reflect.DeepEqual(got, exp) {
			t.Fatalf("order %v: %+v, want %+v", perm, got, exp)
		}
	}
	if got := heads(exp); !reflect.DeepEqual(got, []string{"b.2=6", "c.2=5"}) {
		t.Fatalf("heads %v", got)
	}
}
