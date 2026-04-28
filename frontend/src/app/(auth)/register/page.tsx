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
  const strengthClass = strength > 0 ? `strength-${strength}` : "";

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
                      className={`strength-seg ${index <= strength ? strengthClass : ""}`}
                    />
                  ))}
                </div>
                <span className={`strength-label ${strengthClass}`}>{strengthLabel}</span>
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
    </main>
  );
}
