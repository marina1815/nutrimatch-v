import { Select } from "@/components/ui/Select";
import { ACTIVITY_OPTIONS, GOAL_OPTIONS, LIFESTYLE_OPTIONS } from "@/lib/constants";
import { UserProfile } from "@/lib/types";

type Props = {
  data: UserProfile;
  setData: React.Dispatch<React.SetStateAction<UserProfile>>;
  errors?: {
    activityLevel?: string;
    lifestyleType?: string;
    goal?: string;
  };
};

export function LifestyleStep({ data, setData, errors }: Props) {
  return (
    <>
      <Select
        label="Niveau d’activité"
        value={data.lifestyle.activityLevel}
        options={ACTIVITY_OPTIONS}
        onChange={(e) =>
          setData((prev) => ({
            ...prev,
            lifestyle: { ...prev.lifestyle, activityLevel: e.target.value as any },
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
            lifestyle: { ...prev.lifestyle, lifestyleType: e.target.value as any },
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
            lifestyle: { ...prev.lifestyle, goal: e.target.value as any },
          }))
        }
        error={errors?.goal}
      />
    </>
  );
}