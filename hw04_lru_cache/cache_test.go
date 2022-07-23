package hw04lrucache

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	t.Run("empty cache", func(t *testing.T) {
		c := NewCache(10)

		_, ok := c.Get("aaa")
		require.False(t, ok)

		_, ok = c.Get("bbb")
		require.False(t, ok)
	})

	t.Run("simple", func(t *testing.T) {
		c := NewCache(5)

		wasInCache := c.Set("aaa", 100) // 100
		require.False(t, wasInCache)

		wasInCache = c.Set("bbb", 200) // 200 100
		require.False(t, wasInCache)

		val, ok := c.Get("aaa") // 100 200
		require.True(t, ok)
		require.Equal(t, 100, val)

		val, ok = c.Get("bbb") // 200 100
		require.True(t, ok)
		require.Equal(t, 200, val)

		wasInCache = c.Set("aaa", 300) // 300 200
		require.True(t, wasInCache)

		val, ok = c.Get("aaa") // 300 200
		require.True(t, ok)
		require.Equal(t, 300, val)

		val, ok = c.Get("ccc") // 300 200
		require.False(t, ok)
		require.Nil(t, val)
	})

	t.Run("purge logic", func(t *testing.T) {
		c := NewCache(3)
		c.Set("1", 100)
		c.Set("2", 200)
		c.Set("3", 300)
		actual, ok := c.Get("3")
		require.Equal(t, 300, actual)
		require.True(t, ok)

		c.Clear()
		actual, ok = c.Get("3")
		require.Equal(t, nil, actual)
		require.False(t, ok)
	})
}

func TestCacheMultithreading(t *testing.T) {
	c := NewCache(10)
	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 1_000_000; i++ {
			c.Set(Key(strconv.Itoa(i)), i)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1_000_000; i++ {
			c.Get(Key(strconv.Itoa(rand.Intn(1_000_000))))
		}
	}()

	wg.Wait()
}

func TestAdditionalTests(t *testing.T) {
	t.Run("add items over capacity", func(t *testing.T) {
		c := NewCache(3)
		c.Set("100", 100) // [100]
		c.Set("200", 200) // [200 100]
		c.Set("300", 300) // [300 200 100]
		c.Set("400", 400) // [400 300 200]
		res, ok := c.Get("100")
		require.False(t, ok)
		require.Equal(t, nil, res)
		c.Set("500", 500) // [500 400 300]
		res, ok = c.Get("200")
		require.False(t, ok)
		require.Equal(t, nil, res)
	})

	t.Run("old value will be removed", func(t *testing.T) {
		c := NewCache(3)
		c.Set("100", 100)
		c.Set("200", 200)
		c.Set("300", 300) // [300 200 100]

		c.Set("200", 201) // [201 300 100]
		c.Set("100", 101) // [101 201 300]
		c.Get("200")      // [201 101 300]
		c.Get("300")      // [300 201 101]
		c.Set("300", 301) // [301 201 101]
		c.Set("101", 102) // [102 301 201]

		c.Set("400", 400) // [400 102 301]

		res, ok := c.Get("200")
		require.False(t, ok)
		require.Equal(t, nil, res)
	})
}
