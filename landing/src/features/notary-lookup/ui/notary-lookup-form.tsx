/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';

import { TextField } from '@/shared/ui/text-field';
import { notaryLookupSchema, type NotaryLookupInput } from '../model/schema';

export interface NotaryLookupFormProps {
  /**
   * onSubmit is invoked only after Zod validation has succeeded. The
   * caller is responsible for the network round trip (and, at the
   * moment, for rendering the result elsewhere on the page).
   */
  onSubmit: (input: NotaryLookupInput) => void | Promise<void>;
  /**
   * busy lets the parent disable the submit button while a request is
   * in flight; the form itself does not track async state to keep
   * this feature framework-agnostic.
   */
  busy?: boolean;
}

/**
 * Accessible, validated single-field form used as the reference
 * integration of React Hook Form + Zod in this codebase. Future
 * forms in the landing page follow the same pattern:
 *
 *   1. Declare the shape in `model/schema.ts` with zod.
 *   2. Wire `useForm({ resolver: zodResolver(schema) })` here.
 *   3. Register inputs via `register('field')` and surface errors via
 *      the TextField shared primitive.
 *
 * No global form state; no useEffect. Errors come from zod.
 */
export function NotaryLookupForm({ onSubmit, busy = false }: NotaryLookupFormProps) {
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<NotaryLookupInput>({
    resolver: zodResolver(notaryLookupSchema),
    mode: 'onBlur',
  });

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-3" noValidate>
      <TextField
        label="Matrix server"
        placeholder="matrix.example.org"
        autoComplete="off"
        spellCheck={false}
        {...register('server_name')}
        error={errors.server_name?.message}
      />
      <button
        type="submit"
        disabled={busy || isSubmitting}
        className="inline-flex w-fit items-center justify-center rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white hover:bg-neutral-800 disabled:cursor-not-allowed disabled:opacity-60"
      >
        {busy || isSubmitting ? 'Checking...' : 'Check notary keys'}
      </button>
    </form>
  );
}
