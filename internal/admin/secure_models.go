package admin

import "time"

type SQLUser struct {
	ID               string          `db:"id"`
	Username         string          `db:"username"`
	Email            string          `db:"email"`
	Level            UserLevel       `db:"level"`
	PasswordHash     string          `db:"password_hash"`
	TelegramID        EncryptedString `db:"telegram_id_enc"`
	EmbyUsername      EncryptedString `db:"emby_username_enc"`
	EmbyPassword      EncryptedString `db:"emby_password_enc"`
	EmbyUserID        string          `db:"emby_user_id"`
	Coins            int64           `db:"coins"`
	LastChargeAt      time.Time       `db:"last_charge_at"`
	ExpiresAt         time.Time       `db:"expires_at"`
	EmbyDisabledAt    time.Time       `db:"emby_disabled_at"`
	CreatedAt         time.Time       `db:"created_at"`
	UpdatedAt         time.Time       `db:"updated_at"`
}

type SQLSystemConfig struct {
	Key       string    `db:"key"`
	ValueJSON string    `db:"value_json"`
	Encrypted bool      `db:"encrypted"`
	UpdatedAt time.Time `db:"updated_at"`
}

type SQLSensitiveConfig struct {
	Key       string          `db:"key"`
	Value     EncryptedString `db:"value_enc"`
	UpdatedAt time.Time       `db:"updated_at"`
}
