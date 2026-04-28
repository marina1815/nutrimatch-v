"use client";
import Link from "next/link";
import { useEffect, useRef } from "react";

export default function LandingPage() {
  const blobRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!blobRef.current) return;
      blobRef.current.style.transform = `translate(${e.clientX - 200}px, ${e.clientY - 200}px)`;
    };
    window.addEventListener("mousemove", handleMouseMove);
    return () => window.removeEventListener("mousemove", handleMouseMove);
  }, []);

  return (
    <main className="landing">
      {/* Ambient blob that follows cursor */}
      <div className="blob" ref={blobRef} />

      {/* Navbar */}
      <nav className="nav">
        <span className="logo">NutriMatch</span>
        <div className="nav-links">
          <Link href="/login" className="nav-link">Sign in</Link>
          <Link href="/register" className="nav-btn">Get started</Link>
        </div>
      </nav>

      {/* Hero */}
      <section className="hero">
        <div className="hero-tag">
          <span className="dot" />
          AI-powered meal matching
        </div>

        <h1 className="hero-title">
          Eat right,<br />
          <em>effortlessly.</em>
        </h1>

        <p className="hero-sub">
          NutriMatch builds your personal nutrition profile — your weight, lifestyle,
          allergies, goals — and finds the meals that actually fit you.
          No generic diets. No guesswork.
        </p>

        <div className="hero-actions">
          <Link href="/register" className="btn-primary">
            Build my profile
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
              <path d="M3 8h10M9 4l4 4-4 4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
            </svg>
          </Link>
          <Link href="/onboarding" className="btn-ghost">See how it works</Link>
        </div>

        {/* Stats row */}
        <div className="stats">
          {[
            { value: "3 pillars", label: "Preferences · Lifestyle · Constraints" },
            { value: "100%", label: "Personalised to your profile" },
            { value: "Fail-safe", label: "Allergies always respected" },
          ].map((s) => (
            <div key={s.value} className="stat">
              <span className="stat-value">{s.value}</span>
              <span className="stat-label">{s.label}</span>
            </div>
          ))}
        </div>
      </section>

      {/* How it works */}
      <section className="how">
        <h2 className="section-title">How it works</h2>
        <div className="steps">
          {[
            {
              n: "01",
              title: "Fill your profile",
              desc: "Tell us your sex, age, weight, height, activity level, objectives and food preferences.",
            },
            {
              n: "02",
              title: "We build your nutrition plan",
              desc: "We calculate your caloric needs, apply allergy filters and build a unique nutritional fingerprint.",
            },
            {
              n: "03",
              title: "Get matched meals",
              desc: "Our hybrid engine — database + AI — returns personalised meal suggestions that respect every constraint.",
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

      {/* CTA banner */}
      <section className="cta-banner">
        <h2>Ready to eat smarter?</h2>
        <Link href="/register" className="btn-primary">
          Start for free
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <path d="M3 8h10M9 4l4 4-4 4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
        </Link>
      </section>

      {/* Footer */}
      <footer className="footer">
        <span className="logo">NutriMatch</span>
        <span>© 2025 — Software Security Project</span>
      </footer>
    </main>
  );
}
