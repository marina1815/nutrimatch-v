import type { NextConfig } from "next";

const apiURL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
const isDev = process.env.NODE_ENV !== "production";
const csp = [
  "default-src 'self'",
  `connect-src 'self' ${apiURL}`,
  "img-src 'self' data: blob:",
  "font-src 'self'",
  "media-src 'none'",
  "object-src 'none'",
  "base-uri 'self'",
  "form-action 'self'",
  "frame-ancestors 'none'",
  "manifest-src 'self'",
  "worker-src 'self' blob:",
  "style-src 'self'",
  `script-src 'self'${isDev ? " 'unsafe-inline' 'unsafe-eval'" : ""}`,
  ...(isDev ? [] : ["require-trusted-types-for 'script'"]),
  ...(isDev ? [] : ["upgrade-insecure-requests"]),
].join("; ");

const nextConfig: NextConfig = {
  distDir: "build",
  reactCompiler: true,
  async headers() {
    return [
      {
        source: "/:path*",
        headers: [
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "X-Frame-Options", value: "DENY" },
          { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
          { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=(), publickey-credentials-get=(self)" },
          { key: "Content-Security-Policy", value: csp },
        ],
      },
    ];
  },
};

export default nextConfig;
