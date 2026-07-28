// Package skills contains concrete MCP tool implementations for the Luminous API.
package skills

import (
	"context"
	"encoding/json"
	"fmt"

	"luminous/internal/httpclient"
	"luminous/internal/skill"
)

// QuerySchoolSkill implements the query_school MCP tool.
// It calls GET {API_BASE_URL}/api/v1/schools/{school_code} to retrieve school info.
type QuerySchoolSkill struct {
	client *httpclient.Client
}

// NewQuerySchool creates a QuerySchoolSkill backed by the given HTTP client.
func NewQuerySchool(client *httpclient.Client) *QuerySchoolSkill {
	return &QuerySchoolSkill{client: client}
}

// Definition returns the MCP tool metadata for query_school.
func (s *QuerySchoolSkill) Definition() skill.ToolDef {
	return skill.ToolDef{
		Name:        "query_school",
		Description: "根据学校代码查询学校详情。调用时机：当用户询问某所学校的基本信息、网站地址、支持的功能列表时使用。传入学校代码（如 XAUAT），返回该学校的名称、网站和教务功能列表。",
		InputSchema: skill.InputSchema{
			Type: "object",
			Properties: map[string]skill.PropertyDef{
				"school_code": {
					Type:        "string",
					Description: "学校代码，通常为大写英文字母缩写，例如 XAUAT（西安建筑科技大学）、XUT（西安理工大学）",
				},
			},
			Required: []string{"school_code"},
		},
	}
}

// Execute calls the Luminous API and returns the school info as a JSON string.
func (s *QuerySchoolSkill) Execute(ctx context.Context, args map[string]any) (string, error) {
	schoolCode, ok := args["school_code"]
	if !ok || schoolCode == nil {
		return "", fmt.Errorf("缺少必填参数 school_code")
	}

	schoolCodeStr, ok := schoolCode.(string)
	if !ok || schoolCodeStr == "" {
		return "", fmt.Errorf("school_code 必须是字符串类型")
	}

	body, err := s.client.Get(ctx, "api/v1/schools/"+schoolCodeStr)
	if err != nil {
		return "", fmt.Errorf("查询学校 %s 失败: %w", schoolCodeStr, err)
	}

	// Pass through the backend JSON, validating it is well-formed.
	var parsed any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("后端返回了无效的 JSON: %w", err)
	}

	// Re-marshal to get compact, valid JSON.
	result, err := json.Marshal(parsed)
	if err != nil {
		return "", fmt.Errorf("序列化响应失败: %w", err)
	}

	return string(result), nil
}
