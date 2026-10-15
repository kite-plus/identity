export type AccountSection = "profile" | "security" | "devices" | "delete";
export interface AccountProfile {
  firstName: string;
  lastName: string;
  displayName: string;
  nickName?: string;
  preferredLanguage?: string;
}
export interface AccountDevice {
  id: string;
  name: string;
  createdAt?: string;
  updatedAt?: string;
  current: boolean;
}
export interface AccountOverview {
  userId: string;
  createdAt?: string;
  profile: AccountProfile;
  email: string;
  emailVerified: boolean;
  loginName: string;
  passkeys: { id: string; name: string; state: string }[];
  totp: boolean;
  forceMfa: boolean;
  hasPassword: boolean;
  devices: AccountDevice[];
  passwordPolicy: {
    minLength?: string;
    hasUppercase?: boolean;
    hasLowercase?: boolean;
    hasNumber?: boolean;
    hasSymbol?: boolean;
  };
}
export type AccountCommand =
  | { type: "profile"; displayName: string; language: string }
  | { type: "email"; email: string }
  | { type: "email-verify"; code: string }
  | { type: "email-resend" }
  | { type: "password"; current: string; password: string }
  | { type: "passkey-start" }
  | { type: "passkey-finish"; id: string; name: string; credential: Record<string, unknown> }
  | { type: "passkey-remove"; id: string }
  | { type: "totp-start" }
  | { type: "totp-verify"; code: string }
  | { type: "totp-remove" }
  | { type: "device-revoke"; id: string }
  | { type: "devices-revoke" }
  | { type: "delete"; confirmation: string };
export type AccountResult =
  | { ok: true; data?: { id?: string; options?: Record<string, unknown>; secret?: string; uri?: string }; redirect?: string }
  | { ok: false; message: string; reauthRequired?: boolean; redirect?: string };
