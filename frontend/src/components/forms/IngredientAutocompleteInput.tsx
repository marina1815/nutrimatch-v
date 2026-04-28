"use client";

import { useEffect, useState } from "react";
import { suggestIngredients } from "@/lib/api";
import { getIngredientLabel } from "@/lib/ingredient-labels";

type Props = {
  label: string;
  placeholder: string;
  values: string[];
  onChange: (values: string[]) => void;
  error?: string;
  maxItems: number;
  helperText?: string;
};

function normalizeChoice(value: string): string {
  return value.replace(/\s+/g, " ").trim().toLowerCase();
}

export function IngredientAutocompleteInput({
  label,
  placeholder,
  values,
  onChange,
  error,
  maxItems,
  helperText,
}: Props) {
  const [query, setQuery] = useState("");
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const [focused, setFocused] = useState(false);
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);
  const isFull = values.length >= maxItems;
  const showDropdown = !isFull && (focused || query.trim().length > 0);

  useEffect(() => {
    let cancelled = false;
    const cleanedQuery = query.trim();

    if (cleanedQuery.length < 2 || isFull) {
      setSuggestions([]);
      setLoading(false);
      setSearched(false);
      return;
    }

    const timer = window.setTimeout(async () => {
      try {
        setLoading(true);
        setSearched(false);
        const items = await suggestIngredients(cleanedQuery, 8);
        if (!cancelled) {
          const selected = new Set(values.map(normalizeChoice));
          setSuggestions(items.filter((item) => !selected.has(normalizeChoice(item))));
          setSearched(true);
        }
      } catch {
        if (!cancelled) {
          setSuggestions([]);
          setSearched(true);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }, 180);

    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [query, values, isFull]);

  const addChoice = (choice: string) => {
    const cleaned = normalizeChoice(choice);
    if (!cleaned || isFull) {
      return;
    }

    const exists = values.some((value) => normalizeChoice(value) === cleaned);
    if (exists) {
      setQuery("");
      setSuggestions([]);
      return;
    }

    onChange([...values, cleaned].slice(0, maxItems));
    setQuery("");
    setSuggestions([]);
  };

  const removeChoice = (choice: string) => {
    const cleaned = normalizeChoice(choice);
    onChange(values.filter((value) => normalizeChoice(value) !== cleaned));
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") {
      event.preventDefault();
      if (suggestions[0]) {
        addChoice(suggestions[0]);
      }
    }
  };

  return (
    <div className="nm-field">
      <label className="nm-label">{label}</label>
      {values.length > 0 && (
        <div className="nm-chip-list" aria-label={`${label} selection`}>
          {values.map((value) => (
            <button
              key={value}
              type="button"
              className="nm-chip"
              onClick={() => removeChoice(value)}
              title="Retirer"
            >
              {getIngredientLabel(value)}
              <span aria-hidden="true">x</span>
            </button>
          ))}
        </div>
      )}
      <div className="nm-combobox">
        <input
          className={`nm-input nm-combobox-input ${error ? "nm-input-error" : ""}`}
          placeholder={placeholder}
          value={query}
          disabled={isFull}
          onFocus={() => setFocused(true)}
          onBlur={() => window.setTimeout(() => setFocused(false), 120)}
          onChange={(event) => setQuery(event.target.value)}
          onKeyDown={handleKeyDown}
          autoComplete="off"
          role="combobox"
          aria-expanded={focused}
          aria-autocomplete="list"
        />
        {showDropdown && (
          <div className="nm-suggestion-list" role="listbox">
            {query.trim().length < 2 && (
              <div className="nm-suggestion-empty">
                Tape au moins 2 lettres pour charger les ingredients acceptes par Spoonacular.
              </div>
            )}
            {loading && <div className="nm-suggestion-empty">Recherche dans Spoonacular...</div>}
            {!loading && suggestions.map((suggestion) => (
              <button
                key={suggestion}
                type="button"
                className="nm-suggestion"
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => addChoice(suggestion)}
                role="option"
              >
                <span>{getIngredientLabel(suggestion)}</span>
                <small>{suggestion}</small>
              </button>
            ))}
            {!loading && searched && query.trim().length >= 2 && suggestions.length === 0 && (
              <div className="nm-suggestion-empty">
                Aucun ingredient reconnu pour cette recherche.
              </div>
            )}
          </div>
        )}
      </div>
      <span className="nm-help">
        {helperText || "Recherche un ingredient puis choisis une option Spoonacular. Aucune saisie libre n'est ajoutee."}
      </span>
      {isFull && <span className="nm-help">Limite atteinte: {maxItems} elements.</span>}
      {error && <span className="nm-error">{error}</span>}
    </div>
  );
}
