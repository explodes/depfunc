package depfunc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringmultimap_New(t *testing.T) {
	m := make(stringmultimap)

	assert.Len(t, m, 0)
}

func TestStringmultimap_Add(t *testing.T) {
	m := make(stringmultimap)

	m.Add("test", "1")

	assert.Len(t, m, 1)
}

func TestStringmultimap_AddMultiple(t *testing.T) {
	m := make(stringmultimap)

	assert.Len(t, m["test1"], 0)
	assert.Len(t, m["test2"], 0)

	m.Add("test1", "1")
	m.Add("test2", "2")

	assert.Len(t, m, 2)
	assert.Len(t, m["test1"], 1)
	assert.Len(t, m["test2"], 1)
}

func TestStringmultimap_AddSameKey(t *testing.T) {
	m := make(stringmultimap)

	assert.Len(t, m, 0)

	m.Add("test1", "1")
	m.Add("test1", "2")

	assert.Len(t, m, 1)
}

func TestStringmultimap_AddToSet(t *testing.T) {
	m := make(stringmultimap)

	assert.Len(t, m["test1"], 0)

	m.Add("test1", "1")
	m.Add("test1", "2")

	assert.Len(t, m["test1"], 2)
}

func TestStringset_Add(t *testing.T) {
	s := make(StringSet)

	assert.Len(t, s, 0)

	s.Add("a")

	assert.Len(t, s, 1)
}

func TestStringset_Remove(t *testing.T) {
	s := make(StringSet)

	assert.Len(t, s, 0)

	s.Add("a")
	s.Remove("a")

	assert.Len(t, s, 0)
}

func TestStringset_Contains(t *testing.T) {
	s := make(StringSet)

	assert.False(t, s.Contains("a"))

	s.Add("a")

	assert.True(t, s.Contains("a"))
}

func TestStringset_String(t *testing.T) {
	s := make(StringSet)

	out := s.String()

	assert.Equal(t, "{}", out)
}

func TestStringset_String_value(t *testing.T) {
	s := make(StringSet)
	s.Add("a")

	out := s.String()

	assert.Equal(t, `{"a"}`, out)
}

func TestStringset_String_values(t *testing.T) {
	// because of the non-deterministic ordering of map, "a" or "b" could come first
	const (
		aFirst = `{"a","b"}`
		bFirst = `{"b","a"}`
	)
	s := make(StringSet)
	s.Add("a")
	s.Add("b")

	out := s.String()

	if out != aFirst && out != bFirst {
		t.Errorf("unexpected string want %s or %s, got %s", aFirst, bFirst, out)
	}
}

func TestStringstack_Push(t *testing.T) {
	ss := &stringstack{}

	assert.Len(t, ss.stack, 0)

	ss.Push("a")

	assert.Len(t, ss.stack, 1)
}

func TestStringstack_Pop(t *testing.T) {
	ss := &stringstack{}

	ss.Push("a")
	a := ss.Pop()

	assert.Len(t, ss.stack, 0)
	assert.Equal(t, "a", a)
}

func TestStringstack_Top(t *testing.T) {
	ss := &stringstack{}

	ss.Push("a")
	a := ss.Top()

	assert.Equal(t, "a", a)
}
