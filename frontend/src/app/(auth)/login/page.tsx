"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { ApiError, loginUser } from "@/lib/api";
import { getSafeErrorMessage } from "@/lib/ui-errors";

export default function LoginPage() {
  const router = useRouter();
  const [form, setForm] = useState({ email: "", password: "" });
  const [errors, setErrors] = useState<{ email?: string; password?: string; form?: string }>({});
  const [loading, setLoading] = useState(false);

  const validate = () => {
    const nextErrors: typeof errors = {};
    if (!form.email.includes("@")) nextErrors.email = "Enter a valid email address";
    if (form.password.length < 6) nextErrors.password = "Password must be at least 6 characters";
    return nextErrors;
  };

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    const nextErrors = validate();
    if (Object.keys(nextErrors).length > 0) {
      setErrors(nextErrors);
      return;
    }

    setLoading(true);
    setErrors({});

    try {
      await loginUser(form);
      router.push("/onboarding");
    } catch (error) {
      if (error instanceof ApiError) {
        setErrors({ form: getSafeErrorMessage(error, "auth.login") });
      } else {
        setErrors({ form: getSafeErrorMessage(error, "auth.login") });
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <main className="page">
      <div className="card">
        <Link href="/" className="logo">NutriMatch</Link>

        <h1 className="title">Welcome back</h1>
        <p className="sub">Sign in to access your nutrition profile</p>

        <form onSubmit={(event) => void handleSubmit(event)} className="form" noValidate>
          <div className="field">
            <label className="label" htmlFor="email">Email</label>
            <input
              id="email"
              type="email"
              autoComplete="email"
              placeholder="you@example.com"
              maxLength={254}
              className={`input ${errors.email ? "input-error" : ""}`}
              value={form.email}
              onChange={(event) => {
                setForm({ ...form, email: event.target.value });
                setErrors({ ...errors, email: undefined, form: undefined });
              }}
            />
            {errors.email && <span className="error">{errors.email}</span>}
          </div>

          <div className="field">
            <div className="label-row">
              <label className="label" htmlFor="password">Password</label>
              <span className="forgot">Forgot password?</span>
            </div>
            <input
              id="password"
              type="password"
              autoComplete="current-password"
              placeholder="........"
              maxLength={128}
              className={`input ${errors.password ? "input-error" : ""}`}
              value={form.password}
              onChange={(event) => {
                setForm({ ...form, password: event.target.value });
                setErrors({ ...errors, password: undefined, form: undefined });
              }}
            />
            {errors.password && <span className="error">{errors.password}</span>}
          </div>

          {errors.form && <span className="error">{errors.form}</span>}

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
        .input::placeholder { color: var(--muted); }
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
