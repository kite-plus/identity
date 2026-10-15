import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { createRef } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { TextInput } from "./input";
vi.mock("next-intl", () => ({ useLocale: () => "zh" }));
afterEach(cleanup);
describe("账号表单输入", () => {
  it("通过标签提供可访问的输入，并保留原生属性", () => {
    render(
      <TextInput
        label="邮箱"
        type="email"
        required
        autoComplete="email"
        maxLength={200}
        defaultValue="reader@example.com"
      />,
    );
    const input = screen.getByLabelText("邮箱");
    expect(input).toHaveValue("reader@example.com");
    expect(input).toBeRequired();
    expect(input).toHaveAttribute("autocomplete", "email");
    expect(input).toHaveAttribute("maxlength", "200");
  });
  it("将错误信息关联到输入字段", () => {
    render(<TextInput label="验证码" error="验证码无效" />);
    expect(screen.getByLabelText("验证码")).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByLabelText("验证码")).toHaveAccessibleDescription("验证码无效");
  });
  it("密码默认隐藏，并允许用户切换可见状态", () => {
    render(<TextInput label="密码" type="password" defaultValue="TestPassword" />);
    const input = screen.getByLabelText("密码");
    expect(input).toHaveAttribute("type", "password");
    fireEvent.click(screen.getByRole("button", { name: "显示密码" }));
    expect(input).toHaveAttribute("type", "text");
    fireEvent.click(screen.getByRole("button", { name: "隐藏密码" }));
    expect(input).toHaveAttribute("type", "password");
  });
  it("支持禁用状态和表单注册引用", () => {
    const ref = createRef<HTMLInputElement>();
    render(<TextInput label="显示名称" disabled ref={ref} />);
    expect(ref.current).toBe(screen.getByLabelText("显示名称"));
    expect(ref.current).toBeDisabled();
  });
  it("保留自动聚焦和表单变更事件", () => {
    const onChange = vi.fn();
    render(<TextInput label="用户名" autoFocus onChange={onChange} />);
    const input = screen.getByLabelText("用户名");
    expect(input).toHaveFocus();
    fireEvent.change(input, { target: { value: "reader" } });
    expect(onChange).toHaveBeenCalledTimes(1);
  });
});
