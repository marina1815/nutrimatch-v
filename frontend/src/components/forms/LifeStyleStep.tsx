import { Select } from "@/components/ui/Select";
import { ACTIVITY_OPTIONS, GOAL_OPTIONS, LIFESTYLE_OPTIONS } from "@/lib/constants";
import { ActivityLevel, Goal, LifestyleType, UserProfile } from "@/lib/types";

type Props = {
  data: UserProfile;
  setData: React.Dispatch<React.SetStateAction<UserProfile>>;
  errors?: {
    activityLevel?: string;
    lifestyleType?: string;
    goal?: string;
    maxReadyTime?: string;
  };
};

export function LifestyleStep({ data, setData, errors }: Props) {
  return (
    <>
      <Select
        label="Niveau d'activite"
        value={data.lifestyle.activityLevel}
        options={ACTIVITY_OPTIONS}
        onChange={(e) =>
          setData((prev) => ({
            ...prev,
            lifestyle: {
              ...prev.lifestyle,
              activityLevel: e.target.value as ActivityLevel,
            },
          }))
        }
        error={errors?.activityLevel}
      />

      <Select
        label="Mode de vie"
        value={data.lifestyle.lifestyleType}
        options={LIFESTYLE_OPTIONS}
        onChange={(e) =>
          setData((prev) => ({
            ...prev,
            lifestyle: {
              ...prev.lifestyle,
              lifestyleType: e.target.value as LifestyleType,
            },
          }))
        }
        error={errors?.lifestyleType}
      />

      <Select
        label="Objectif"
        value={data.lifestyle.goal}
        options={GOAL_OPTIONS}
        onChange={(e) =>
          setData((prev) => ({
            ...prev,
            lifestyle: { ...prev.lifestyle, goal: e.target.value as Goal },
          }))
        }
        error={errors?.goal}
      />

      <div className="nm-field">
        <label className="nm-label">Temps maximal de preparation (minutes)</label>
        <input
          className={`nm-input ${errors?.maxReadyTime ? "nm-input-error" : ""}`}
          type="number"
          min={5}
          max={240}
          value={data.lifestyle.maxReadyTime ?? ""}
          onChange={(e) =>
            setData((prev) => ({
              ...prev,
              lifestyle: {
                ...prev.lifestyle,
                maxReadyTime: Number(e.target.value) || "",
              },
            }))
          }
        />
        {errors?.maxReadyTime && <span className="nm-error">{errors.maxReadyTime}</span>}
      </div>
    </>
  );
}
