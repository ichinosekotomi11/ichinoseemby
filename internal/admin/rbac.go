package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

type contextKey string

const userContextKey contextKey = "eden.user"

var (
	ErrFeatureDisabled = errors.New("feature is disabled")
	ErrLevelDenied     = errors.New("user level cannot access this feature")
)

func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(userContextKey).(User)
	return user, ok
}

func AuthorizeFeature(store *Store, feature string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFromContext(r.Context())
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "未登录或登录状态已失效")
			return
		}

		var policy FeaturePolicy
		err := store.View(func(state State) error {
			policies := state.SystemConfig.Features
			if policies == nil {
				policies = DefaultFeaturePolicies()
			}
			var exists bool
			policy, exists = policies[feature]
			if !exists {
				return ErrFeatureDisabled
			}
			return nil
		})
		if err != nil {
			writeJSONError(w, http.StatusForbidden, "功能未开放")
			return
		}
		if err := policy.Allows(user.Level); err != nil {
			writeJSONError(w, http.StatusForbidden, "当前用户等级无权使用该功能")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (p FeaturePolicy) Allows(level UserLevel) error {
	if !p.Enabled {
		return ErrFeatureDisabled
	}
	for _, allowed := range p.AllowedLevels {
		if allowed == level {
			return nil
		}
	}
	return ErrLevelDenied
}

func (p FeaturePolicy) CostFor(level UserLevel) int64 {
	multiplier := int64(1)
	if p.LevelCostMultiplier != nil {
		if configured, ok := p.LevelCostMultiplier[level]; ok {
			multiplier = configured
		}
	}
	return p.CoinCost * multiplier
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
