import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { createRef } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Button } from "./button";
afterEach(cleanup);
describe("账号操作按钮", () => {
  it("默认不会提交所在表单", () => {
    const submit = vi.fn((event) => event.preventDefault());
    render(
      <form onSubmit={submit}>
        <Button>取消</Button>
      </form>,
    );
    fireEvent.click(screen.getByRole("button", { name: "取消" }));
    expect(submit).not.toHaveBeenCalled();
  });
  it("明确指定提交按钮时提交表单", () => {
    const submit = vi.fn((event) => event.preventDefault());
    render(
      <form onSubmit={submit}>
        <Button type="submit">保存</Button>
      </form>,
    );
    fireEvent.click(screen.getByRole("button", { name: "保存" }));
    expect(submit).toHaveBeenCalledTimes(1);
  });
  it("禁用按钮不会执行操作", () => {
    const click = vi.fn();
    render(
      <Button disabled onClick={click}>
        删除
      </Button>,
    );
    fireEvent.click(screen.getByRole("button", { name: "删除" }));
    expect(click).not.toHaveBeenCalled();
  });
  it("保留可访问名称和原生引用", () => {
    const ref = createRef<HTMLButtonElement>();
    render(
      <Button ref={ref} aria-label="关闭对话框">
        ×
      </Button>,
    );
    expect(ref.current).toBe(screen.getByRole("button", { name: "关闭对话框" }));
  });
});
