/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Updated
 */

import { useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { zodResolver } from '@hookform/resolvers/zod';

import { TextField } from '@/shared/ui/text-field';
import {
  notaryLookupSchema,
  NOTARY_LOOKUP_ERROR_I18N,
  type NotaryLookupInput,
} from '../model/schema';

export interface NotaryLookupFormProps {
  onSubmit: (input: NotaryLookupInput) => void | Promise<void>;
  busy?: boolean;
}

/**
 * Single-field form used as the reference integration of React
 * Hook Form + Zod + i18next in this codebase. Validation runs
 * on submit only so the input never turns red while the visitor
 * is still typing.
 */
export function NotaryLookupForm({ onSubmit, busy = false }: NotaryLookupFormProps) {
  const { t } = useTranslation();
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<NotaryLookupInput>({
    resolver: zodResolver(notaryLookupSchema),
    mode: 'onSubmit',
    reValidateMode: 'onSubmit',
  });

  const rawError = errors.server_name?.message;
  const errorText = rawError
    ? t(NOTARY_LOOKUP_ERROR_I18N[rawError] ?? 'lookup.validation.badShape')
    : undefined;

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-3" noValidate>
      <TextField
        label={t('lookup.field')}
        placeholder={t('lookup.placeholder')}
        autoComplete="off"
        spellCheck={false}
        description={t('lookup.hint')}
        {...register('server_name')}
        error={errorText}
      />
      <button
        type="submit"
        disabled={busy || isSubmitting}
        className="btn btn-primary w-fit"
      >
        {busy || isSubmitting ? t('lookup.submitting') : t('lookup.submit')}
      </button>
    </form>
  );
}
