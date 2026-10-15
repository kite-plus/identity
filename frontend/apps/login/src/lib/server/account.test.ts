import { AccountCommand } from "@/lib/account-types";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { mutateAccount } from "./account";
const mocks = vi.hoisted(() => ({ request: vi.fn(), context: vi.fn(), settings: vi.fn() }));
vi.mock("./account-context", async (importOriginal) => ({
  ...(await importOriginal<typeof import("./account-context")>()),
  requireAccountContext: mocks.context,
  accountRequest: mocks.request,
}));
vi.mock("next-intl/server", () => ({ getLocale: async () => "zh" }));
vi.mock("next/cache", () => ({ revalidatePath: vi.fn() }));
vi.mock("@/lib/zitadel", () => ({ getLoginSettings: mocks.settings }));
vi.mock("@/lib/cookies", () => ({ setLanguageCookie: vi.fn(), getAllSessions: vi.fn(), removeSessionFromCookie: vi.fn() }));
beforeEach(() => {
  vi.clearAllMocks();
  mocks.context.mockResolvedValue({
    userId: "alice",
    cookie: { id: "current", token: "private", loginName: "alice" },
    session: { factors: { user: { id: "alice" } } },
    serviceConfig: {},
    domain: "localhost",
  });
});
describe("账号写入的用户隔离", () => {
  it("拒绝退出其他用户的设备", async () => {
    mocks.request.mockImplementation(async (_context, path) =>
      path.endsWith("/email") ? { email: { isEmailVerified: true } } : { session: { factors: { user: { id: "bob" } } } },
    );
    const result = await mutateAccount({ type: "device-revoke", id: "bob-session" });
    expect(result.ok).toBe(false);
    expect(mocks.request.mock.calls.some((call) => call[2] === "DELETE")).toBe(false);
  });
  it("不允许客户端指定资料修改的用户", async () => {
    mocks.request.mockResolvedValue({
      profile: { firstName: "Alice", lastName: "Reader", displayName: "New name", preferredLanguage: "zh" },
    });
    const result = await mutateAccount({
      type: "profile",
      displayName: "New name",
      language: "zh",
      userId: "bob",
    } as AccountCommand);
    expect(result.ok).toBe(true);
    const write = mocks.request.mock.calls.find((call) => call[2] === "PUT");
    expect(write?.[1]).toBe("/auth/v1/users/me/profile");
    expect(write?.[3].userId).toBeUndefined();
  });
  it("邮箱未验证时拒绝修改密码", async () => {
    mocks.request.mockResolvedValue({ email: { isEmailVerified: false } });
    expect((await mutateAccount({ type: "password", current: "Old", password: "New" })).ok).toBe(false);
    expect(mocks.request.mock.calls.some((call) => call[2] === "PUT")).toBe(false);
  });
  it("拒绝删除账号时不准确的确认文字", async () => {
    mocks.request.mockResolvedValue({ email: { isEmailVerified: true } });
    expect((await mutateAccount({ type: "delete", confirmation: "yes" })).ok).toBe(false);
    expect(mocks.request.mock.calls.some((call) => call[2] === "DELETE")).toBe(false);
  });
  it("保护没有其他登录方式的最后一枚通行密钥", async () => {
    mocks.request.mockImplementation(async (_context, path) => {
      if (path.endsWith("/email")) return { email: { isEmailVerified: true } };
      if (path.endsWith("/authentication_methods")) return { authMethodTypes: ["AUTHENTICATION_METHOD_TYPE_PASSKEY"] };
      return { result: [{ id: "only-key", state: "AUTH_FACTOR_STATE_READY" }] };
    });
    expect((await mutateAccount({ type: "passkey-remove", id: "only-key" })).ok).toBe(false);
    expect(mocks.request.mock.calls.some((call) => call[2] === "DELETE")).toBe(false);
  });
});
