"use client";

import { useEffect, useMemo, useState } from "react";
import { suggestIngredients } from "@/lib/api";
import { parseCommaSeparatedList } from "@/lib/profile-normalization";

type Props = {
  label: string;
  placeholder: string;
  values: string[];
  onChange: (values: string[]) => void;
  error?: string;
  maxItems: number;
};

function readTrailingToken(value: string): string {
  const parts = value.split(",");
  return parts[parts.length - 1]?.trim() ?? "";
}

export function IngredientAutocompleteInput({
  label,
  placeholder,
  values,
  onChange,
  error,
  maxItems,
}: Props) {
  const [draft, setDraft] = useState(values.join(", "));
  const [suggestions, setSuggestions] = useState<string[]>([]);

  useEffect(() => {
    setDraft(values.join(", "));
  }, [values]);

  const trailingToken = useMemo(() => readTrailingToken(draft), [draft]);

  useEffect(() => {
    let cancelled = false;

    if (trailingToken.length < 2) {
      setSuggestions([]);
      return;
    }

    const timer = window.setTimeout(async () => {
      try {
        const items = await suggestIngredients(trailingToken, 5);
        if (!cancelled) {
          setSuggestions(items.filter((item) => item !== trailingToken.toLowerCase()));
        }
      } catch {
        if (!cancelled) {
          setSuggestions([]);
        }
      }
    }, 180);

    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [trailingToken]);

  const applyDraft = (nextDraft: string) => {
    setDraft(nextDraft);
    onChange(parseCommaSeparatedList(nextDraft, maxItems, 50));
  };

  const applySuggestion = (suggestion: string) => {
    const parts = draft.split(",");
    parts[parts.length - 1] = ` ${suggestion}`;
    const nextDraft = parts.join(",").replace(/^ /, "");
    applyDraft(nextDraft);
    setSuggestions([]);
  };

  return (
    <div className="nm-field">
      <label className="nm-label">{label}</label>
      <input
        className={`nm-input ${error ? "nm-input-error" : ""}`}
        placeholder={placeholder}
        value={draft}
        onChange={(event) => applyDraft(event.target.value)}
      />
      {suggestions.length > 0 && (
        <div className="nm-inline-actions">
          {suggestions.map((suggestion) => (
            <button
              key={suggestion}
              type="button"
              className="nm-link-btn"
              onClick={() => applySuggestion(suggestion)}
            >
              {suggestion}
            </button>
          ))}
        </div>
      )}
      {error && <span className="nm-error">{error}</span>}
    </div>
  );
}
