package responses

import (
	"encoding/json"
	"fmt"
	"strings"
)

// retrievalPromptBase 是并行检索子代理的公共指令（fable5 优化）：
// 检索策略（缓存优先）、质量要求（权威来源/排除营销煽动）、输出约束
// （facts/sources 数量、confidence 语义）。两处任务生成器共用。
func retrievalPromptBase() string {
	return "你是并行检索子代理，负责高质量信息检索。\n" +
		"检索策略：1) 先调用 retrieve_info 查询本地知识缓存（零成本，命中直接采用）；" +
		"2) 未命中再用 web_search 检索；3) 优先权威来源（政府/官方机构/主流媒体/学术），" +
		"排除营销话术（最有效/零风险/限时抢购/专家推荐）与煽动性内容（恐慌/末日/必须转发）。\n" +
		"输出约束：facts 3-8 条核心事实（涉及时效的信息标注时间）；sources 最多 5 个；" +
		"confidence 0-1（来源越权威、交叉验证越多，可信度越高）。\n"
}

// fleet_plan.go：把 info-frame 接入 fleet 并行子代理（2026-08-03 方案 B）。
// 模型调用 fleet 工具前，先用 BuildFleetRetrievalTasks 展开检索计划为
// 并行子代理任务（每任务 = 场景×语言×四维 一帧），子代理各自用
// web_search 检索并返回 InfoFrame JSON，最后 MergeFrames 拼图。

// FleetTaskSpec is one parallel sub-agent retrieval task.
type FleetTaskSpec struct {
	Prompt      string   `json:"prompt"`
	Description string   `json:"description"`
	ReadOnly    bool     `json:"read_only"`
	Tools       []string `json:"tools"`
	MaxSteps    int      `json:"max_steps"`
}

// BuildFleetRetrievalTasks expands a research plan into parallel fleet tasks.
// For each (domain × language) it emits one sub-agent task whose prompt asks
// for a structured InfoFrame JSON back. 默认覆盖：全部语言 × 指定场景。
// depth 控制四维查询模板（fact 必含，其余按深度）。
func BuildFleetRetrievalTasks(topic string, depth ResearchDepth, langs []string, domains []InfoDomain) []FleetTaskSpec {
	if len(langs) == 0 {
		langs = []string{"zh", "en"}
	}
	if len(domains) == 0 {
		domains = []InfoDomain{DomainGeneral}
	}
	plan := PlanResearch(topic, depth)

	var tasks []FleetTaskSpec
	for _, d := range domains {
		audiences := AudiencesFor(d)
		audDesc := ""
		if len(audiences) > 0 {
			audDesc = "（该场景典型人群: " + strings.Join(audienceNames(audiences), "、") + "）"
		}
		for _, lang := range langs {
			// 该帧的检索查询：四维模板 + 场景提示 + 语言后缀
			queries := make([]string, 0, len(plan.Queries))
			for _, q := range plan.Queries {
				queries = append(queries, SceneQuery(q.Query, d, lang))
			}
			prompt := retrievalPromptBase() + fmt.Sprintf(
				"本次任务：主题「%s」的【%s】场景%s【%s】语言帧。\n"+
					"请检索以下查询（可合并为 1-3 次搜索）：\n%s\n\n"+
					"只输出一个 JSON 对象（InfoFrame 格式），不要多余文字：\n"+
					`{"domain":"%s","language":"%s","topic":"%s","facts":["核心事实1","核心事实2"],"sources":[{"title":"来源名","url":"https://..."}],"confidence":0.0}`,
				topic, domainName(d), audDesc, lang,
				"- "+strings.Join(queries, "\n- "),
				d, lang, topic)
			tasks = append(tasks, FleetTaskSpec{
				Prompt:      prompt,
				Description: fmt.Sprintf("检索 %s/%s", domainName(d), lang),
				ReadOnly:    true,
				Tools:       []string{"web_search", "retrieve_info", "web_fetch"},
				MaxSteps:    5,
			})
		}
	}
	return tasks
}

func domainName(d InfoDomain) string {
	switch d {
	case DomainEconomic:
		return "经济"
	case DomainIndustrial:
		return "工业"
	case DomainCode:
		return "代码"
	case DomainStudent:
		return "学生"
	case DomainResearch:
		return "科研"
	default:
		return "通用"
	}
}

func audienceNames(as []SceneAudience) []string {
	names := map[SceneAudience]string{
		AudInvestor: "投资者", AudAnalyst: "分析师", AudEntrepreneur: "创业者", AudWorker: "从业者",
		AudEngineer: "工程师", AudSupplyChain: "供应链", AudFactory: "工厂",
		AudDeveloper: "开发者", AudArchitect: "架构师", AudDevOps: "运维",
		AudUndergrad: "本科生", AudPostgrad: "研究生", AudJobSeeker: "求职者", AudAbroad: "留学",
		AudScholar: "学者", AudResearcher: "研究员", AudReviewer: "审稿人",
	}
	out := make([]string, 0, len(as))
	for _, a := range as {
		if n, ok := names[a]; ok {
			out = append(out, n)
		}
	}
	return out
}

// ParseInfoFrame decodes a sub-agent's JSON reply into an InfoFrame.
func ParseInfoFrame(raw []byte) (*InfoFrame, error) {
	var f InfoFrame
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse info frame: %w", err)
	}
	if f.Topic == "" {
		return nil, fmt.Errorf("parse info frame: empty topic")
	}
	return &f, nil
}

// AssembleFrameView runs the parallel-retrieval contract: parse every
// sub-agent reply and merge into the 信息拼图.
func AssembleFrameView(topic string, rawReplies [][]byte) FrameView {
	frames := make([]*InfoFrame, 0, len(rawReplies))
	for _, raw := range rawReplies {
		if f, err := ParseInfoFrame(raw); err == nil {
			frames = append(frames, f)
		}
	}
	return MergeFrames(topic, frames)
}
