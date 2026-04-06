"use client";
import Link from "next/link";
import { useState } from "react";

export default function LoginPage() {
  const [form, setForm] = useState({ email: "", password: "" });
  const [errors, setErrors] = useState<{ email?: string; password?: string }>({});
  const [loading, setLoading] = useState(false);

  const validate = () => {
    const e: typeof errors = {};
    if (!form.email.includes("@")) e.email = "Enter a valid email address";
    if (form.password.length < 6) e.password = "Password must be at least 6 characters";
    return e;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const errs = validate();
    if (Object.keys(errs).length) { setErrors(errs); return; }
    setLoading(true);
    // TODO: POST /api/auth/login
    setTimeout(() => { setLoading(false); window.location.href = "/onboarding"; }, 1200);
  };

  return (
    <main className="page">
      <div className="card">
        {/* Logo */}
        <Link href="/" className="logo">NutriMatch</Link>

        <h1 className="title">Welcome back</h1>
        <p className="sub">Sign in to access your nutrition profile</p>

        <form onSubmit={handleSubmit} className="form" noValidate>
          {/* Email */}
          <div className="field">
            <label className="label" htmlFor="email">Email</label>
            <input
              id="email"
              type="email"
              autoComplete="email"
              placeholder="you@example.com"
              className={`input ${errors.email ? "input-error" : ""}`}
              value={form.email}
              onChange={(e) => { setForm({ ...form, email: e.target.value }); setErrors({ ...errors, email: undefined }); }}
            />
            {errors.email && <span className="error">{errors.email}</span>}
          </div>

          {/* Password */}
          <div className="field">
            <div className="label-row">
              <label className="label" htmlFor="password">Password</label>
              <span className="forgot">Forgot password?</span>
            </div>
            <input
              id="password"
              type="password"
              autoComplete="current-password"
              placeholder="••••••••"
              className={`input ${errors.password ? "input-error" : ""}`}
              value={form.password}
              onChange={(e) => { setForm({ ...form, password: e.target.value }); setErrors({ ...errors, password: undefined }); }}
            />
            {errors.password && <span className="error">{errors.password}</span>}
          </div>

          <button type="submit" className="btn" disabled={loading}>
            {loading ? <span className="spinner" /> : "Sign in"}
          </button>
        </form>

        <p className="switch">
          No account yet?{" "}
          <Link href="/register" className="switch-link">Create one</Link>
        </p>
      </div>

      <style>{`
        *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
        :root {
          --bg: #0a0f0a; --surface: #111811; --border: #1e2b1e;
          --green: #4ade80; --green-dim: #166534; --green-glow: rgba(74,222,128,0.1);
          --text: #f0fdf0; --muted: #6b7c6b; --error: #f87171;
          --font-display: 'Georgia', serif;
          --font-body: 'Helvetica Neue', Helvetica, sans-serif;
        }
        body { background: var(--bg); color: var(--text); font-family: var(--font-body); }

        .page {
          min-height: 100vh; display: flex; align-items: center; justify-content: center;
          padding: 2rem;
          background: radial-gradient(ellipse at 60% 20%, rgba(74,222,128,0.06) 0%, transparent 60%);
        }
        .card {
          width: 100%; max-width: 420px;
          background: var(--surface); border: 1px solid var(--border);
          border-radius: 16px; padding: 2.5rem;
          display: flex; flex-direction: column; gap: 1.5rem;
          animation: fadeUp 0.5s ease both;
        }
        .logo {
          font-family: var(--font-display); font-size: 1.2rem;
          color: var(--green); text-decoration: none; width: fit-content;
        }
        .title { font-family: var(--font-display); font-size: 1.8rem; color: var(--text); }
        .sub { font-size: 0.88rem; color: var(--muted); margin-top: -1rem; }

        .form { display: flex; flex-direction: column; gap: 1.25rem; }
        .field { display: flex; flex-direction: column; gap: 0.4rem; }
        .label { font-size: 0.82rem; font-weight: 600; color: var(--text); letter-spacing: 0.02em; }
        .label-row { display: flex; justify-content: space-between; align-items: center; }
        .forgot { font-size: 0.78rem; color: var(--muted); cursor: pointer; transition: color 0.2s; }
        .forgot:hover { color: var(--green); }

        .input {
          background: var(--bg); border: 1px solid var(--border);
          border-radius: 8px; padding: 0.65rem 0.9rem;
          color: var(--text); font-size: 0.92rem; font-family: var(--font-body);
          outline: none; transition: border-color 0.2s, box-shadow 0.2s; width: 100%;
        }
        .input::placeho lder { color: var(--muted); }
        .input:focus { border-color: var(--green); box-shadow: 0 0 0 3px var(--green-glow); }
        .input-error { border-color: var(--error) !important; }
        .error { font-size: 0.78rem; color: var(--error); }

        .btn {
          background: var(--green); color: #0a0f0a;
          border: none; border-radius: 8px; padding: 0.8rem;
          font-size: 0.95rem; font-weight: 700; cursor: pointer;
          transition: opacity 0.2s, transform 0.2s; margin-top: 0.25rem;
          display: flex; align-items: center; justify-content: center; min-height: 44px;
        }
        .btn:hover:not(:disabled) { opacity: 0.88; transform: translateY(-1px); }
        .btn:disabled { opacity: 0.6; cursor: not-allowed; }

        .spinner {
          width: 18px; height: 18px; border: 2px solid #0a0f0a;
          border-top-color: transparent; border-radius: 50%;
          animation: spin 0.7s linear infinite;
        }
        @keyframes spin { to { transform: rotate(360deg); } }

        .switch { font-size: 0.85rem; color: var(--muted); text-align: center; }
        .switch-link { color: var(--green); text-decoration: none; font-weight: 600; }
        .switch-link:hover { text-decoration: underline; }

        @keyframes fadeUp {
          from { opacity: 0; transform: translateY(20px); }
          to   { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </main>
  );
}
