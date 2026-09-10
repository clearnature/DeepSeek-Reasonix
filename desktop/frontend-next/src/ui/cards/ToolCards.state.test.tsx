// @vitest-environment jsdom
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { ReadsCard } from "./ReadsCard";
import { ToolCard } from "./ToolCard";

afterEach(cleanup);

describe("tool outcome cards", () => {
  it("keeps a failed read visible after reads are folded", () => {
    const { container } = render(<ReadsCard tools={[
      { id: "ok", name: "read_file", args: '{"path":"a.ts"}', output: "1→ok", readOnly: true },
      { id: "bad", name: "read_file", args: '{"path":"b.ts"}', err: "no such file", readOnly: true },
    ]} />);
    expect(screen.getByText("1 项失败")).toBeTruthy();
    expect(container.querySelector('[data-call="bad"][data-bad]')).toBeTruthy();
  });

  it("marks an err-only tool result as failed in its heading", () => {
    render(<ToolCard tool={{ id: "bad", name: "bash", err: "boom", readOnly: false }} running={false} />);
    expect(screen.getByText("失败")).toBeTruthy();
    expect(screen.getByText("boom")).toBeTruthy();
  });
});
