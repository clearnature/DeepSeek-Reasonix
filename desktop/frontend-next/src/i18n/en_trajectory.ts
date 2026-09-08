// The trajectory pane: what ran, when, and how much of that the record covers.
// Its own catalogue for the reason en_graph has one — en.ts sits at the
// file-size ceiling, and a screen's worth of wording is read together.
export const EN_TRAJECTORY: Record<string, string> = {
  "时间轴": "Timeline",
  "存到 {path}": "Saved to {path}",
  "已下载 {name}": "Downloaded {name}",
  // What the rows cover. The three the host distinguishes plus the one the page
  // is in until it has been told: absence is not one of them.
  "记录到容量上限就停了 —— 最后一行之后还发生过什么，没有留下":
    "The record stopped at its size limit — whatever happened after the last row was not kept",
  "这次运行没有记录轨迹 —— 下面只是本次连接看到的实时事件":
    "Nothing recorded this run — what follows is only the live events this connection saw",
  "完整记录 · 重进会话会照原样重建": "The whole record · reopening the session rebuilds it as it is",
  "还没读到这份轨迹覆盖了多少": "How much this trajectory covers has not been read yet",
};
