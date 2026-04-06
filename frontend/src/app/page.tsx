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

      <style>{`
        *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

        :root {
          --bg: #0a0f0a;
          --surface: #111811;
          --border: #1e2b1e;
          --green: #4ade80;
          --green-dim: #166534;
          --green-glow: rgba(74,222,128,0.15);
          --text: #f0fdf0;
          --muted: #6b7c6b;
          --font-display: 'Georgia', serif;
          --font-body: 'Helvetica Neue', Helvetica, sans-serif;
        }

        body { background: var(--bg); color: var(--text); font-family: var(--font-body); }

        .landing { min-height: 100vh; overflow: hidden; position: relative; }

        /* Cursor blob */
        .blob {
          position: fixed; top: 0; left: 0; width: 400px; height: 400px;
          background: radial-gradient(circle, rgba(74,222,128,0.12) 0%, transparent 70%);
          border-radius: 50%; pointer-events: none; z-index: 0;
          transition: transform 0.15s ease; will-change: transform;
        }

        /* Nav */
        .nav {
          position: sticky; top: 0; z-index: 50;
          display: flex; align-items: center; justify-content: space-between;
          padding: 1.25rem 4rem;
          background: rgba(10,15,10,0.7);
          backdrop-filter: blur(12px);
          border-bottom: 1px solid var(--border);
        }
        .logo { font-family: var(--font-display); font-size: 1.3rem; color: var(--green); letter-spacing: -0.01em; }
        .nav-links { display: flex; align-items: center; gap: 1.5rem; }
        .nav-link { color: var(--muted); text-decoration: none; font-size: 0.9rem; transition: color 0.2s; }
        .nav-link:hover { color: var(--text); }
        .nav-btn {
          background: var(--green); color: #0a0f0a; border-radius: 6px;
          padding: 0.45rem 1rem; font-size: 0.85rem; font-weight: 600;
          text-decoration: none; transition: opacity 0.2s;
        }
        .nav-btn:hover { opacity: 0.85; }

        /* Hero */
        .hero {
          position: relative; z-index: 1;
          max-width: 860px; margin: 0 auto;
          padding: 7rem 2rem 5rem;
          display: flex; flex-direction: column; gap: 2rem;
        }
        .hero-tag {
          display: inline-flex; align-items: center; gap: 0.5rem;
          background: var(--green-glow); border: 1px solid var(--green-dim);
          color: var(--green); border-radius: 999px;
          padding: 0.35rem 0.9rem; font-size: 0.8rem; width: fit-content;
          animation: fadeUp 0.6s ease both;
        }
        .dot { width: 6px; height: 6px; border-radius: 50%; background: var(--green); animation: pulse 2s infinite; }
        @keyframes pulse { 0%,100%{opacity:1} 50%{opacity:0.4} }

        .hero-title {
          font-family: var(--font-display); font-size: clamp(3rem, 8vw, 5.5rem);
          line-height: 1.05; letter-spacing: -0.03em; color: var(--text);
          animation: fadeUp 0.6s 0.1s ease both;
        }
        .hero-title em { color: var(--green); font-style: italic; }

        .hero-sub {
          max-width: 540px; color: var(--muted); line-height: 1.7; font-size: 1.05rem;
          animation: fadeUp 0.6s 0.2s ease both;
        }

        .hero-actions {
          display: flex; gap: 1rem; align-items: center; flex-wrap: wrap;
          animation: fadeUp 0.6s 0.3s ease both;
        }
        .btn-primary {
          display: inline-flex; align-items: center; gap: 0.5rem;
          background: var(--green); color: #0a0f0a;
          border-radius: 8px; padding: 0.75rem 1.5rem;
          font-weight: 700; font-size: 0.95rem; text-decoration: none;
          transition: opacity 0.2s, transform 0.2s;
        }
        .btn-primary:hover { opacity: 0.88; transform: translateY(-1px); }
        .btn-ghost {
          color: var(--muted); text-decoration: none; font-size: 0.9rem;
          border-bottom: 1px solid var(--border); padding-bottom: 1px;
          transition: color 0.2s;
        }
        .btn-ghost:hover { color: var(--text); }

        /* Stats */
        .stats {
          display: flex; gap: 2.5rem; flex-wrap: wrap; padding-top: 1rem;
          border-top: 1px solid var(--border);
          animation: fadeUp 0.6s 0.4s ease both;
        }
        .stat { display: flex; flex-direction: column; gap: 0.2rem; }
        .stat-value { font-family: var(--font-display); font-size: 1.3rem; color: var(--green); }
        .stat-label { font-size: 0.78rem; color: var(--muted); }

        /* How it works */
        .how {
          position: relative; z-index: 1;
          max-width: 860px; margin: 0 auto;
          padding: 5rem 2rem;
        }
        .section-title {
          font-family: var(--font-display); font-size: 2rem;
          margin-bottom: 3rem; color: var(--text);
        }
        .steps { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 2rem; }
        .step {
          background: var(--surface); border: 1px solid var(--border);
          border-radius: 12px; padding: 1.75rem;
          display: flex; flex-direction: column; gap: 0.75rem;
          transition: border-color 0.2s;
        }
        .step:hover { border-color: var(--green-dim); }
        .step-n { font-family: var(--font-display); font-size: 2rem; color: var(--green-dim); }
        .step-title { font-size: 1rem; font-weight: 600; color: var(--text); }
        .step-desc { font-size: 0.87rem; color: var(--muted); line-height: 1.65; }

        /* CTA Banner */
        .cta-banner {
          position: relative; z-index: 1;
          margin: 2rem auto 5rem; max-width: 860px; padding: 0 2rem;
          background: var(--surface); border: 1px solid var(--border);
          border-radius: 16px; padding: 3rem;
          display: flex; align-items: center; justify-content: space-between;
          flex-wrap: wrap; gap: 1.5rem;
        }
        .cta-banner h2 { font-family: var(--font-display); font-size: 1.8rem; }

        /* Footer */
        .footer {
          position: relative; z-index: 1;
          display: flex; justify-content: space-between; align-items: center;
          padding: 1.5rem 4rem; border-top: 1px solid var(--border);
          font-size: 0.82rem; color: var(--muted);
        }

        @keyframes fadeUp {
          from { opacity: 0; transform: translateY(16px); }
          to   { opacity: 1; transform: translateY(0); }
        }

        @media (max-width: 600px) {
          .nav { padding: 1rem 1.5rem; }
          .footer { padding: 1.5rem; flex-direction: column; gap: 0.5rem; text-align: center; }
          .cta-banner { padding: 2rem 1.5rem; }
          .stats { gap: 1.5rem; }
        }
      `}</style>
    </main>
  );
}
