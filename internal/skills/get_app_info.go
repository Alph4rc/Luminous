package skills

import (
	"context"
	"encoding/json"
	"fmt"

	"luminous/internal/httpclient"
	"luminous/internal/skill"
)

// GetAppInfoSkill implements the get_app_info MCP tool.
// It calls GET {API_BASE_URL}/api/v1/app to retrieve app version/release info.
type GetAppInfoSkill struct {
	client *httpclient.Client
}

// NewGetAppInfo creates a GetAppInfoSkill backed by the given HTTP client.
func NewGetAppInfo(client *httpclient.Client) *GetAppInfoSkill {
	return &GetAppInfoSkill{client: client}
}

// Definition returns the MCP tool metadata for get_app_info.
func (s *GetAppInfoSkill) Definition() skill.ToolDef {
	return skill.ToolDef{
		Name:        "get_app_info",
		Description: "获取最新的 App 版本信息和更新公告。当用户询问「最新版本」「App 更新」「更新日志」「下载链接」时使用。无需参数。",
		InputSchema: skill.InputSchema{
			Type:       "object",
			Properties: map[string]skill.PropertyDef{},
		},
	}
}

// Execute calls the Luminous API and returns the app info as a JSON string.
func (s *GetAppInfoSkill) Execute(ctx context.Context, args map[string]any) (string, error) {
	body, err := s.client.Get(ctx, "api/v1/app")
	if err != nil {
		return "", fmt.Errorf("获取 App 信息失败: %w", err)
	}

	var parsed any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("后端返回了无效的 JSON: %w", err)
	}

	result, err := json.Marshal(parsed)
	if err != nil {
		return "", fmt.Errorf("序列化响应失败: %w", err)
	}

	return string(result), nil
}
