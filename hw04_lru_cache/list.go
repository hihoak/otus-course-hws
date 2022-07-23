package hw04lrucache

type List interface {
	Len() int
	Front() *ListItem
	Back() *ListItem
	PushFront(v interface{}) *ListItem
	PushBack(v interface{}) *ListItem
	Remove(i *ListItem)
	MoveToFront(i *ListItem)
}

type ListItem struct {
	Value interface{}
	Next  *ListItem
	Prev  *ListItem
}

type list struct {
	length int
	front  *ListItem
	back   *ListItem
}

func (l *list) Len() int {
	return l.length
}

func (l *list) Front() *ListItem {
	return l.front
}

func (l *list) Back() *ListItem {
	return l.back
}

func (l *list) PushFront(v interface{}) *ListItem {
	newListItem := &ListItem{
		Value: v,
		Next:  l.front,
		Prev:  nil,
	}
	if l.length != 0 {
		l.front.Prev = newListItem
	}
	if l.length == 0 {
		// если длина листа нулевая, то back и front должны указывать на один и тот же ListItem
		l.back = newListItem
	}
	l.front = newListItem
	l.length++
	return newListItem
}

func (l *list) PushBack(v interface{}) *ListItem {
	newListItem := &ListItem{
		Value: v,
		Next:  nil,
		Prev:  l.back,
	}
	if l.length != 0 {
		l.back.Next = newListItem
	}
	if l.length == 0 {
		// если длина листа нулевая, то back и front должны указывать на один и тот же ListItem
		l.front = newListItem
	}
	l.back = newListItem
	l.length++
	return newListItem
}

//nolint:gocritic,wastedassign
// gocritic - don't know how to simplify if-else statement
// wastedassign - false positive to 'i = nil'.
func (l *list) Remove(i *ListItem) {
	// по условиям задачи мы вызываем методы Remove и MoveToFront над существующими ListItem,
	// но все же хочу добавить данное условие
	if i == nil {
		return
	}

	if l.front == i && l.back == i { // если list состоит только из одного этого элемента
		l.front, l.back = nil, nil
	} else if l.front == i { // i является front листа
		i.Next.Prev = nil
		l.front = i.Next
	} else if l.back == i { // i является back листа
		i.Prev.Next = nil
		l.back = i.Prev
	} else { // i находится где-то в середине листа
		i.Next.Prev = i.Prev
		i.Prev.Next = i.Next
	}
	i = nil
	l.length--
}

func (l *list) MoveToFront(i *ListItem) {
	if i == nil || l.front == i {
		return
	}

	if l.back == i {
		i.Prev.Next = nil
		l.back = i.Prev
	} else {
		// находится внутри листа привязываем соседние ListItem
		i.Prev.Next = i.Next
		i.Next.Prev = i.Prev
	}
	i.Prev = nil

	i.Next = l.front
	l.front.Prev = i
	l.front = i
}

func NewList() List {
	return &list{}
}
