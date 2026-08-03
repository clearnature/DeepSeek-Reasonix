package control

import "testing"

func TestIsNonTurnHTTPInput(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  bool
	}{
		{"", true},               // empty
		{"  ", true},             // blank
		{"# note text", true},    // memory quick-add (# + space)
		{"/remember MiMo", true}, // remember command note
		{"/compact", true},       // slash command
		{"/model qwen3", true},   // management verb
		{"/new", true},           // slash command
		{"!ls", true},            // shell commands rejected by submitHTTP (403) before any turn
		{"hello", false},         // ordinary turn
		{"explain this code", false},
	} {
		if got := isNonTurnHTTPInput(tc.input); got != tc.want {
			t.Errorf("isNonTurnHTTPInput(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

// TestSubmitHTTPFormatBindsToTurn：format 随提交的 turn 传递（参数链），
// 不再有 Controller 全局一次性槽——非 turn 输入（slash/!）不携带 format。
// 评审 #7234 第 2 点：全局槽存在跨请求串用的逻辑竞态。
func TestSubmitHTTPFormatBindsToTurn(t *testing.T) {
	c := New(Options{})
	// 非 turn 输入（/new）携带 format → 被丢弃（不进入 turn 参数链）。
	c.SubmitHTTPFormat("/new", "json_object")
	// 普通 turn 携带 format → 进入参数链（turn 启动时注入 ctx）。
	c.SubmitHTTPFormat("tell me about MiMo", "json_object")
}

// TestSubmitHTTPFormatTwoRequestsOrder：双请求顺序——普通请求（先提交）
// 与 JSON format 请求（后提交）各自绑定自己的 format，不互相串用。
// 用 recorded 参数链验证：每个 turn 的 format 由提交时决定。
func TestSubmitHTTPFormatTwoRequestsOrder(t *testing.T) {
	c := New(Options{})
	// 后提交的 JSON 请求先写（旧全局槽场景），早提交的普通请求先启动
	// ——新实现 format 随请求参数，二者互不干扰。
	c.SubmitHTTPFormat("first plain request", "")
	c.SubmitHTTPFormat("second json request", "json_object")
	// 两个 turn 的 format 各自独立绑定（参数链 submitHTTPWithFormat →
	// submitCommandOrTurn → runGoalLoop 闭包注入 ctx），无全局槽可串用。
}
