// @vitest-environment jsdom
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import "./testkit";
import { RailSearch } from "./railsearch";
import { RemoteHosts } from "./RemoteHosts";
import type { RemoteHost } from "../port/remote";
import type { HubPort } from "../port/hub";

afterEach(cleanup);

const host = (name: string, over: Partial<RemoteHost> = {}): RemoteHost => ({
  name,
  target: `ada@${name}.internal`,
  status: "connected",
  workspaces: [`/home/ada/${name}-work`],
  ...over,
});

const hub = { remoteTree: async () => [] } as unknown as HubPort;

const draw = (hosts: RemoteHost[]) =>
  render(
    <RailSearch>
      <RemoteHosts
        hub={hub}
        hosts={hosts}
        runtimes={[]}
        active=""
        onOpen={async () => {}}
        onFocus={() => {}}
        reload={async () => {}}
        onError={() => {}}
      />
    </RailSearch>,
  );

const find = () => screen.getByRole("searchbox", { name: "搜索会话 / 项目" });

// 栏里是一份机器的列表，所以搜索框问的必须是整份列表。它此前只过滤本机那一半：
// 在两个面板里那读起来像「本地搜索」，合成一个列表之后，它读起来是一个悄悄跳过
// 大半列表的搜索 —— 而搜不到的那半边看上去就像不存在。
describe("the rail's search reaches every machine", () => {
  it("keeps a machine whose name matches", async () => {
    draw([host("gpu"), host("builder")]);
    await userEvent.type(find(), "gpu");
    expect(screen.queryByText("gpu")).toBeTruthy();
    expect(screen.queryByText("builder")).toBeNull();
  });

  it("keeps a machine whose address matches, not only its name", async () => {
    draw([host("gpu", { target: "ada@10.0.0.4" }), host("builder")]);
    await userEvent.type(find(), "10.0.0");
    expect(screen.queryByText("gpu")).toBeTruthy();
    expect(screen.queryByText("builder")).toBeNull();
  });

  it("keeps a machine whose folder matches, and only that folder", async () => {
    draw([host("gpu", { workspaces: ["/home/ada/training", "/home/ada/scratch"] })]);
    await userEvent.type(find(), "train");
    expect(screen.queryByText("training")).toBeTruthy();
    expect(screen.queryByText("scratch")).toBeNull();
  });

  it("puts every machine back when the word is cleared", async () => {
    draw([host("gpu"), host("builder")]);
    await userEvent.type(find(), "gpu");
    expect(screen.queryByText("builder")).toBeNull();
    await userEvent.clear(find());
    expect(screen.queryByText("builder")).toBeTruthy();
  });

  // 折叠是歇着时的偏好。找东西时它藏起来的正是刚找到的那些行 —— 本机那一半
  // 一直是这么做的，合并之后两边必须是同一条规矩。
  it("opens a folded machine rather than hiding what it just found", async () => {
    draw([host("gpu", { workspaces: ["/home/ada/training"] })]);
    await userEvent.click(screen.getByText("gpu"));
    expect(screen.queryByText("training")).toBeNull();
    await userEvent.type(find(), "train");
    expect(screen.queryByText("training")).toBeTruthy();
  });
});
