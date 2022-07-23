package hw04lrucache

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestList(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		l := NewList()

		require.Equal(t, 0, l.Len())
		require.Nil(t, l.Front())
		require.Nil(t, l.Back())
	})

	t.Run("complex", func(t *testing.T) {
		l := NewList()

		l.PushFront(10) // [10]
		l.PushBack(20)  // [10, 20]
		l.PushBack(30)  // [10, 20, 30]
		require.Equal(t, 3, l.Len())

		middle := l.Front().Next // 20
		l.Remove(middle)         // [10, 30]
		require.Equal(t, 2, l.Len())

		for i, v := range [...]int{40, 50, 60, 70, 80} {
			if i%2 == 0 {
				l.PushFront(v)
			} else {
				l.PushBack(v)
			}
		} // [80, 60, 40, 10, 30, 50, 70]

		require.Equal(t, 7, l.Len())
		require.Equal(t, 80, l.Front().Value)
		require.Equal(t, 70, l.Back().Value)

		l.MoveToFront(l.Front()) // [80, 60, 40, 10, 30, 50, 70]
		l.MoveToFront(l.Back())  // [70, 80, 60, 40, 10, 30, 50]

		elems := make([]int, 0, l.Len())
		for i := l.Front(); i != nil; i = i.Next {
			elems = append(elems, i.Value.(int))
		}
		require.Equal(t, []int{70, 80, 60, 40, 10, 30, 50}, elems)
	})
}

func TestListAdditional(t *testing.T) {
	var nilListItem *ListItem
	cases := []struct {
		name  string
		logic func()
	}{
		{
			name: "test 1",
			logic: func() {
				l := NewList()
				li1 := l.PushFront(10) // [10]
				li2 := l.PushFront(20) // [20 10]
				l.Remove(li2)
				require.Equal(t, nilListItem, li1.Next)
				l.Remove(li1)
				require.Equal(t, 0, l.Len())
			},
		},
		{
			name: "test 2",
			logic: func() {
				l := NewList()
				li10 := l.PushFront(10) // [10]
				li20 := l.PushFront(20) // [20 10]
				li30 := l.PushBack(30)  // [20 10 30]
				require.Equal(t, 3, l.Len())
				l.Remove(li10) // [20 30]
				require.Equal(t, 2, l.Len())
				_ = l.PushFront(40) // [40 20 30]
				l.Remove(li30)      // [40 20]
				l.Remove(li20)      // [40]
				require.Equal(t, 1, l.Len())
			},
		},
		{
			name: "test 3",
			logic: func() {
				l := NewList()
				li10 := l.PushBack(10) // [10]
				li20 := l.PushBack(20) // [10 20]
				li30 := l.PushBack(30) // [10 20 30]
				require.Equal(t, 3, l.Len())
				l.Remove(li10) // [20 30]
				require.Equal(t, 2, l.Len())
				_ = l.PushFront(40) // [40 20 30]
				l.Remove(li30)      // [40 20]
				l.Remove(li20)      // [40]
				require.Equal(t, 1, l.Len())
			},
		},
		{
			name: "test 4",
			logic: func() {
				l := NewList()
				li10 := l.PushBack(10) // [10]
				l.Remove(li10)         // []
				require.Equal(t, 0, l.Len())
				li20 := l.PushBack(20) // [20]
				l.Remove(li20)         // []
				require.Equal(t, 0, l.Len())
				li30 := l.PushBack(30) // [30]
				l.Remove(li30)         // []
				require.Equal(t, 0, l.Len())
			},
		},
		{
			name: "test 5",
			logic: func() {
				l := NewList()
				li10 := l.PushBack(10) // [10]
				l.MoveToFront(li10)    // [10]
				require.Equal(t, li10, l.Front())
				l.MoveToFront(li10) // [10]
				require.Equal(t, li10, l.Front())
				li20 := l.PushBack(20) // [10 20]
				require.Equal(t, li10, l.Front())
				l.MoveToFront(li20) // [20 10]
				require.Equal(t, li20, l.Front())
				li30 := l.PushFront(30) // [30 20 10]
				require.Equal(t, li30, l.Front())
				l.MoveToFront(li20) // [20 30 10]
				require.Equal(t, li20, l.Front())
				require.Equal(t, li30, li20.Next)
			},
		},
		{
			name: "test 6",
			logic: func() {
				l := NewList()
				l.Remove(nil)
				l.MoveToFront(nil)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.logic()
		})
	}
}
