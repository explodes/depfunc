package depfunc

import "bytes"

type StringSet map[string]struct{}

func (ss StringSet) Contains(s string) bool {
	_, exists := ss[s]
	return exists
}

func (ss StringSet) Add(s string) {
	ss[s] = struct{}{}
}

func (ss StringSet) Remove(s string) {
	delete(ss, s)
}

func (ss StringSet) String() string {
	buf := &bytes.Buffer{}
	buf.WriteRune('{')

	i := 1
	for s := range ss {
		buf.WriteRune('"')
		buf.WriteString(s)
		buf.WriteRune('"')
		if i < len(ss) {
			buf.WriteRune(',')
		}
		i++
	}
	buf.WriteRune('}')

	return buf.String()
}

type stringmultimap map[string]StringSet

func (m stringmultimap) Add(key, value string) {
	set := m[key]
	if set == nil {
		set = make(StringSet)
		m[key] = set
	}
	set.Add(value)
}

type stringstack struct {
	stack []string
}

func (ss *stringstack) Push(s string) {
	ss.stack = append(ss.stack, s)
}

func (ss *stringstack) Pop() string {
	l := len(ss.stack)
	top, bottom := ss.stack[l-1], ss.stack[:l-1]
	ss.stack[l-1] = ""
	ss.stack = bottom
	return top
}

func (ss *stringstack) Top() string {
	l := len(ss.stack)
	top := ss.stack[l-1]
	return top
}
