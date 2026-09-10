// @vitest-environment jsdom
import { afterEach, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import "./testkit";
import type { ProviderEntry } from "../port/port";
import type { Port } from "./Providers";
import { EditConn } from "./EditConn";

afterEach(cleanup);

it("keeps vision capability discovered while refreshing a saved source", async () => {
  const visionModel = "deepseek-v4-flash-vision-exp";
  const editProvider = vi.fn(async () => {});
  const port = {
    checkProvider: vi.fn(async () => ({
      ok: true,
      models: ["deepseek-v4-flash", visionModel],
      vision: [visionModel],
    })),
    editProvider,
  } as unknown as Port;
  const entry: ProviderEntry = {
    name: "deepseek-flash",
    kind: "responses",
    baseUrl: "https://api.deepseek.com",
    models: ["deepseek-v4-flash"],
    default: "deepseek-v4-flash",
    hasKey: true,
    inUse: false,
    preset: true,
    canSetVision: false,
    visionModels: [],
    visionSettable: [],
  };

  render(<EditConn entry={entry} port={port} busy="" setBusy={() => {}} onDone={() => {}} />);
  await userEvent.click(screen.getByRole("button", { name: "重新问一次有哪些模型" }));

  const modelName = await screen.findByText(visionModel);
  const row = modelName.closest(".mline") as HTMLElement;
  await userEvent.click(within(row).getByRole("checkbox"));
  const vision = within(row).getByRole("button", { name: "读图" });
  expect((vision as HTMLButtonElement).disabled).toBe(false);
  expect(vision.getAttribute("aria-pressed")).toBe("true");

  await userEvent.click(screen.getByRole("button", { name: "保存" }));
  await waitFor(() => expect(editProvider).toHaveBeenCalled());
  expect(editProvider).toHaveBeenCalledWith(expect.objectContaining({ vision: [visionModel] }));
});
