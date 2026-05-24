package admin

import (
	"encoding/json"
	"net/http"
	"time"
)

type Service struct {
	cfg   Config
	store *Store
	emby  EmbyAdmin
}

func NewService(cfg Config, store *Store, emby EmbyAdmin) *Service {
	if cfg.Clock == nil {
		cfg.Clock = func() time.Time { return time.Now() }
	}
	return &Service{cfg: cfg, store: store, emby: emby}
}

func (s *Service) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /api/admin/security-design", s.requireAdmin(s.securityDesign))
	mux.HandleFunc("GET /api/admin/features", s.requireAdmin(s.listFeatures))
	mux.HandleFunc("PUT /api/admin/features", s.requireAdmin(s.updateFeatures))
	mux.Handle("POST /api/checkin", AuthorizeFeature(s.store, "checkin.daily", http.HandlerFunc(s.checkin)))
	return mux
}

func (s *Service) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Service) securityDesign(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"levels": []string{"A: 白名单", "B: 已绑定 Emby", "C: 未绑定或过期"},
		"crypto": "AES-256-GCM for reversible sensitive fields; bcrypt for local passwords; JWT payload keeps only sub/lvl/iat/exp",
	})
}

func (s *Service) listFeatures(w http.ResponseWriter, _ *http.Request) {
	_ = s.store.View(func(state State) error {
		writeJSON(w, http.StatusOK, state.SystemConfig.Features)
		return nil
	})
}

func (s *Service) updateFeatures(w http.ResponseWriter, r *http.Request) {
	var policies map[string]FeaturePolicy
	if err := json.NewDecoder(r.Body).Decode(&policies); err != nil {
		writeJSONError(w, http.StatusBadRequest, "权限配置 JSON 不合法")
		return
	}
	if err := s.store.Update(func(state *State) error {
		state.SystemConfig.Features = policies
		return nil
	}); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "保存权限配置失败")
		return
	}
	writeJSON(w, http.StatusOK, policies)
}

func (s *Service) checkin(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "未登录")
		return
	}
	now := s.cfg.Clock()
	err := s.store.Update(func(state *State) error {
		policy := state.SystemConfig.Features["checkin.daily"]
		reward := policy.LevelReward[user.Level]
		user.Coins += reward
		user.UpdatedAt = now
		state.Users[user.ID] = user
		state.CoinLedger = append(state.CoinLedger, CoinTransaction{
			ID:        newID("coin"),
			UserID:    user.ID,
			Delta:     reward,
			Balance:   user.Coins,
			Reason:    "每日签到",
			Operator:  "system",
			CreatedAt: now,
		})
		writeJSON(w, http.StatusOK, map[string]any{"coins": user.Coins, "reward": reward})
		return nil
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "签到失败")
	}
}

func (s *Service) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.AdminToken == "" || r.Header.Get("X-Admin-Token") != s.cfg.AdminToken {
			writeJSONError(w, http.StatusUnauthorized, "管理员令牌无效")
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
