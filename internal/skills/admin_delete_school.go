package skills

import (
	"context"
	"encoding/json"
	"fmt"

	"luminous/internal/httpclient"
	"luminous/internal/skill"
)

// AdminDeleteSchoolSkill implements the admin_delete_school MCP tool.
// It calls DELETE {API_BASE_URL}/api/v1/admin/schools/:code (requires API_TOKEN).
type AdminDeleteSchoolSkill struct {
	client *httpclient.Client
}

// NewAdminDeleteSchool creates an AdminDeleteSchoolSkill backed by the given HTTP client.
func NewAdminDeleteSchool(client *httpclient.Client) *AdminDeleteSchoolSkill {
	return &AdminDeleteSchoolSkill{client: client}
}

// Definition returns the MCP tool metadata for admin_delete_school.
func (s *AdminDeleteSchoolSkill) Definition() skill.ToolDef {
	return skill.ToolDef{
		Name:        "admin_delete_school",
		Description: "管理员接口：删除一所学校。需要提供学校代码。删除操作不可撤销，请谨慎使用。",
		InputSchema: skill.InputSchema{
			Type: "object",
			Properties: map[string]skill.PropertyDef{
				"code": {
					Type:        "string",
					Description: "要删除的学校代码",
				},
			},
			Required: []string{"code"},
		},
	}
}

// Execute calls the Luminous admin API to delete a school.
func (s *AdminDeleteSchoolSkill) Execute(ctx context.Context, args map[string]any) (string, error) {
	code, ok := args["code"]
	if !ok || code == nil {
		return "", fmt.Errorf("缺少必填参数 code")
	}
	codeStr, ok := code.(string)
	if !ok || codeStr == "" {
		return "", fmt.Errorf("code 必须是字符串类型")
	}

	respBody, err := s.client.Delete(ctx, "api/v1/admin/schools/"+codeStr)
	if err != nil {
		return "", fmt.Errorf("删除学校 %s 失败: %w", codeStr, err)
	}

	var parsed any
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("后端返回了无效的 JSON: %w", err)
	}

	result, err := json.Marshal(parsed)
	if err != nil {
		return "", fmt.Errorf("序列化响应失败: %w", err)
	}

	return string(result), nil
}
