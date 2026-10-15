import { create, timestampFromMs } from "@zitadel/client";
import { SessionSchema } from "@zitadel/proto/zitadel/session/v2/session_pb";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { accountRequest, requireAccountContext } from "./account-context";
const mocks = vi.hoisted(() => ({ cookie: vi.fn(), getSession: vi.fn(), valid: vi.fn() }));
vi.mock("@/lib/cookies", () => ({ getMostRecentSessionCookie: mocks.cookie, updateSessionCookie: vi.fn() }));
vi.mock("next/headers", () => ({ headers: async () => new Headers({ host: "localhost:8091" }) }));
vi.mock("@/lib/service-url", () => ({ getServiceConfig: () => ({ serviceConfig: { baseUrl: "http://identity.test" } }) }));
vi.mock("@/lib/zitadel", () => ({ getSession: mocks.getSession }));
vi.mock("@/lib/session", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/session")>()),
  isSessionValid: mocks.valid,
}));

beforeEach(() => {
  vi.clearAllMocks();
  mocks.cookie.mockResolvedValue({ id: "session", token: "private-token", loginName: "reader" });
  mocks.valid.mockResolvedValue(true);
});
describe("账号会话权限边界", () => {
  it("拒绝缺少签名 Cookie 的请求", async () => {
    mocks.cookie.mockResolvedValue(undefined);
    await expect(requireAccountContext()).rejects.toMatchObject({ reason: "login" });
    expect(mocks.getSession).not.toHaveBeenCalled();
  });
  it("拒绝只识别了用户名的会话", async () => {
    mocks.getSession.mockResolvedValue({ session: create(SessionSchema, { factors: { user: { id: "reader" } } }) });
    await expect(requireAccountContext()).rejects.toMatchObject({ reason: "login" });
  });
  it("拒绝缺少所需双重验证的会话", async () => {
    mocks.getSession.mockResolvedValue({
      session: create(SessionSchema, {
        factors: { user: { id: "reader" }, password: { verifiedAt: timestampFromMs(Date.now()) } },
      }),
    });
    mocks.valid.mockResolvedValue(false);
    await expect(requireAccountContext()).rejects.toMatchObject({ reason: "login" });
  });
  it("只从已验证的服务端会话确定用户", async () => {
    mocks.getSession.mockResolvedValue({
      session: create(SessionSchema, {
        factors: { user: { id: "reader" }, password: { verifiedAt: timestampFromMs(Date.now()) } },
      }),
    });
    const context = await requireAccountContext();
    expect(context.userId).toBe("reader");
    expect(mocks.getSession).toHaveBeenCalledWith({
      serviceConfig: { baseUrl: "http://identity.test" },
      sessionId: "session",
      sessionToken: "private-token",
    });
  });
  it("向自助 API 传递用户会话令牌", async () => {
    mocks.getSession.mockResolvedValue({
      session: create(SessionSchema, {
        factors: { user: { id: "reader" }, password: { verifiedAt: timestampFromMs(Date.now()) } },
      }),
    });
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: async () => ({}) });
    vi.stubGlobal("fetch", fetchMock);
    await accountRequest(await requireAccountContext(), "/auth/v1/users/me");
    expect(fetchMock.mock.calls[0][1].headers.get("Authorization")).toBe("Bearer private-token");
    expect(fetchMock.mock.calls[0][1].cache).toBe("no-store");
    vi.unstubAllGlobals();
  });
});
