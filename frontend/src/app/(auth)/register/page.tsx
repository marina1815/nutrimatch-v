"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { ApiError, getCurrentSession, registerUser } from "@/lib/api";
import { setCurrentProfileId } from "@/lib/session";
import { getSafeErrorMessage } from "@/lib/ui-errors";

interface FormState {
  name: string;
  email: string;
  password: string;
  confirm: string;
}

interface FormErrors {
  name?: string;
  email?: string;
  password?: string;
  confirm?: string;
  form?: string;
}

export default function RegisterPage() {
  const router = useRouter();
  const [form, setForm] = useState<FormState>({ name: "", email: "", password: "", confirm: "" });
  const [errors, setErrors] = useState<FormErrors>({});
  const [loading, setLoading] = useState(false);

  const setField = (field: keyof FormState) => (event: React.ChangeEvent<HTMLInputElement>) => {
    setForm((current) => ({ ...current, [field]: event.target.value }));
    setErrors((current) => ({ ...current, [field]: undefined, form: undefined }));
  };

  const validate = (): FormErrors => {
    const nextErrors: FormErrors = {};
    if (form.name.trim().length < 2) nextErrors.name = "Name must be at least 2 characters";
    if (!form.email.includes("@")) nextErrors.email = "Enter a valid email address";
    if (form.password.length < 12) nextErrors.password = "Password must be at least 12 characters";
    if (form.confirm !== form.password) nextErrors.confirm = "Passwords do not match";
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
      await registerUser({
        name: form.name,
        email: form.email,
        password: form.password,
      });
      try {
        const session = await getCurrentSession();
        if (session.profileId) {
          setCurrentProfileId(session.profileId);
        }
        router.push(session.hasProfile ? "/results" : "/onboarding");
      } catch {
        router.push("/onboarding");
      }
    } catch (error) {
      if (error instanceof ApiError) {
        setErrors({ form: getSafeErrorMessage(error, "auth.register") });
      } else {
        setErrors({ form: getSafeErrorMessage(error, "auth.register") });
      }
    } finally {
      setLoading(false);
    }
  };

  const strength = (() => {
    const password = form.password;
    if (!password) return 0;
    let score = 0;
    if (password.length >= 12) score++;
    if (/[A-Z]/.test(password)) score++;
    if (/[0-9]/.test(password)) score++;
    if (/[^A-Za-z0-9]/.test(password)) score++;
    return score;
  })();

  const strengthLabel = ["", "Weak", "Fair", "Good", "Strong"][strength];
  const strengthColor = ["", "#f87171", "#fbbf24", "#60a5fa", "#4ade80"][strength];

  return (
    <main className="page">
      <div className="card">
        <Link href="/" className="logo">NutriMatch</Link>

        <h1 className="title">Create your account</h1>
        <p className="sub">Start building your personalised nutrition profile</p>

        <form onSubmit={(event) => void handleSubmit(event)} className="form" noValidate>
          <div className="field">
            <label className="label" htmlFor="name">Full name</label>
            <input
              id="name"
              type="text"
              autoComplete="name"
              placeholder="Amine Benali"
              maxLength={120}
              className={`input ${errors.name ? "input-error" : ""}`}
              value={form.name}
              onChange={setField("name")}
            />
            {errors.name && <span className="error">{errors.name}</span>}
          </div>

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
              onChange={setField("email")}
            />
            {errors.email && <span className="error">{errors.email}</span>}
          </div>

          <div className="field">
            <label className="label" htmlFor="password">Password</label>
            <input
              id="password"
              type="password"
              autoComplete="new-password"
              placeholder="Min. 12 characters"
              maxLength={128}
              className={`input ${errors.password ? "input-error" : ""}`}
              value={form.password}
              onChange={setField("password")}
            />
            {form.password && (
              <div className="strength-row">
                <div className="strength-bar">
                  {[1, 2, 3, 4].map((index) => (
                    <div
                      key={index}
                      className="strength-seg"
                      style={{ background: index <= strength ? strengthColor : "var(--border)" }}
                    />
                  ))}
                </div>
                <span className="strength-label" style={{ color: strengthColor }}>{strengthLabel}</span>
              </div>
            )}
            {errors.password && <span className="error">{errors.password}</span>}
          </div>

          <div className="field">
            <label className="label" htmlFor="confirm">Confirm password</label>
            <input
              id="confirm"
              type="password"
              autoComplete="new-password"
              placeholder="........"
              maxLength={128}
              className={`input ${errors.confirm ? "input-error" : ""}`}
              value={form.confirm}
              onChange={setField("confirm")}
            />
            {errors.confirm && <span className="error">{errors.confirm}</span>}
          </div>

          <p className="terms">
            By creating an account you agree that your data is used solely to generate
            personalised meal suggestions and is never shared with third parties.
          </p>

          {errors.form && <span className="error">{errors.form}</span>}

          <button type="submit" className="btn" disabled={loading}>
            {loading ? <span className="spinner" /> : "Create account"}
          </button>
        </form>

        <p className="switch">
          Already have an account?{" "}
          <Link href="/login" className="switch-link">Sign in</Link>
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
          background: radial-gradient(ellipse at 40% 30%, rgba(74,222,128,0.06) 0%, transparent 60%);
        }
        .card {
          width: 100%; max-width: 440px;
          background: var(--surface); border: 1px solid var(--border);
          border-radius: 16px; padding: 2.5rem;
          display: flex; flex-direction: column; gap: 1.5rem;
          animation: fadeUp 0.5s ease both;
        }
        .logo { font-family: 'Georgia', serif; font-size: 1.2rem; color: var(--green); text-decoration: none; width: fit-content; }
        .title { font-family: 'Georgia', serif; font-size: 1.8rem; color: var(--text); }
        .sub { font-size: 0.88rem; color: var(--muted); margin-top: -1rem; }

        .form { display: flex; flex-direction: column; gap: 1.25rem; }
        .field { display: flex; flex-direction: column; gap: 0.4rem; }
        .label { font-size: 0.82rem; font-weight: 600; color: var(--text); letter-spacing: 0.02em; }

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

        .strength-row { display: flex; align-items: center; gap: 0.75rem; }
        .strength-bar { display: flex; gap: 4px; flex: 1; }
        .strength-seg { height: 3px; flex: 1; border-radius: 99px; transition: background 0.3s; }
        .strength-label { font-size: 0.75rem; font-weight: 600; min-width: 40px; }

        .terms {
          font-size: 0.75rem; color: var(--muted); line-height: 1.6;
          border-left: 2px solid var(--border); padding-left: 0.75rem;
        }

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
