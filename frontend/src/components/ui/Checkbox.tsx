import React from "react";

type Props = {
  label: string;
  checked: boolean;
  onChange: () => void;
};

export function Checkbox({ label, checked, onChange }: Props) {
  return (
    <label className="nm-check">
      <input type="checkbox" checked={checked} onChange={onChange} />
      <span>{label}</span>
    </label>
  );
}