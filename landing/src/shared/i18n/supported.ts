/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

/**
 * Supported UI languages for the landing. The list is duplicated here
 * rather than imported from the i18next resource bundle so language
 * detection (URL/localStorage/navigator) can run before any translation
 * file has been loaded.
 */
export const supportedLanguages = [
  'ar', 'bn', 'de', 'en', 'es', 'fr', 'he', 'hi', 'id', 'it', 'ja',
  'ko', 'nl', 'pl', 'pt', 'ru', 'th', 'tr', 'uk', 'ur', 'vi', 'zh',
] as const;

export type SupportedLanguage = (typeof supportedLanguages)[number];

/** rtlLanguages need the document direction flipped to rtl. */
export const rtlLanguages = new Set<SupportedLanguage>(['ar', 'he', 'ur']);

export const defaultLanguage: SupportedLanguage = 'en';

export function isSupportedLanguage(candidate: string): candidate is SupportedLanguage {
  return (supportedLanguages as readonly string[]).includes(candidate);
}

/**
 * parseSupportedLanguage accepts a possibly-region-tagged language string
 * ("en-US", "ru_RU", "ZH-cn") and returns the matching supported code or
 * null.
 */
export function parseSupportedLanguage(candidate: string | null | undefined): SupportedLanguage | null {
  if (!candidate) return null;
  const normalized = candidate.split(/[-_]/)[0].toLowerCase();
  return isSupportedLanguage(normalized) ? normalized : null;
}
