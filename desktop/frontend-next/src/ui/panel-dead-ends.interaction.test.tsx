// @vitest-environment jsdom
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import "./testkit";
import { Account } from "./Account";
import { Memory } from "./Memory";
import { Versions } from "./Versions";
import { HttpError, type AgentPort } from "../port/port";
import { MockPort } from "../port/mock";
import { reason } from "../i18n/kernel";

afterEach(cleanup);

// A settings page must never be somewhere a reader arrives and can do nothing.
// Two ways it used to become one, and both had an answer already in the tree.
//
// A read the kernel refuses comes back with a code and a sentence the frontend
// already carries a translation for; folding that into the component's initial
// null spent both, and the panel went on saying "loading…" about a question
// that had been answered, for the rest of the session.
//
// An empty store is the other: the panel reads, rewrites and removes facts,
// and writing one is the composer's /remember — which this page never named.
const refusal = (code: string) => new HttpError(403, "english fallback for logs", { code }, true);
const said = (code: string) => reason(refusal(code));

describe("a read the kernel refuses", () => {
  it("says why on the version panel instead of loading forever", async () => {
    const port = {
      versions: () => Promise.reject(refusal("studio.no_install")),
      pinVersion: () => Promise.resolve(),
      goToVersion: () => Promise.resolve(),
      onUpdateProgress: () => () => {},
    };
    render(<Versions port={port} />);
    expect(await screen.findByText(said("studio.no_install"))).toBeTruthy();
    expect(screen.queryByText("正在读取版本…")).toBeNull();
    expect(screen.getByRole("button", { name: "再试一次" })).toBeTruthy();
  });

  it("says why on the memory panel instead of calling an unread store empty", async () => {
    const port = new MockPort() as unknown as AgentPort;
    const refusing = { ...port, memories: () => Promise.reject(refusal("internal.failed")) } as unknown as AgentPort;
    render(<Memory port={refusing} />);
    expect(await screen.findByText(said("internal.failed"))).toBeTruthy();
    // The old first paint said this outright, before any answer had arrived.
    expect(screen.queryByText("读不到记忆。")).toBeNull();
  });

  it("says why on the account panel instead of checking forever", () => {
    const why = said("account.signin_disabled");
    render(<Account port={new MockPort() as unknown as AgentPort} state={null} unread={why} reload={() => {}} />);
    expect(screen.getByText(why)).toBeTruthy();
    expect(screen.queryByText("正在检查登录状态…")).toBeNull();
  });

  // Unknown and refused are different states, and the wait is still the wait.
  it("still waits while the answer is genuinely on its way", () => {
    render(<Account port={new MockPort() as unknown as AgentPort} state={null} unread="" reload={() => {}} />);
    expect(screen.getByText("正在检查登录状态…")).toBeTruthy();
  });
});

describe("a store with nothing in it", () => {
  // Not a second write path — the one the CLI's memory view already points at.
  it("names the way to write a fact instead of ending the page", async () => {
    const port = new MockPort() as unknown as AgentPort;
    const empty = { ...port, memories: () => Promise.resolve({ memories: [], recallQuery: "" }) } as unknown as AgentPort;
    render(<Memory port={empty} />);
    expect(await screen.findByText("还没有记下任何东西。")).toBeTruthy();
    expect(screen.getByText("/remember")).toBeTruthy();
  });
});
