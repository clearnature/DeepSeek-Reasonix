// @vitest-environment jsdom
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SayCard } from "./SayCard";

afterEach(cleanup);

describe("completed reasoning", () => {
  it("can be opened again after it automatically folds", async () => {
    const { container } = render(<SayCard item={{ t: "say", id: "s", text: "answer", reasoning: "reason", done: true }} />);
    const details = container.querySelector("details") as HTMLDetailsElement;
    expect(details.open).toBe(false);
    await userEvent.click(screen.getByText(/想了/));
    expect(details.open).toBe(true);
  });
});
