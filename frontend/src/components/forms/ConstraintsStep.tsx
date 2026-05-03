import { Checkbox } from "@/components/ui/Checkbox";
import { IngredientAutocompleteInput } from "@/components/forms/IngredientAutocompleteInput";
import { COMMON_ALLERGIES, COMMON_CONDITIONS } from "@/lib/constants";
import { labelAllergyValue, labelConditions } from "@/lib/display-labels";
import { ChronicDisease, Condition, Intolerance, ProfileTaxonomy, UserProfile } from "@/lib/types";
import { useId } from "react";

type Props = {
  data: UserProfile;
  setData: React.Dispatch<React.SetStateAction<UserProfile>>;
  errors?: {
    allergies?: string;
    conditions?: string;
    excludedIngredients?: string;
    chronicDiseases?: string;
    medications?: string;
  };
  taxonomy?: ProfileTaxonomy | null;
};

export function ConstraintsStep({ data, setData, errors, taxonomy }: Props) {
  const medicationsId = useId();
  const chronicConditionKeys = new Set<Condition>([
    "diabetes",
    "hypertension",
    "cardiac",
    "renal_failure",
    "hypercholesterolemia",
    "digestive_sensitivity",
  ]);
  const rawAllergyOptions = taxonomy?.allergies?.length ? taxonomy.allergies : COMMON_ALLERGIES;
  const conditionOptions = taxonomy?.conditions?.length ? taxonomy.conditions : COMMON_CONDITIONS;

  const allergyLabel = (value: string, fallback: string) => {
    const label = labelAllergyValue(value, fallback);
    return label === "-" ? "" : label;
  };
  const visibleAllergyOptions = rawAllergyOptions.reduce<typeof rawAllergyOptions>((items, item) => {
    const label = allergyLabel(item.value, item.label);
    if (!label) {
      return items;
    }
    const duplicate = items.some((existing) =>
      allergyLabel(existing.value, existing.label).toLowerCase() === label.toLowerCase(),
    );
    return duplicate ? items : [...items, item];
  }, []);
  const conditionLabel = (value: string, fallback: string) => {
    const label = labelConditions([value]);
    return label === "-" ? fallback : label;
  };

  const toggleArrayValue = (
    section: "allergies" | "conditions",
    value: string,
  ) => {
    setData((prev) => {
      const current = prev.constraints[section];
      const typedValue =
        section === "allergies" ? (value as Intolerance) : (value as Condition);
      const exists = current.includes(typedValue as never);

      return {
        ...prev,
        constraints: {
          ...prev.constraints,
          [section]: exists
            ? current.filter((item) => item !== typedValue)
            : [...current, typedValue],
        },
      };
    });
  };

  const toggleCondition = (value: string) => {
    setData((prev) => {
      const condition = value as Condition;
      const exists = prev.constraints.conditions.includes(condition);
      const nextConditions = exists
        ? prev.constraints.conditions.filter((item) => item !== condition)
        : [...prev.constraints.conditions, condition];
      const nextChronicDiseases = nextConditions.filter((item): item is ChronicDisease =>
        chronicConditionKeys.has(item),
      );

      return {
        ...prev,
        constraints: {
          ...prev.constraints,
          conditions: nextConditions,
          hasChronicDisease: nextChronicDiseases.length > 0,
          chronicDiseases: nextChronicDiseases,
        },
      };
    });
  };

  return (
    <div className="nm-stack">
      <div className="nm-field">
        <label className="nm-label">Sensibilités alimentaires</label>
        <span className="nm-help">
          Allergies et intolérances réellement présentes dans le catalogue. Chaque choix bloque les recettes concernées.
        </span>
        <div className="nm-check-grid">
          {visibleAllergyOptions.map((item) => (
            <Checkbox
              key={item.value}
              label={allergyLabel(item.value, item.label)}
              checked={data.constraints.allergies.includes(item.value)}
              onChange={() => toggleArrayValue("allergies", item.value)}
            />
          ))}
        </div>
        {errors?.allergies && <span className="nm-error">{errors.allergies}</span>}
      </div>

      <div className="nm-field">
        <label className="nm-label">Maladies et contraintes santé</label>
        <span className="nm-help">
          Ces choix sont des contraintes fortes: une recette incompatible est rejetée avant toute sélection.
        </span>
        <div className="nm-check-grid">
          {conditionOptions.map((item) => (
            <Checkbox
              key={item.value}
              label={conditionLabel(item.value, item.label)}
              checked={data.constraints.conditions.includes(item.value as Condition)}
              onChange={() => toggleCondition(item.value)}
            />
          ))}
        </div>
        {errors?.conditions && <span className="nm-error">{errors.conditions}</span>}
      </div>

      <IngredientAutocompleteInput
        label="Ingrédients à exclure strictement"
        placeholder="Cherche: porc, crevette, sucre..."
        values={data.constraints.excludedIngredients}
        onChange={(nextValues) =>
          setData((prev) => ({
            ...prev,
            constraints: {
              ...prev.constraints,
              excludedIngredients: nextValues,
            },
          }))
        }
        error={errors?.excludedIngredients}
        maxItems={30}
        helperText="Contrainte forte: seuls les ingrédients reconnus par le catalogue sont envoyés au moteur."
      />

      <div className="nm-field">
        <label className="nm-label">Prends-tu des médicaments ?</label>
        <div className="nm-inline-actions">
          <button
            type="button"
            className={`nm-link-btn ${data.constraints.takesMedication ? "nm-link-btn-primary" : ""}`}
            onClick={() =>
              setData((prev) => ({
                ...prev,
                constraints: { ...prev.constraints, takesMedication: true },
              }))
            }
          >
            Oui
          </button>

          <button
            type="button"
            className={`nm-link-btn ${!data.constraints.takesMedication ? "nm-link-btn-primary" : ""}`}
            onClick={() =>
              setData((prev) => ({
                ...prev,
                constraints: {
                  ...prev.constraints,
                  takesMedication: false,
                  medications: "",
                },
              }))
            }
          >
            Non
          </button>
        </div>
      </div>

      {data.constraints.takesMedication && (
        <div className="nm-field">
          <label className="nm-label" htmlFor={medicationsId}>Lesquels ?</label>
          <input
            id={medicationsId}
            className={`nm-input ${errors?.medications ? "nm-input-error" : ""}`}
            maxLength={250}
            placeholder="Ex: metformine, antihypertenseur..."
            value={data.constraints.medications}
            onChange={(e) =>
              setData((prev) => ({
                ...prev,
                constraints: {
                  ...prev.constraints,
                  medications: e.target.value,
                },
              }))
            }
          />
          {errors?.medications && <span className="nm-error">{errors.medications}</span>}
        </div>
      )}
    </div>
  );
}
