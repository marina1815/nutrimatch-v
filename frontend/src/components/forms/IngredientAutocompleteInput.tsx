"use client";

import { useEffect, useId, useState } from "react";
import { suggestIngredients } from "@/lib/api";
import { getIngredientLabel } from "@/lib/ingredient-labels";
import { CatalogOption } from "@/lib/types";

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
  const [suggestions, setSuggestions] = useState<CatalogOption[]>([]);
  const [focused, setFocused] = useState(false);
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);
  const inputId = useId();
  const isFull = values.length >= maxItems;
  const showDropdown = !isFull && (focused || query.trim().length > 0);
  const listboxId = `${label.toLowerCase().replace(/[^a-z0-9]+/g, "-")}-suggestions`;

  useEffect(() => {
    let cancelled = false;
    const cleanedQuery = query.trim();

    if (cleanedQuery.length < 2 || isFull) {
      return;
    }

    const timer = window.setTimeout(async () => {
      try {
        setLoading(true);
        setSearched(false);
        const items = await suggestIngredients(cleanedQuery, 8);
        if (!cancelled) {
          const selected = new Set(values.map(normalizeChoice));
          setSuggestions(items.filter((item) => !selected.has(normalizeChoice(item.value))));
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

  const addChoice = (choice: CatalogOption | string) => {
    const value = typeof choice === "string" ? choice : choice.value;
    const cleaned = normalizeChoice(value);
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
    setLoading(false);
    setSearched(false);
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

  const handleQueryChange = (nextQuery: string) => {
    setQuery(nextQuery);
    if (nextQuery.trim().length < 2) {
      setSuggestions([]);
      setLoading(false);
      setSearched(false);
    }
  };

  return (
    <div className="nm-field">
      <label className="nm-label" htmlFor={inputId}>{label}</label>
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
          id={inputId}
          className={`nm-input nm-combobox-input ${error ? "nm-input-error" : ""}`}
          placeholder={placeholder}
          value={query}
          disabled={isFull}
          onFocus={() => setFocused(true)}
          onBlur={() => window.setTimeout(() => setFocused(false), 120)}
          onChange={(event) => handleQueryChange(event.target.value)}
          onKeyDown={handleKeyDown}
          autoComplete="off"
          role="combobox"
          aria-controls={listboxId}
          aria-expanded={showDropdown}
          aria-autocomplete="list"
        />
        {showDropdown && (
          <div id={listboxId} className="nm-suggestion-list" role="listbox">
            {query.trim().length < 2 && (
              <div className="nm-suggestion-empty">
                Tape au moins 2 lettres pour charger les ingrédients du catalogue NutriMatch.
              </div>
            )}
            {loading && <div className="nm-suggestion-empty">Recherche dans le catalogue...</div>}
            {!loading && suggestions.map((suggestion) => (
              <button
                key={suggestion.value}
                type="button"
                className="nm-suggestion"
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => addChoice(suggestion)}
                role="option"
                aria-selected={false}
              >
                <span>{suggestion.label || getIngredientLabel(suggestion.value)}</span>
              </button>
            ))}
            {!loading && searched && query.trim().length >= 2 && suggestions.length === 0 && (
              <div className="nm-suggestion-empty">
                Aucun ingrédient reconnu pour cette recherche.
              </div>
            )}
          </div>
        )}
      </div>
      <span className="nm-help">
        {helperText || "Recherche un ingrédient puis choisis une option du catalogue. Aucune saisie libre n'est ajoutée."}
      </span>
      {isFull && <span className="nm-help">Limite atteinte: {maxItems} éléments.</span>}
      {error && <span className="nm-error">{error}</span>}
    </div>
  );
}
