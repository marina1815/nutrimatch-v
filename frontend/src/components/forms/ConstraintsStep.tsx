import { Checkbox } from "@/components/ui/Checkbox";
import {
  CHRONIC_DISEASE_OPTIONS,
  INTOLERANCE_OPTIONS,
  COMMON_CONDITIONS,
} from "@/lib/constants";
import { UserProfile, ChronicDisease } from "@/lib/types";

type Props = {
  data: UserProfile;
  setData: React.Dispatch<React.SetStateAction<UserProfile>>;
  errors?: {
    chronicDiseases?: string;
    medications?: string;
  };
};

export function ConstraintsStep({ data, setData, errors }: Props) {
  const toggleArrayValue = (
    section: "allergies" | "conditions",
    value: string
  ) => {
    setData((prev) => {
      const current = prev.constraints[section];
      const exists = current.includes(value);

      return {
        ...prev,
        constraints: {
          ...prev.constraints,
          [section]: exists ? current.filter((item) => item !== value) : [...current, value],
        },
      };
    });
  };

  const toggleDisease = (value: string) => {
    setData((prev) => {
      const exists = prev.constraints.chronicDiseases.includes(value as ChronicDisease);

      return {
        ...prev,
        constraints: {
          ...prev.constraints,
          chronicDiseases: exists
            ? prev.constraints.chronicDiseases.filter((item) => item !== value)
            : [...prev.constraints.chronicDiseases, value as ChronicDisease],
        },
      };
    });
  };

  return (
    <div className="nm-stack">
      <div className="nm-field">
        <label className="nm-label">Allergies</label>
        <div className="nm-check-grid">
          {INTOLERANCE_OPTIONS.map((item) => (
            <Checkbox
              key={item.value}
              label={item.label}
              checked={data.constraints.allergies.includes(item.value)}
              onChange={() => toggleArrayValue("allergies", item.value)}
            />
          ))}
        </div>
      </div>

      <div className="nm-field">
        <label className="nm-label">Autres contraintes de santé</label>
        <div className="nm-check-grid">
          {COMMON_CONDITIONS.map((item) => (
            <Checkbox
              key={item}
              label={item}
              checked={data.constraints.conditions.includes(item)}
              onChange={() => toggleArrayValue("conditions", item)}
            />
          ))}
        </div>
      </div>

      <div className="nm-field">
        <label className="nm-label">Ingrédients à exclure</label>
        <input
          className="nm-input"
          placeholder="Porc, crevettes, sucre..."
          value={data.constraints.excludedIngredients.join(", ")}
          onChange={(e) =>
            setData((prev) => ({
              ...prev,
              constraints: {
                ...prev.constraints,
                excludedIngredients: e.target.value
                  .split(",")
                  .map((item) => item.trim())
                  .filter(Boolean),
              },
            }))
          }
        />
      </div>

      <div className="nm-field">
        <label className="nm-label">As-tu une maladie chronique ?</label>
        <div className="nm-inline-actions">
          <button
            type="button"
            className={`nm-link-btn ${data.constraints.hasChronicDisease ? "nm-link-btn-primary" : ""}`}
            onClick={() =>
              setData((prev) => ({
                ...prev,
                constraints: { ...prev.constraints, hasChronicDisease: true },
              }))
            }
          >
            Oui
          </button>

          <button
            type="button"
            className={`nm-link-btn ${!data.constraints.hasChronicDisease ? "nm-link-btn-primary" : ""}`}
            onClick={() =>
              setData((prev) => ({
                ...prev,
                constraints: {
                  ...prev.constraints,
                  hasChronicDisease: false,
                  chronicDiseases: [],
                },
              }))
            }
          >
            Non
          </button>
        </div>
      </div>

      {data.constraints.hasChronicDisease && (
        <div className="nm-field">
          <label className="nm-label">Laquelle ?</label>
          <div className="nm-check-grid">
            {CHRONIC_DISEASE_OPTIONS.map((item) => (
              <Checkbox
                key={item.value}
                label={item.label}
                checked={data.constraints.chronicDiseases.includes(item.value as ChronicDisease)}
                onChange={() => toggleDisease(item.value)}
              />
            ))}
          </div>
          {errors?.chronicDiseases && (
            <span className="nm-error">{errors.chronicDiseases}</span>
          )}
        </div>
      )}

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
          <label className="nm-label">Lesquels ?</label>
          <input
            className={`nm-input ${errors?.medications ? "nm-input-error" : ""}`}
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