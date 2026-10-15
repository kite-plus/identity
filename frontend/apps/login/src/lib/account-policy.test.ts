import { create, timestampFromMs } from "@zitadel/client";
import { SessionSchema } from "@zitadel/proto/zitadel/session/v2/session_pb";
import { describe, expect, it, vi } from "vitest";
import { hasFreshAccountAuthentication } from "./account-policy";
vi.mock("./zitadel", () => ({}));

const now = Date.now();
const recent = timestampFromMs(now - 1000);
const old = timestampFromMs(now - 11 * 60 * 1000);

describe("账号敏感操作的身份验证", () => {
  it("拒绝只识别用户名的会话", () => {
    expect(hasFreshAccountAuthentication(create(SessionSchema, { factors: { user: { id: "reader" } } }), now)).toBe(false);
  });
  it("拒绝只有用户在场、没有用户验证的安全密钥", () => {
    expect(
      hasFreshAccountAuthentication(
        create(SessionSchema, { factors: { webAuthN: { verifiedAt: recent, userVerified: false } } }),
        now,
      ),
    ).toBe(false);
  });
  it("拒绝过期会话", () => {
    expect(
      hasFreshAccountAuthentication(
        create(SessionSchema, { expirationDate: old, factors: { password: { verifiedAt: recent } } }),
        now,
      ),
    ).toBe(false);
  });
  it("拒绝十分钟前的密码验证", () => {
    expect(hasFreshAccountAuthentication(create(SessionSchema, { factors: { password: { verifiedAt: old } } }), now)).toBe(
      false,
    );
  });
  it("拒绝密码刚验证但第二因素过旧的会话", () => {
    expect(
      hasFreshAccountAuthentication(
        create(SessionSchema, { factors: { password: { verifiedAt: recent }, totp: { verifiedAt: old } } }),
        now,
      ),
    ).toBe(false);
  });
  it("接受最近验证的密码和第二因素", () => {
    expect(
      hasFreshAccountAuthentication(
        create(SessionSchema, { factors: { password: { verifiedAt: recent }, totp: { verifiedAt: recent } } }),
        now,
      ),
    ).toBe(true);
  });
  it("接受新验证的通行密钥，即使旧密码和第二因素过期", () => {
    expect(
      hasFreshAccountAuthentication(
        create(SessionSchema, {
          factors: {
            password: { verifiedAt: old },
            totp: { verifiedAt: old },
            webAuthN: { verifiedAt: recent, userVerified: true },
          },
        }),
        now,
      ),
    ).toBe(true);
  });
  it("拒绝未来时间的验证记录", () => {
    expect(
      hasFreshAccountAuthentication(
        create(SessionSchema, { factors: { password: { verifiedAt: timestampFromMs(now + 1000) } } }),
        now,
      ),
    ).toBe(false);
  });
});
