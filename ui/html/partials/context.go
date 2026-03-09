package partials

import (
	"context"
	"fmt"
	"strings"
)

type ContextKey string

const (
	UserNameKey  ContextKey = "profileUserName"
	UserEmailKey ContextKey = "profileUserEmail"
)

func GetUserName(ctx context.Context) string {
	name, _ := ctx.Value(UserNameKey).(string)
	return name
}

func GetUserEmail(ctx context.Context) string {
	email, _ := ctx.Value(UserEmailKey).(string)
	return email
}

func GetInitials(ctx context.Context) string {
	name := GetUserName(ctx)
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "?"
	}
	if len(parts) == 1 {
		r := []rune(parts[0])
		return strings.ToUpper(string(r[0:1]))
	}
	first := []rune(parts[0])
	last := []rune(parts[len(parts)-1])
	return strings.ToUpper(string(first[0:1]) + string(last[0:1]))
}

func GetAvatarColor(ctx context.Context) string {
	name := GetUserName(ctx)
	var hash int
	for _, c := range name {
		hash = int(c) + ((hash << 5) - hash)
	}
	hue := hash % 360
	if hue < 0 {
		hue += 360
	}
	return fmt.Sprintf("hsl(%d, 65%%, 50%%)", hue)
}
