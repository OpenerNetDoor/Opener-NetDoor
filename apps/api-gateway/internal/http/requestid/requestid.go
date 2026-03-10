package requestid

import "context"

type contextKey string

const key contextKey = "request_id"

func WithContext(ctx context.Context, rid string) context.Context {
	return context.WithValue(ctx, key, rid)
}

func FromContext(ctx context.Context) string {
	v, _ := ctx.Value(key).(string)
	return v
}
