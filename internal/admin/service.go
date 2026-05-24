package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

var (
	errUsernameExists = errors.New("username exists")
	errUserNotFound   = errors.New("user not found")
	errInvalidLevel   = errors.New("invalid level")
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
	mux.HandleFunc("GET /", s.dashboard)
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /api/admin/security-design", s.requireAdmin(s.securityDesign))
	mux.HandleFunc("GET /api/admin/users", s.requireAdmin(s.listUsers))
	mux.HandleFunc("POST /api/admin/users", s.requireAdmin(s.createUser))
	mux.HandleFunc("PATCH /api/admin/users/", s.requireAdmin(s.updateUser))
	mux.HandleFunc("GET /api/admin/features", s.requireAdmin(s.listFeatures))
	mux.HandleFunc("PUT /api/admin/features", s.requireAdmin(s.updateFeatures))
	mux.Handle("POST /api/checkin", AuthorizeFeature(s.store, "checkin.daily", http.HandlerFunc(s.checkin)))
	return mux
}

func (s *Service) dashboard(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(adminDashboardHTML))
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

type adminUserView struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email,omitempty"`
	Level          UserLevel `json:"level"`
	Coins          int64     `json:"coins"`
	EmbyDisabledAt time.Time `json:"emby_disabled_at,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (s *Service) listUsers(w http.ResponseWriter, _ *http.Request) {
	users := []adminUserView{}
	_ = s.store.View(func(state State) error {
		for _, user := range state.Users {
			users = append(users, adminUserView{
				ID:             user.ID,
				Username:       user.Username,
				Email:          user.Email,
				Level:          user.Level,
				Coins:          user.Coins,
				EmbyDisabledAt: user.EmbyDisabledAt,
				CreatedAt:      user.CreatedAt,
				UpdatedAt:      user.UpdatedAt,
			})
		}
		return nil
	})
	writeJSON(w, http.StatusOK, users)
}

func (s *Service) createUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string    `json:"username"`
		Email    string    `json:"email"`
		Level    UserLevel `json:"level"`
		Coins    int64     `json:"coins"`
		Password string    `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "用户 JSON 不合法")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		writeJSONError(w, http.StatusBadRequest, "用户名不能为空")
		return
	}
	if req.Level == "" {
		req.Level = LevelC
	}
	if !validLevel(req.Level) {
		writeJSONError(w, http.StatusBadRequest, "用户等级只能是 A、B、C")
		return
	}
	passwordHash := ""
	if req.Password != "" {
		hash, err := HashPassword(req.Password)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "密码加密失败")
			return
		}
		passwordHash = hash
	}
	now := s.cfg.Clock()
	user := User{
		ID:           newID("user"),
		Username:     req.Username,
		Email:        strings.TrimSpace(req.Email),
		Level:        req.Level,
		PasswordHash: passwordHash,
		Coins:        req.Coins,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.store.Update(func(state *State) error {
		if _, exists := state.UsersByName[user.Username]; exists {
			return errUsernameExists
		}
		state.Users[user.ID] = user
		state.UsersByName[user.Username] = user.ID
		if user.Coins != 0 {
			state.CoinLedger = append(state.CoinLedger, CoinTransaction{
				ID:        newID("coin"),
				UserID:    user.ID,
				Delta:     user.Coins,
				Balance:   user.Coins,
				Reason:    "管理员创建用户初始金币",
				Operator:  "admin",
				CreatedAt: now,
			})
		}
		return nil
	}); err != nil {
		if err == errUsernameExists {
			writeJSONError(w, http.StatusConflict, "用户名已存在")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "创建用户失败")
		return
	}
	writeJSON(w, http.StatusCreated, adminUserView{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Level:     user.Level,
		Coins:     user.Coins,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}

func (s *Service) updateUser(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	if userID == "" {
		writeJSONError(w, http.StatusBadRequest, "用户 ID 不能为空")
		return
	}
	var req struct {
		Level     *UserLevel `json:"level"`
		CoinDelta *int64     `json:"coin_delta"`
		Reason    string     `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "用户更新 JSON 不合法")
		return
	}
	now := s.cfg.Clock()
	var updated User
	if err := s.store.Update(func(state *State) error {
		user, exists := state.Users[userID]
		if !exists {
			return errUserNotFound
		}
		if req.Level != nil {
			if !validLevel(*req.Level) {
				return errInvalidLevel
			}
			user.Level = *req.Level
		}
		if req.CoinDelta != nil {
			user.Coins += *req.CoinDelta
			reason := strings.TrimSpace(req.Reason)
			if reason == "" {
				reason = "管理员调整金币"
			}
			state.CoinLedger = append(state.CoinLedger, CoinTransaction{
				ID:        newID("coin"),
				UserID:    user.ID,
				Delta:     *req.CoinDelta,
				Balance:   user.Coins,
				Reason:    reason,
				Operator:  "admin",
				CreatedAt: now,
			})
		}
		user.UpdatedAt = now
		state.Users[userID] = user
		updated = user
		return nil
	}); err != nil {
		switch err {
		case errUserNotFound:
			writeJSONError(w, http.StatusNotFound, "用户不存在")
		case errInvalidLevel:
			writeJSONError(w, http.StatusBadRequest, "用户等级只能是 A、B、C")
		default:
			writeJSONError(w, http.StatusInternalServerError, "更新用户失败")
		}
		return
	}
	writeJSON(w, http.StatusOK, adminUserView{
		ID:             updated.ID,
		Username:       updated.Username,
		Email:          updated.Email,
		Level:          updated.Level,
		Coins:          updated.Coins,
		EmbyDisabledAt: updated.EmbyDisabledAt,
		CreatedAt:      updated.CreatedAt,
		UpdatedAt:      updated.UpdatedAt,
	})
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

func validLevel(level UserLevel) bool {
	return level == LevelA || level == LevelB || level == LevelC
}
