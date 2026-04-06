"use client";

import Link from "next/link";
import { RecommendationList } from "@/components/results/RecommendationList";
import { mockMeals } from "@/lib/mock-data";

export default function ResultsPage() {
  return (
    <main className="nm-page">
      <section className="nm-results-shell">
        <div className="nm-header-row">
          <div>
            <span className="nm-logo">NutriMatch</span>
            <h1 className="nm-title">Your meal recommendations</h1>
            <p className="nm-sub">
              Based on your profile, preferences, lifestyle and dietary constraints.
            </p>
          </div>

          <div className="nm-inline-actions">
            <Link href="/onboarding" className="nm-link-btn">Edit profile</Link>
            <Link href="/profile" className="nm-link-btn nm-link-btn-primary">View profile</Link>
          </div>
        </div>

        <RecommendationList meals={mockMeals} />
      </section>
    </main>
  );
}