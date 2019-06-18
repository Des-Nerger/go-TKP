package main

type stack []state

type state int
const (
	topLevel state = iota
	topLevel1
	ДИАЛ
	innerLevel
)

func (r *stack) push(state state) {
	*r = append(*r, state)
}

func (r *stack) removeTop() {
	*r = (*r)[ : r.indexOfTop()]
}

func (r *stack) peek() state {
	return (*r)[r.indexOfTop()]
}

func (r *stack) replaceTop(state state) {
	(*r)[r.indexOfTop()] = state
}

func (r *stack) indexOfTop() int {
	return len(*r)-1
}
