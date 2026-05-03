"use client";

import Link from "next/link";
import { useEffect, useRef } from "react";

export default function LandingPage() {
  const blobRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleMouseMove = (event: MouseEvent) => {
      if (!blobRef.current) {
        return;
      }
      blobRef.current.style.transform = `translate(${event.clientX - 200}px, ${event.clientY - 200}px)`;
    };
    window.addEventListener("mousemove", handleMouseMove);
    return () => window.removeEventListener("mousemove", handleMouseMove);
  }, []);

  return (
    <main className="landing">
      <div className="blob" ref={blobRef} />

      <nav className="nav">
        <span className="logo">NutriMatch</span>
        <div className="nav-links">
          <Link href="/login" className="nav-link">Se connecter</Link>
          <Link href="/register" className="nav-btn">Commencer</Link>
        </div>
      </nav>

      <section className="hero">
        <div className="hero-tag">
          <span className="dot" />
          Recommandations sûres depuis le catalogue local
        </div>

        <h1 className="hero-title">
          Mange mieux,<br />
          <em>sans deviner.</em>
        </h1>

        <p className="hero-sub">
          NutriMatch construit ton profil nutritionnel, applique tes allergies,
          contraintes santé et préférences, puis propose uniquement des repas compatibles.
          L&apos;IA explique les choix, elle ne contourne jamais les règles de sécurité.
        </p>

        <div className="hero-actions">
          <Link href="/register" className="btn-primary">
            Construire mon profil
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
              <path d="M3 8h10M9 4l4 4-4 4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          </Link>
          <Link href="/onboarding" className="btn-ghost">Voir le parcours</Link>
        </div>

        <div className="stats">
          {[
            { value: "3 piliers", label: "Préférences · Mode de vie · Contraintes" },
            { value: "20/jour", label: "Recettes sûres renouvelées toutes les 24h" },
            { value: "Priorité santé", label: "Allergies et maladies toujours respectées" },
          ].map((stat) => (
            <div key={stat.value} className="stat">
              <span className="stat-value">{stat.value}</span>
              <span className="stat-label">{stat.label}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="how">
        <h2 className="section-title">Comment ça marche</h2>
        <div className="steps">
          {[
            {
              n: "01",
              title: "Tu remplis ton profil",
              desc: "Âge, sexe, taille, poids, mode de vie, objectif, goûts et contraintes santé.",
            },
            {
              n: "02",
              title: "Le moteur filtre les recettes",
              desc: "Le backend applique d'abord les règles dures: allergies, exclusions, maladies et médicaments.",
            },
            {
              n: "03",
              title: "Tu reçois tes suggestions",
              desc: "20 repas sûrs sont proposés pour 24h, avec une explication claire pour chaque recette.",
            },
          ].map((step) => (
            <div key={step.n} className="step">
              <span className="step-n">{step.n}</span>
              <h3 className="step-title">{step.title}</h3>
              <p className="step-desc">{step.desc}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="cta-banner">
        <h2>Prêt à manger plus intelligemment ?</h2>
        <Link href="/register" className="btn-primary">
          Démarrer gratuitement
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <path d="M3 8h10M9 4l4 4-4 4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </Link>
      </section>

      <footer className="footer">
        <span className="logo">NutriMatch</span>
        <span>© 2026 · Projet sécurité logicielle</span>
      </footer>
    </main>
  );
}
