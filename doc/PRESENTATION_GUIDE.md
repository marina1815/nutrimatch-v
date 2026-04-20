# Presentation Guide: NutriMatch

Welcome to the NutriMatch presentation. This guide will help you showcase the robust architecture, security measures, and the "Human-First" approach of our application.

## 1. Project Vision (The 3 Pillars)
NutriMatch goes beyond random recipes. It builds a **Nutritional Profile** based on:
1.  **Preferences**: User likes, dislikes, and meal styles.
2.  **Lifestyle**: Activity level, morphology, and goals (Weight loss, Muscle gain, etc.).
3.  **Health Constraints**: Allergies, chronic diseases, and medical rules.

## 2. Technical Architecture: Hybrid Treatment
Explain how we combine performance and safety:

### A. Data Retrieval & Validation
- **Frontend**: Multi-step onboarding with real-time validation ([lib/validation.ts](file:///c:/Users/DELL/Desktop/nutrimatch/frontend/src/lib/validation.ts)).
- **Backend**: Strict DTO validation and persistence in PostgreSQL.

### B. The Hybrid Recommendation Engine
Located in [recommendation_service.go](file:///c:/Users/DELL/Desktop/nutrimatch/backend/internal/services/recommendation_service.go):
1.  **Deterministic Firewall (Fail-Safe)**: Before any AI intervention, every recipe from Spoonacular is filtered against medical rules and nutritional thresholds. If a recipe contains a blocked allergen, it is **discarded immediately**.
2.  **AI Affinement**: Large Language Models (Gemini) are used to **re-rank** and **explain** already-approved recipes, adding a layer of "intelligence" without compromising safety.
3.  **Traceability**: Every recommendation includes a "Why" (explanation) and a traceability log of why it passed or failed.

## 3. Security Hardening (STRIDE)
Refer to [stride.md](file:///c:/Users/DELL/Desktop/nutrimatch/doc/stride.md) for full details. Here are the highlights:

| Category | Measure in NutriMatch |
| :--- | :--- |
| **Spoofing** | JWT Authentication and CSRF tokens enforced on all profile operations. |
| **Tampering** | Secure API search plans that sanitize user inputs before calling external APIs. |
| **Repudiation** | Audit trail ([audit_events](file:///c:/Users/DELL/Desktop/nutrimatch/backend/migrations/0001_init.sql#L186)) for every profile change and recommendation run. |
| **Information Disclosure** | Health data is treated with high sensitivity; PII is strictly managed. |
| **Denial of Service** | Rate limiting by IP and User ID implemented in the backend router. |
| **Elevation of Privilege** | Chokepoint architecture: all flows must pass through validation and auth middlewares. |

## 4. Live Demonstration Flow
1.  **Login/Register**: Show the secure entry.
2.  **Onboarding**: Complete the form (e.g., set "Allergy: Shrimp").
3.  **Analysis**: Show the loading state while the "Hybrid Orchestrator" runs.
4.  **Results**: Show the personalized list and point out the **Explanation** ("Selected because it fits your high-protein goal...").
5.  **Fail-Safe Check**: Verify that "Shrimp" recipes are nowhere to be found, even if they are popular.

---
*NutriMatch - S'adapte à l'humain, pas l'inverse.*
