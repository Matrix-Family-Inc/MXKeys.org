/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Tue Apr 07 2026 UTC
 * Status: Created
 */

import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import App from './App';
import i18n from './i18n';
import { en } from './locales/en';
import { ru } from './locales/ru';

function collectLeafPaths(source: Record<string, unknown>, prefix = ''): string[] {
  return Object.entries(source).flatMap(([key, value]) => {
    const nextPath = prefix ? `${prefix}.${key}` : key;
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      return collectLeafPaths(value as Record<string, unknown>, nextPath);
    }
    return [nextPath];
  });
}

describe('App', () => {
  beforeEach(async () => {
    window.localStorage.clear();
    window.history.replaceState({}, '', '/');
    await i18n.changeLanguage('en');
  });

  it('renders the landing shell in English', () => {
    render(<App />);

    expect(screen.getByRole('heading', { name: 'MXKeys' })).toBeInTheDocument();
    expect(screen.getAllByText('Federation Trust Infrastructure')).not.toHaveLength(0);
    expect(screen.getAllByRole('link', { name: 'View API' })).not.toHaveLength(0);
    expect(screen.getAllByRole('link', { name: 'MXKeys GitHub repository' })).not.toHaveLength(0);
    expect(screen.getAllByText('Protected operational routes')).not.toHaveLength(0);
    expect(document.documentElement.lang).toBe('en');
  });

  it('renders the landing shell in Russian and updates document language', async () => {
    await i18n.changeLanguage('ru');

    render(<App />);

    expect(screen.getAllByText('Инфраструктура доверия федерации')).not.toHaveLength(0);
    expect(screen.getAllByRole('link', { name: 'Смотреть API' })).not.toHaveLength(0);
    expect(screen.getAllByRole('link', { name: 'Репозиторий MXKeys на GitHub' })).not.toHaveLength(0);
    expect(screen.getAllByText('Защищённые operational routes')).not.toHaveLength(0);
    expect(document.documentElement.lang).toBe('ru');
    expect(window.location.search).toBe('?lang=ru');
  });

  it('keeps locale key structure in sync', () => {
    expect(collectLeafPaths(ru)).toEqual(collectLeafPaths(en));
  });
});
