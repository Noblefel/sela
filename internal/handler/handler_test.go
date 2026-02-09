package handler

import (
	"context"
)

type mockSession struct{}

func (ms *mockSession) Destroy(context.Context) error    { return nil }
func (ms *mockSession) Get(context.Context, string) any  { return nil }
func (ms *mockSession) Pop(context.Context, string) any  { return nil }
func (ms *mockSession) Put(context.Context, string, any) {}
