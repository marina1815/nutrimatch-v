import { Checkbox } from "@/components/ui/Checkbox";
import { IngredientAutocompleteInput } from "@/components/forms/IngredientAutocompleteInput";
import {
  COMMON_ALLERGIES,
  COMMON_CONDITIONS,
} from "@/lib/constants";
import { ChronicDisease, Condition, Intolerance, UserProfile } from "@/lib/types";

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
};

export function ConstraintsStep({ data, setData, errors }: Props) {
  const chronicConditionKeys = new Set<Condition>(["diabetes", "hypertension", "cardiac", "renal_failure"]);

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
        <label className="nm-label">Allergies</label>
        <div className="nm-check-grid">
          {COMMON_ALLERGIES.map((item) => (
            <Checkbox
              key={item.value}
              label={item.label}
              checked={data.constraints.allergies.includes(item.value)}
              onChange={() => toggleArrayValue("allergies", item.value)}
            />
          ))}
        </div>
        {errors?.allergies && <span className="nm-error">{errors.allergies}</span>}
      </div>

      <div className="nm-field">
        <label className="nm-label">Contraintes de sante</label>
        <span className="nm-help">
          Un seul choix ici suffit: les maladies chroniques sont derivees automatiquement pour les regles medicales.
        </span>
        <div className="nm-check-grid">
          {COMMON_CONDITIONS.map((item) => (
            <Checkbox
              key={item.value}
              label={item.label}
              checked={data.constraints.conditions.includes(item.value)}
              onChange={() => toggleCondition(item.value)}
            />
          ))}
        </div>
        {errors?.conditions && <span className="nm-error">{errors.conditions}</span>}
      </div>

      <IngredientAutocompleteInput
        label="Ingredients a exclure strictement"
        placeholder="Cherche: pork, shrimp, sugar..."
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
        helperText="Contrainte forte envoyee au moteur: selectionne des ingredients reconnus pour eviter les fautes et synonymes ambigus."
      />

      <div className="nm-field">
        <label className="nm-label">Prends-tu des medicaments ?</label>
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
          <label className="nm-label">Lesquels ?</label>
          <input
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
