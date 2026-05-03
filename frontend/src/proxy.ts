import { NextRequest, NextResponse } from "next/server";

const PUBLIC_PATHS = new Set([
  "/",
  "/login",
  "/signin",
  "/register",
  "/signup",
  "/signout",
  "/auth/oidc/callback",
]);

function normalizeSource(value: string | undefined, fallback: string): string {
  try {
    return new URL(value || fallback).origin;
  } catch {
    return new URL(fallback).origin;
  }
}

function isPublicPath(pathname: string): boolean {
  if (PUBLIC_PATHS.has(pathname)) {
    return true;
  }
  return pathname.startsWith("/_next/") ||
    pathname.startsWith("/favicon") ||
    pathname.startsWith("/assets/");
}

export function proxy(request: NextRequest) {
  const nonce = Buffer.from(crypto.randomUUID()).toString("base64");
  const isDev = process.env.NODE_ENV === "development";
  const apiSource = normalizeSource(process.env.NEXT_PUBLIC_API_URL, "http://localhost:8080");
  const upgradeInsecureRequests = process.env.CSP_UPGRADE_INSECURE_REQUESTS === "true";
  const devConnectSources = isDev ? " ws://localhost:* ws://127.0.0.1:*" : "";

  const csp = [
    "default-src 'self'",
    `script-src 'self' 'nonce-${nonce}' 'strict-dynamic'${isDev ? " 'unsafe-eval'" : ""}`,
    `style-src 'self' 'nonce-${nonce}'`,
    `connect-src 'self' ${apiSource}${devConnectSources}`,
    "img-src 'self' data: blob:",
    "font-src 'self'",
    "media-src 'none'",
    "object-src 'none'",
    "base-uri 'self'",
    "form-action 'self'",
    "frame-ancestors 'none'",
    "manifest-src 'self'",
    "worker-src 'self' blob:",
    ...(upgradeInsecureRequests ? ["upgrade-insecure-requests"] : []),
  ]
    .join("; ")
    .replace(/\s{2,}/g, " ")
    .trim();

  const requestHeaders = new Headers(request.headers);
  requestHeaders.set("x-nonce", nonce);
  requestHeaders.set("Content-Security-Policy", csp);

  if (!isPublicPath(request.nextUrl.pathname) && !request.cookies.get("nm_refresh")?.value) {
    const loginURL = request.nextUrl.clone();
    loginURL.pathname = "/login";
    loginURL.searchParams.set("next", request.nextUrl.pathname);
    const redirect = NextResponse.redirect(loginURL);
    redirect.headers.set("Content-Security-Policy", csp);
    return redirect;
  }

  const response = NextResponse.next({
    request: {
      headers: requestHeaders,
    },
  });
  response.headers.set("Content-Security-Policy", csp);
  return response;
}

export const config = {
  matcher: [
    {
      source: "/((?!api|_next/static|_next/image|favicon.ico).*)",
      missing: [
        { type: "header", key: "next-router-prefetch" },
        { type: "header", key: "purpose", value: "prefetch" },
      ],
    },
  ],
};
