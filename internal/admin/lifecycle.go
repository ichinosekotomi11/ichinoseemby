package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type DowngradeMode string

const (
	DowngradeDisableEmby DowngradeMode = "disable"
	DowngradeDeleteEmby  DowngradeMode = "delete"
)

type BillingPolicy struct {
	CycleCost     int64
	DowngradeMode DowngradeMode
	Operator      string
}

func EnforceBLevelBilling(ctx context.Context, store *Store, emby EmbyAdmin, cipher *FieldCipher, now time.Time, policy BillingPolicy) error {
	return store.Update(func(state *State) error {
		for id, user := range state.Users {
			// A 级是永久白名单，跳过所有定时删号、扣币和 Emby 处置。
			if user.Level == LevelA {
				continue
			}
			// C 级本来就无 Emby 服务权限，不重复扣费，也不重复触发禁用。
			if user.Level != LevelB {
				continue
			}
			if user.Coins >= policy.CycleCost {
				user.Coins -= policy.CycleCost
				user.UpdatedAt = now
				state.Users[id] = user
				state.CoinLedger = append(state.CoinLedger, CoinTransaction{
					ID:        newID("coin"),
					UserID:    id,
					Delta:     -policy.CycleCost,
					Balance:   user.Coins,
					Reason:    "B级用户周期扣费",
					Operator:  policy.Operator,
					CreatedAt: now,
				})
				continue
			}

			link, hasLink := state.EmbyLinks[id]
			if hasLink && emby != nil {
				// B 级金币不足时，先调用 Emby API 处理外部账号，再提交本地等级变化。
				// 这样可以避免本地已经降级但 Emby 仍可播放的权限漂移。
				// 默认动作建议为禁用账号：POST /Users/{Id}/Policy，并设置 IsDisabled=true。
				// 若站点合规策略要求彻底清理，可切换为 DELETE /Users/{Id}。
				var err error
				switch policy.DowngradeMode {
				case DowngradeDeleteEmby:
					err = emby.DeleteUser(ctx, link.EmbyUserID)
				default:
					err = emby.DisableUser(ctx, link.EmbyUserID)
				}
				if err != nil {
					// Emby 侧失败时不提交本地降级，交给下一个周期重试或人工处理。
					// 这能保证“本地等级状态”和“Emby 实际权限”不会互相说谎。
					return fmt.Errorf("handle emby user %s before downgrade: %w", link.EmbyUserID, err)
				}
			}

			previousLevel := user.Level
			user.Level = LevelC
			user.EmbyDisabledAt = now
			user.UpdatedAt = now
			state.Users[id] = user

			state.CoinLedger = append(state.CoinLedger, CoinTransaction{
				ID:        newID("coin"),
				UserID:    id,
				Delta:     0,
				Balance:   user.Coins,
				Reason:    "金币不足，B级降级为C级",
				Operator:  policy.Operator,
				CreatedAt: now,
			})

			note := map[string]any{
				"previous_level": previousLevel,
				"current_level":  LevelC,
				"coins":          user.Coins,
				"cycle_cost":     policy.CycleCost,
				"emby_user_id":   link.EmbyUserID,
				"emby_action":    policy.DowngradeMode,
				"occurred_at":    now,
			}
			noteJSON, _ := json.Marshal(note)
			encryptedNote := ""
			if cipher != nil {
				encryptedNote, _ = cipher.EncryptString(string(noteJSON))
			}
			state.SecurityEvents = append(state.SecurityEvents, SecurityEvent{
				ID:            newID("sec"),
				UserID:        id,
				Type:          "USER_DOWNGRADED_B_TO_C",
				EncryptedNote: encryptedNote,
				CreatedAt:     now,
			})
		}
		return nil
	})
}

func newID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
