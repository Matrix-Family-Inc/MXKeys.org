/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

import { afterEach, describe, expect, it } from 'vitest';
import { cleanup, render } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import { HomePage } from './ui/home-page';

afterEach(cleanup);

describe('HomePage', () => {
  it('renders the landing shell id', () => {
    // HomePage now includes widgets that call TanStack Query hooks;
    // a provider is required for the component tree to mount.
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const { container } = render(
      <QueryClientProvider client={qc}>
        <HomePage />
      </QueryClientProvider>,
    );
    // #home is the scroll-target hub every in-page nav link points at.
    expect(container.querySelector('#home')).not.toBeNull();
  });
});
