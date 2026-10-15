import { coerceToArrayBuffer, coerceToBase64Url } from "./base64";

export async function useAccountPasskey(
  options: Record<string, unknown>,
  registration: boolean,
): Promise<Record<string, unknown>> {
  if (!window.isSecureContext || !navigator.credentials || typeof PublicKeyCredential === "undefined")
    throw new Error("unsupported");
  const publicKey = structuredClone((options.publicKey || options) as Record<string, unknown>);
  publicKey.challenge = coerceToArrayBuffer(publicKey.challenge, "challenge");
  const credentials = (registration ? publicKey.excludeCredentials : publicKey.allowCredentials) as
    { id: string | ArrayBuffer }[] | undefined;
  credentials?.forEach((credential) => {
    credential.id = coerceToArrayBuffer(credential.id, "id");
  });
  if (registration) {
    const user = publicKey.user as { id: string | ArrayBuffer };
    user.id = coerceToArrayBuffer(user.id, "user.id");
  }
  const credential = (
    registration
      ? await navigator.credentials.create({ publicKey: publicKey as unknown as PublicKeyCredentialCreationOptions })
      : await navigator.credentials.get({ publicKey: publicKey as unknown as PublicKeyCredentialRequestOptions })
  ) as PublicKeyCredential | null;
  if (!credential) throw new Error("cancelled");
  const response = credential.response;
  const serializedResponse: Record<string, unknown> = {
    clientDataJSON: coerceToBase64Url(response.clientDataJSON, "clientDataJSON"),
  };
  if (registration) {
    const attestation = response as AuthenticatorAttestationResponse;
    serializedResponse.attestationObject = coerceToBase64Url(attestation.attestationObject, "attestationObject");
    if (attestation.getTransports) serializedResponse.transports = attestation.getTransports();
  } else {
    const assertion = response as AuthenticatorAssertionResponse;
    serializedResponse.authenticatorData = coerceToBase64Url(assertion.authenticatorData, "authenticatorData");
    serializedResponse.signature = coerceToBase64Url(assertion.signature, "signature");
    serializedResponse.userHandle = assertion.userHandle ? coerceToBase64Url(assertion.userHandle, "userHandle") : null;
  }
  return {
    id: credential.id,
    rawId: coerceToBase64Url(credential.rawId, "rawId"),
    type: credential.type,
    response: serializedResponse,
    clientExtensionResults: credential.getClientExtensionResults(),
  };
}
