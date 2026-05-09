package dispatch

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func alwaysTrue[T any]() Filter[T]  { return func(_ T) bool { return true } }
func alwaysFalse[T any]() Filter[T] { return func(_ T) bool { return false } }

func TestFilter_And(t *testing.T) {
	t.Run("all true", func(t *testing.T) {
		f := alwaysTrue[int]().And(alwaysTrue[int](), alwaysTrue[int]())
		require.True(t, f(0))
	})
	t.Run("first false", func(t *testing.T) {
		f := alwaysFalse[int]().And(alwaysTrue[int]())
		require.False(t, f(0))
	})
	t.Run("other false", func(t *testing.T) {
		f := alwaysTrue[int]().And(alwaysFalse[int]())
		require.False(t, f(0))
	})
	t.Run("no others — acts as identity", func(t *testing.T) {
		require.True(t, alwaysTrue[int]().And()(0))
		require.False(t, alwaysFalse[int]().And()(0))
	})
}

func TestFilter_Or(t *testing.T) {
	t.Run("first true", func(t *testing.T) {
		f := alwaysTrue[int]().Or(alwaysFalse[int]())
		require.True(t, f(0))
	})
	t.Run("other true", func(t *testing.T) {
		f := alwaysFalse[int]().Or(alwaysTrue[int]())
		require.True(t, f(0))
	})
	t.Run("all false", func(t *testing.T) {
		f := alwaysFalse[int]().Or(alwaysFalse[int]())
		require.False(t, f(0))
	})
	t.Run("no others", func(t *testing.T) {
		require.True(t, alwaysTrue[int]().Or()(0))
		require.False(t, alwaysFalse[int]().Or()(0))
	})
}

func TestFilter_Not(t *testing.T) {
	require.False(t, alwaysTrue[int]().Not()(0))
	require.True(t, alwaysFalse[int]().Not()(0))
}

func TestAll(t *testing.T) {
	t.Run("all true", func(t *testing.T) {
		require.True(t, All(alwaysTrue[int](), alwaysTrue[int]())(0))
	})
	t.Run("one false", func(t *testing.T) {
		require.False(t, All(alwaysTrue[int](), alwaysFalse[int]())(0))
	})
	t.Run("empty — always true", func(t *testing.T) {
		require.True(t, All[int]()(0))
	})
}

func TestAny(t *testing.T) {
	t.Run("one true", func(t *testing.T) {
		require.True(t, Any(alwaysFalse[int](), alwaysTrue[int]())(0))
	})
	t.Run("all false", func(t *testing.T) {
		require.False(t, Any(alwaysFalse[int](), alwaysFalse[int]())(0))
	})
	t.Run("empty — always false", func(t *testing.T) {
		require.False(t, Any[int]()(0))
	})
}

func TestFilter_Composition(t *testing.T) {
	// (true AND false) OR true  == true
	f := alwaysTrue[int]().And(alwaysFalse[int]()).Or(alwaysTrue[int]())
	require.True(t, f(0))

	// NOT (true OR false) == false
	g := alwaysTrue[int]().Or(alwaysFalse[int]()).Not()
	require.False(t, g(0))
}
