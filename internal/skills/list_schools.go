package skills

import (
	"context"
	"encoding/json"
	"fmt"

	"luminous/internal/httpclient"
	"luminous/internal/skill"
)

// ListSchoolsSkill implements the list_schools MCP tool.
// It calls GET {API_BASE_URL}/api/v1/schools to retrieve all enabled schools.
type ListSchoolsSkill struct {
	client *httpclient.Client
}

// NewListSchools creates a ListSchoolsSkill backed by the given HTTP client.
func NewListSchools(client *httpclient.Client) *ListSchoolsSkill {
	return &ListSchoolsSkill{client: client}
}

// Definition returns the MCP tool metadata for list_schools.
func (s *ListSchoolsSkill) Definition() skill.ToolDef {
	return skill.ToolDef{
		Name:        "list_schools",
		Description: "列出所有已启用的学校。当用户询问「有哪些学校」「支持哪些学校」「学校列表」时使用。返回学校代码、名称和功能列表。无需参数。",
		InputSchema: skill.InputSchema{
			Type:       "object",
			Properties: map[string]skill.PropertyDef{},
		},
	}
}

// Execute calls the Luminous API and returns the school list as a JSON string.
func (s *ListSchoolsSkill) Execute(ctx context.Context, args map[string]any) (string, error) {
	body, err := s.client.Get(ctx, "api/v1/schools")
	if err != nil {
		return "", fmt.Errorf("获取学校列表失败: %w", err)
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
