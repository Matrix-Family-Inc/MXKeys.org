/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

/**
 * cn merges Tailwind class lists with intelligent conflict resolution.
 * Callers pass any clsx-compatible shape (strings, arrays, objects); the
 * result has Tailwind utility collisions deduplicated (later wins), so
 * overrides from component consumers behave as expected without manual
 * ordering.
 */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}
