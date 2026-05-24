package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type EmbyAdmin interface {
	DisableUser(ctx context.Context, embyUserID string) error
	DeleteUser(ctx context.Context, embyUserID string) error
}

type EmbyClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewEmbyClient(cfg EmbyConfig) *EmbyClient {
	return &EmbyClient{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  cfg.APIKey,
		client:  &http.Client{Timeout: cfg.Timeout},
	}
}

func (c *EmbyClient) DisableUser(ctx context.Context, embyUserID string) error {
	if c.baseURL == "" || c.apiKey == "" {
		return fmt.Errorf("emby client is not configured")
	}

	// Emby 的禁用动作走用户 Policy 更新接口。
	// 实际生产环境建议先读取当前 Policy，再仅将 IsDisabled 改为 true，避免覆盖库权限、设备限制等既有策略。
	payload := map[string]any{"IsDisabled": true}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/Users/"+embyUserID+"/Policy", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Emby-Token", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("emby disable user returned %s", resp.Status)
	}
	return nil
}

func (c *EmbyClient) DeleteUser(ctx context.Context, embyUserID string) error {
	if c.baseURL == "" || c.apiKey == "" {
		return fmt.Errorf("emby client is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/Users/"+embyUserID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Emby-Token", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("emby delete user returned %s", resp.Status)
	}
	return nil
}
