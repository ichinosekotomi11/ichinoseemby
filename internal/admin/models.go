package admin

import "time"

type UserLevel string

const (
	LevelA UserLevel = "A"
	LevelB UserLevel = "B"
	LevelC UserLevel = "C"
)

type State struct {
	Users              map[string]User         `json:"users"`
	UsersByName        map[string]string       `json:"users_by_name"`
	RegistrationPolicy RegistrationPolicy      `json:"registration_policy"`
	CoinLedger          []CoinTransaction       `json:"coin_ledger"`
	EmbyLinks           map[string]EmbyUserLink `json:"emby_links"`
	SystemConfig        SystemConfig           `json:"system_config"`
	SecurityEvents      []SecurityEvent        `json:"security_events"`
}

type User struct {
	ID               string    `json:"id"`
	Username         string    `json:"username"`
	Email            string    `json:"email,omitempty"`
	Level            UserLevel `json:"level"`
	PasswordHash     string    `json:"password_hash,omitempty"`
	TelegramIDCipher string    `json:"telegram_id_cipher,omitempty"`
	EmbyNameCipher   string    `json:"emby_username_cipher,omitempty"`
	EmbyPassCipher   string    `json:"emby_password_cipher,omitempty"`
	Coins            int64     `json:"coins"`
	EmbyDisabledAt   time.Time `json:"emby_disabled_at,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type RegistrationPolicy struct {
	Enabled           bool                 `json:"enabled"`
	DefaultCoinGrant  int64                `json:"default_coin_grant"`
	Windows           []RegistrationWindow `json:"windows"`
	RequireEmbyCreate bool                 `json:"require_emby_create"`
}

type RegistrationWindow struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	StartsAt       time.Time `json:"starts_at"`
	EndsAt         time.Time `json:"ends_at"`
	Quota          int       `json:"quota"`
	Used           int       `json:"used"`
	DefaultCoins   int64     `json:"default_coins"`
	AllowLocalOnly bool      `json:"allow_local_only"`
}

type CoinTransaction struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Delta     int64     `json:"delta"`
	Balance   int64     `json:"balance"`
	Reason    string    `json:"reason"`
	Operator  string    `json:"operator"`
	CreatedAt time.Time `json:"created_at"`
}

type SecurityEvent struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	Type           string    `json:"type"`
	EncryptedNote  string    `json:"encrypted_note"`
	CreatedAt      time.Time `json:"created_at"`
}

type EmbyUserLink struct {
	UserID     string    `json:"user_id"`
	EmbyUserID string    `json:"emby_user_id"`
	LinkedAt   time.Time `json:"linked_at"`
}

type SystemConfig struct {
	Features map[string]FeaturePolicy `json:"features"`
}

type FeaturePolicy struct {
	Enabled              bool                 `json:"enabled"`
	AllowedLevels        []UserLevel          `json:"allowed_levels"`
	CoinCost             int64                `json:"coin_cost"`
	LevelCostMultiplier  map[UserLevel]int64  `json:"level_cost_multiplier"`
	LevelReward          map[UserLevel]int64  `json:"level_reward"`
	LevelWinProbability  map[UserLevel]float64 `json:"level_win_probability"`
	Metadata             map[string]string    `json:"metadata,omitempty"`
}

func NewState(defaultCoins int64) State {
	return State{
		Users:       map[string]User{},
		UsersByName: map[string]string{},
		RegistrationPolicy: RegistrationPolicy{
			Enabled:          false,
			DefaultCoinGrant: defaultCoins,
			Windows:          []RegistrationWindow{},
		},
		CoinLedger: []CoinTransaction{},
		EmbyLinks:  map[string]EmbyUserLink{},
		SystemConfig: SystemConfig{Features: DefaultFeaturePolicies()},
		SecurityEvents: []SecurityEvent{},
	}
}

func DefaultFeaturePolicies() map[string]FeaturePolicy {
	return map[string]FeaturePolicy{
		"moviepilot.request": {
			Enabled:       true,
			AllowedLevels: []UserLevel{LevelA, LevelB},
			CoinCost:      10,
			LevelCostMultiplier: map[UserLevel]int64{
				LevelA: 0,
				LevelB: 1,
				LevelC: 2,
			},
		},
		"lottery.advanced": {
			Enabled:       true,
			AllowedLevels: []UserLevel{LevelA},
			CoinCost:      20,
			LevelWinProbability: map[UserLevel]float64{
				LevelA: 0.15,
				LevelB: 0.05,
				LevelC: 0.01,
			},
		},
		"registration.apply": {
			Enabled:       true,
			AllowedLevels: []UserLevel{LevelC},
		},
		"checkin.daily": {
			Enabled:       true,
			AllowedLevels: []UserLevel{LevelA, LevelB, LevelC},
			LevelReward: map[UserLevel]int64{
				LevelA: 10,
				LevelB: 5,
				LevelC: 2,
			},
		},
	}
}
