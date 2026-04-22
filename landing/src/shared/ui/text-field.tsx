/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Updated
 */

import { forwardRef } from 'react';
import type { InputHTMLAttributes } from 'react';

export interface TextFieldProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  /**
   * Optional hint shown below the input until the field produces
   * an `error`. Use for format examples ("matrix.org, no port
   * required") or short guidance. Styled as muted body copy so
   * it does not shout at the visitor while they type.
   */
  description?: string;
  error?: string | undefined;
}

/**
 * Controlled, accessible single-line text input intended to be
 * used with React Hook Form's `register()`. Ref forwarding is
 * required by RHF; label / description / error wiring is handled
 * here so consumers never wire `<label htmlFor>` +
 * `aria-describedby` by hand and get it wrong.
 *
 * Colours route through the landing design tokens (`border`,
 * `primary`, `danger`, `text-muted`) so the input inherits theme
 * changes without per-form overrides.
 */
export const TextField = forwardRef<HTMLInputElement, TextFieldProps>(
  function TextField({ label, description, error, id, ...rest }, ref) {
    const inputId = id ?? `tf-${rest.name ?? 'field'}`;
    const errorId = `${inputId}-err`;
    const descId = `${inputId}-desc`;
    const describedBy = error ? errorId : description ? descId : undefined;
    return (
      <div className="flex flex-col gap-1">
        <label htmlFor={inputId} className="text-sm font-medium text-text">
          {label}
        </label>
        <input
          id={inputId}
          ref={ref}
          aria-invalid={error ? 'true' : 'false'}
          aria-describedby={describedBy}
          className={
            'rounded-md border bg-bg-surface px-3 py-2 text-sm text-text outline-none focus:ring-2 ' +
            (error
              ? 'border-danger focus:ring-danger'
              : 'border-border focus:ring-primary')
          }
          {...rest}
        />
        {error ? (
          <p id={errorId} className="text-xs text-danger" role="alert">
            {error}
          </p>
        ) : description ? (
          <p id={descId} className="text-xs text-text-muted">
            {description}
          </p>
        ) : null}
      </div>
    );
  },
);
