import React from "react";

type Props = React.InputHTMLAttributes<HTMLInputElement> & {
  label: string;
  error?: string;
};

export function Input({ label, error, ...props }: Props) {
  return (
    <div className="nm-field">
      <label className="nm-label">{label}</label>
      <input className={`nm-input ${error ? "nm-input-error" : ""}`} {...props} />
      {error && <span className="nm-error">{error}</span>}
    </div>
  );
}