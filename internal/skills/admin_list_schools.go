package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"luminous/internal/httpclient"
	"luminous/internal/skill"
)

// AdminListSchoolsSkill implements the admin_list_schools MCP tool.
// It calls GET {API_BASE_URL}/api/v1/admin/schools (requires API_TOKEN).
type AdminListSchoolsSkill struct {
	client *httpclient.Client
}

// NewAdminListSchools creates an AdminListSchoolsSkill backed by the given HTTP client.
func NewAdminListSchools(client *httpclient.Client) *AdminListSchoolsSkill {
	return &AdminListSchoolsSkill{client: client}
}

// Definition returns the MCP tool metadata for admin_list_schools.
func (s *AdminListSchoolsSkill) Definition() skill.ToolDef {
	return skill.ToolDef{
		Name:        "admin_list_schools",
		Description: "管理员接口：列出所有学校（含已启用和未启用），支持分页。当需要查看完整的学校管理列表（包括禁用学校）时使用。",
		InputSchema: skill.InputSchema{
			Type: "object",
			Properties: map[string]skill.PropertyDef{
				"page": {
					Type:        "integer",
					Description: "页码，从 1 开始，默认 1",
				},
				"page_size": {
					Type:        "integer",
					Description: "每页数量，默认 50，最大 200",
				},
			},
		},
	}
}

// Execute calls the Luminous admin API and returns the paginated school list.
func (s *AdminListSchoolsSkill) Execute(ctx context.Context, args map[string]any) (string, error) {
	page := 1
	pageSize := 50

	if v, ok := args["page"]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			page = int(n)
		case int:
			page = n
		case string:
			var err error
			page, err = strconv.Atoi(n)
			if err != nil {
				return "", fmt.Errorf("page 参数必须是整数")
			}
		default:
			return "", fmt.Errorf("page 参数必须是整数")
		}
	}

	if v, ok := args["page_size"]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			pageSize = int(n)
		case int:
			pageSize = n
		case string:
			var err error
			pageSize, err = strconv.Atoi(n)
			if err != nil {
				return "", fmt.Errorf("page_size 参数必须是整数")
			}
		default:
			return "", fmt.Errorf("page_size 参数必须是整数")
		}
	}

	path := fmt.Sprintf("api/v1/admin/schools?page=%d&page_size=%d", page, pageSize)
	body, err := s.client.Get(ctx, path)
	if err != nil {
		return "", fmt.Errorf("获取学校管理列表失败: %w", err)
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
