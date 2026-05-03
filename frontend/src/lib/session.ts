import "client-only";

let accessTokenMemory: string | null = null;

export function getAccessToken(): string | null {
  return accessTokenMemory;
}

export function setAccessToken(token: string): void {
  accessTokenMemory = token;
}

export function clearAccessToken(): void {
  accessTokenMemory = null;
}

export function clearClientSession(): void {
  clearAccessToken();
}
