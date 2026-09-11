// The trajectory pane: what ran, when, and how much of that the record covers.
// Its own catalogue for the reason en_graph has one — en.ts sits at the
// file-size ceiling, and a screen's worth of wording is read together.
export const EN_TRAJECTORY: Record<string, string> = {
  "时间轴": "Timeline",
  "已保存至 {path}": "Saved to {path}",
  "已下载 {name}": "Downloaded {name}",
  // What the rows cover. The three the host distinguishes plus the one the page
  // is in until it has been told: absence is not one of them.
  "记录已达容量上限而停止 —— 最后一行之后的内容未被保留":
    "The record stopped at its size limit — whatever happened after the last row was not kept",
  "本次运行未记录轨迹 —— 以下仅为本次连接观察到的实时事件":
    "Nothing recorded this run — what follows is only the live events this connection saw",
  "完整记录 · 重新进入会话将原样重建": "The whole record · reopening the session rebuilds it as it is",
  "尚未读取该轨迹的覆盖范围": "How much this trajectory covers has not been read yet",
};
