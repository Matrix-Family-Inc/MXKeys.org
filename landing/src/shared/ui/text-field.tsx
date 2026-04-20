/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import { forwardRef } from 'react';
import type { InputHTMLAttributes } from 'react';

export interface TextFieldProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  error?: string | undefined;
}

/**
 * Controlled, accessible single-line text input intended to be used
 * with React Hook Form's `register()`. Ref forwarding is required by
 * RHF; label/error wiring is handled here so consumers never wire
 * `<label htmlFor>` + `aria-describedby` by hand and get it wrong.
 */
export const TextField = forwardRef<HTMLInputElement, TextFieldProps>(
  function TextField({ label, error, id, ...rest }, ref) {
    const inputId = id ?? `tf-${rest.name ?? 'field'}`;
    const errorId = `${inputId}-err`;
    return (
      <div className="flex flex-col gap-1">
        <label htmlFor={inputId} className="text-sm font-medium">
          {label}
        </label>
        <input
          id={inputId}
          ref={ref}
          aria-invalid={error ? 'true' : 'false'}
          aria-describedby={error ? errorId : undefined}
          className={
            'rounded-md border px-3 py-2 text-sm outline-none focus:ring-2 ' +
            (error ? 'border-red-500 focus:ring-red-500' : 'border-neutral-300 focus:ring-neutral-900')
          }
          {...rest}
        />
        {error ? (
          <p id={errorId} className="text-xs text-red-600" role="alert">
            {error}
          </p>
        ) : null}
      </div>
    );
  },
);
