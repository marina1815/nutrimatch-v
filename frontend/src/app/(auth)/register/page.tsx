"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { ApiError, getCurrentSession, registerUser } from "@/lib/api";
import { getSafeErrorMessage } from "@/lib/ui-errors";

interface FormState {
  email: string;
  password: string;
  confirm: string;
}

interface FormErrors {
  email?: string;
  password?: string;
  confirm?: string;
  form?: string;
}

export default function RegisterPage() {
  const router = useRouter();
  const [form, setForm] = useState<FormState>({ email: "", password: "", confirm: "" });
  const [errors, setErrors] = useState<FormErrors>({});
  const [loading, setLoading] = useState(false);

  const setField = (field: keyof FormState) => (event: React.ChangeEvent<HTMLInputElement>) => {
    setForm((current) => ({ ...current, [field]: event.target.value }));
    setErrors((current) => ({ ...current, [field]: undefined, form: undefined }));
  };

  const validate = (): FormErrors => {
    const nextErrors: FormErrors = {};
    if (!form.email.includes("@")) nextErrors.email = "Entre une adresse email valide";
    if (form.password.length < 12) nextErrors.password = "Le mot de passe doit contenir au moins 12 caracteres";
    if (form.confirm !== form.password) nextErrors.confirm = "Les mots de passe ne correspondent pas";
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
        // Le vrai nom complet est collecte dans l'onboarding; on garde ici un libelle neutre
        // pour satisfaire le contrat backend sans dupliquer le champ dans l'UI.
        name: "Utilisateur",
        email: form.email,
        password: form.password,
      });
      try {
        const session = await getCurrentSession();
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

  const strengthLabel = ["", "Faible", "Correct", "Bon", "Fort"][strength];
  const strengthClass = strength > 0 ? `strength-${strength}` : "";

  return (
    <main className="page">
      <div className="card">
        <Link href="/" className="logo">NutriMatch</Link>

        <h1 className="title">Cree ton compte</h1>
        <p className="sub">Commence a construire ton profil nutritionnel personnalise</p>

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
              onChange={setField("email")}
            />
            {errors.email && <span className="error">{errors.email}</span>}
          </div>

          <div className="field">
            <label className="label" htmlFor="password">Mot de passe</label>
            <input
              id="password"
              type="password"
              autoComplete="new-password"
              placeholder="12 caracteres minimum"
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
            <label className="label" htmlFor="confirm">Confirmer le mot de passe</label>
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
            En creant un compte, tu acceptes que tes donnees servent uniquement a generer
            des recommandations personnalisees. Elles ne sont pas revendues a des tiers.
          </p>

          {errors.form && <span className="error">{errors.form}</span>}

          <button type="submit" className="btn" disabled={loading}>
            {loading ? <span className="spinner" /> : "Creer le compte"}
          </button>
        </form>

        <p className="switch">
          Deja un compte ?{" "}
          <Link href="/login" className="switch-link">Se connecter</Link>
        </p>
      </div>
    </main>
  );
}
