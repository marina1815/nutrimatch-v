import React from "react";

type Props = React.InputHTMLAttributes<HTMLInputElement> & {
  label: string;
  error?: string;
};

export function Input({ label, error, ...props }: Props) {
  const generatedID = React.useId();
  const id = props.id ?? generatedID;

  return (
    <div className="nm-field">
      <label className="nm-label" htmlFor={id}>{label}</label>
      <input id={id} className={`nm-input ${error ? "nm-input-error" : ""}`} {...props} />
      {error && <span className="nm-error">{error}</span>}
    </div>
  );
}
