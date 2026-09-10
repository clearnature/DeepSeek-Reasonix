// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Item } from "../../state/session";
import { ApprovalCard } from "./ApprovalCard";
import { AskCard } from "./AskCard";

afterEach(cleanup);

const pending = () => new Promise<void>((resolve) => setTimeout(resolve, 20));

describe("decision cards", () => {
  it("keeps every approval action locked while the decision is in flight", async () => {
    const item = {
      t: "approval", id: "row", a: { id: "gate", tool: "bash", subject: "run checks" },
    } as Extract<Item, { t: "approval" }>;
    let release = () => {};
    const approve = vi.fn(() => new Promise<void>((resolve) => { release = resolve; }));
    render(<ApprovalCard item={item} onApprove={approve} onPlan={vi.fn(pending)} />);

    await userEvent.click(screen.getByRole("button", { name: "允许这一次" }));
    expect(screen.getByText("正在提交…")).toBeTruthy();
    for (const button of screen.getAllByRole("button")) expect((button as HTMLButtonElement).disabled).toBe(true);
    release();
    await waitFor(() => expect((screen.getByRole("button", { name: "允许这一次" }) as HTMLButtonElement).disabled).toBe(false));
    expect(approve).toHaveBeenCalledTimes(1);
  });

  it("does not invent a recommendation and restores answered tab state", () => {
    const item = {
      t: "ask", id: "row", answered: [["B"]], ask: { id: "ask", questions: [
        { id: "q", header: "方向", prompt: "选哪个？", multi: false, options: [{ label: "A" }, { label: "B" }] },
      ] },
    } as Extract<Item, { t: "ask" }>;
    const { container } = render(<AskCard item={item} onAnswer={vi.fn(pending)} />);
    expect(screen.queryByText("推荐")).toBeNull();
    expect(screen.getByRole("button", { name: "B" }).getAttribute("aria-pressed")).toBe("true");
    expect(within(container).getByText(/方向：/).parentElement?.textContent).toContain("B");
  });
});
