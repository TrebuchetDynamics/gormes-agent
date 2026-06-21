package parsers

import "sort"

type ToolCallParserStatus string

const (
	ToolCallParserStatusMapped    ToolCallParserStatus = "mapped"
	ToolCallParserStatusRowBacked ToolCallParserStatus = "row_backed"
)

type ToolCallParserEntry struct {
	UpstreamFile       string
	ModelFamily        string
	ExpectedInputStyle string
	Status             ToolCallParserStatus
	TargetGoPackage    string
	GoldenFixtures     []string
}

func ToolCallParserManifest() []ToolCallParserEntry {
	entries := []ToolCallParserEntry{
		{
			UpstreamFile:       "deepseek_v3_1_parser.py",
			ModelFamily:        "deepseek_v3_1",
			ExpectedInputStyle: "fullwidth_token_pair_no_json_block",
			Status:             ToolCallParserStatusMapped,
			TargetGoPackage:    "internal/llm/repair/toolcallparsers/deepseekv31",
			GoldenFixtures: []string{
				"deepseek_v3_1_basic.json",
				"deepseek_v3_1_malformed.json",
			},
		},
		{
			UpstreamFile:       "deepseek_v3_parser.py",
			ModelFamily:        "deepseek_v3",
			ExpectedInputStyle: "fullwidth_token_pair_with_json_codeblock",
			Status:             ToolCallParserStatusMapped,
			TargetGoPackage:    "internal/llm/repair/toolcallparsers/deepseekv3",
			GoldenFixtures: []string{
				"deepseek_v3_basic.json",
				"deepseek_v3_malformed.json",
			},
		},
		{
			UpstreamFile:       "glm45_parser.py",
			ModelFamily:        "glm_4_5_moe",
			ExpectedInputStyle: "tool_call_arg_key_arg_value_xml",
			Status:             ToolCallParserStatusMapped,
			TargetGoPackage:    "internal/llm/repair/toolcallparsers/glm45",
			GoldenFixtures: []string{
				"glm45_basic.json",
				"glm45_malformed.json",
			},
		},
		{
			UpstreamFile:       "glm47_parser.py",
			ModelFamily:        "glm_4_7",
			ExpectedInputStyle: "tool_call_arg_key_arg_value_xml_newline_tolerant",
			Status:             ToolCallParserStatusMapped,
			TargetGoPackage:    "internal/llm/repair/toolcallparsers/glm47",
			GoldenFixtures: []string{
				"glm47_basic.json",
				"glm47_malformed.json",
			},
		},
		{
			UpstreamFile:       "hermes_parser.py",
			ModelFamily:        "hermes",
			ExpectedInputStyle: "tool_call_xml_json_body",
			Status:             ToolCallParserStatusMapped,
			TargetGoPackage:    "internal/llm/repair/toolcallparsers/hermesxml",
			GoldenFixtures: []string{
				"hermes_basic.json",
				"hermes_malformed.json",
				"hermes_multi.json",
			},
		},
		{
			UpstreamFile:       "kimi_k2_parser.py",
			ModelFamily:        "kimi_k2",
			ExpectedInputStyle: "tool_calls_section_begin_end_pair",
			Status:             ToolCallParserStatusMapped,
			TargetGoPackage:    "internal/llm/repair/toolcallparsers/kimik2",
			GoldenFixtures: []string{
				"kimik2_basic.json",
				"kimik2_malformed.json",
			},
		},
		{
			UpstreamFile:       "llama_parser.py",
			ModelFamily:        "llama_3_x_4",
			ExpectedInputStyle: "raw_json_object_with_optional_python_tag",
			Status:             ToolCallParserStatusMapped,
			TargetGoPackage:    "internal/llm/repair/toolcallparsers/llama",
			GoldenFixtures: []string{
				"llama_basic.json",
				"llama_malformed.json",
			},
		},
		{
			UpstreamFile:       "longcat_parser.py",
			ModelFamily:        "longcat_flash",
			ExpectedInputStyle: "longcat_tool_call_xml_json_body",
			Status:             ToolCallParserStatusMapped,
			TargetGoPackage:    "internal/llm/repair/toolcallparsers/longcat",
			GoldenFixtures: []string{
				"longcat_basic.json",
				"longcat_malformed.json",
			},
		},
		{
			UpstreamFile:       "mistral_parser.py",
			ModelFamily:        "mistral",
			ExpectedInputStyle: "tool_calls_bot_token_pre_v11_or_v11",
			Status:             ToolCallParserStatusMapped,
			TargetGoPackage:    "internal/llm/repair/toolcallparsers/mistral",
			GoldenFixtures: []string{
				"mistral_basic.json",
				"mistral_malformed.json",
			},
		},
		{
			UpstreamFile:       "qwen3_coder_parser.py",
			ModelFamily:        "qwen3_coder",
			ExpectedInputStyle: "tool_call_function_parameter_xml",
			Status:             ToolCallParserStatusMapped,
			TargetGoPackage:    "internal/llm/repair/toolcallparsers/qwen3coder",
			GoldenFixtures: []string{
				"qwen3coder_basic.json",
				"qwen3coder_malformed.json",
			},
		},
		{
			UpstreamFile:       "qwen_parser.py",
			ModelFamily:        "qwen_2_5",
			ExpectedInputStyle: "tool_call_xml_json_body",
			Status:             ToolCallParserStatusMapped,
			TargetGoPackage:    "internal/llm/repair/toolcallparsers/qwen",
			GoldenFixtures: []string{
				"qwen_basic.json",
				"qwen_malformed.json",
			},
		},
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].UpstreamFile < entries[j].UpstreamFile
	})
	out := make([]ToolCallParserEntry, len(entries))
	copy(out, entries)
	return out
}
